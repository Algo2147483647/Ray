package ray_tracing

import (
	"math"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

// Integrator selects the light transport algorithm.
type Integrator string

const (
	IntegratorPath Integrator = "path"
	IntegratorBDPT Integrator = "bdpt"
)

type bdptVertex struct {
	Point           *mat.VecDense
	GeometricNormal *mat.VecDense
	Frame           maths.Frame
	WoLocal         maths.Direction
	Context         bxdf.ShadingContext
	Object          *object.Object
	Beta            optics.Spectrum
	PDFFwdArea      float64
	SampledPDF      float64
	SampledDelta    bool
	LightEndpoint   bool
}

type areaLight struct {
	Object  *object.Object
	Sampler shape.SurfaceSampler
	Area    float64
}

// traceBidirectionalSample builds a camera subpath and an independently
// sampled light subpath, then connects every non-delta pair. The adjacent
// strategies are combined with a power heuristic in area measure.
//
// The implementation is deliberately enabled only for Euclidean 3D scenes:
// surface-area measures and the 1/r^2 geometry term are not interchangeable
// with their curved-space counterparts.
func (h *Handler) traceBidirectionalSample(
	renderCamera camera.Camera,
	objTree *object.ObjectTree,
	wavelengthNM, wavelengthPDF float64,
	index ...int,
) optics.Spectrum {
	if geometry.Get(h.SceneGeometry).Kind() != geometry.EuclideanKind {
		return optics.Spectrum{}
	}

	lights, totalArea := collectAreaLights(objTree)
	if len(lights) == 0 || totalArea <= 0 {
		return h.tracePathSampleSpectrum(renderCamera, objTree, wavelengthNM, wavelengthPDF, index...)
	}

	cameraPath, emitted := h.buildCameraSubpath(renderCamera, objTree, wavelengthNM, wavelengthPDF, index...)
	lightPath := h.buildLightSubpath(objTree, lights, totalArea, wavelengthNM, wavelengthPDF)
	result := emitted

	for li := range lightPath {
		for ci := range cameraPath {
			// A path with two surface endpoints and the camera must still obey
			// the configured maximum number of scattering vertices.
			if int64(li+ci+1) > h.MaxRayLevel {
				continue
			}
			contribution, ok := h.connectBDPTVertices(objTree, lightPath, cameraPath, li, ci)
			if ok {
				result = result.Add(contribution)
			}
		}
	}
	return result
}

func (h *Handler) tracePathSampleSpectrum(
	renderCamera camera.Camera,
	objTree *object.ObjectTree,
	wavelengthNM, wavelengthPDF float64,
	index ...int,
) optics.Spectrum {
	ray := h.RayPool.Get().(*optics.Ray)
	ray.Geometry = h.SceneGeometry
	defer h.RayPool.Put(ray)
	renderCamera.GenerateRay(ray, index...)
	if wavelengthNM > 0 {
		ray.SetSpectralSample(optics.WavelengthSample{LambdaNM: wavelengthNM, PDF: wavelengthPDF})
	} else {
		ray.DisableSpectralSampling()
	}
	h.TraceRay(objTree, ray, 0)
	if wavelengthNM > 0 {
		return optics.NewSampledSpectrum([]float64{optics.SpectralRayToScalar(ray)})
	}
	return optics.NewRGBSpectrum(ray.Color[0], ray.Color[1], ray.Color[2])
}

func collectAreaLights(tree *object.ObjectTree) ([]areaLight, float64) {
	if tree == nil {
		return nil, 0
	}
	lights := make([]areaLight, 0)
	totalArea := 0.0
	for _, obj := range tree.Objects {
		if obj == nil || obj.Material == nil || !obj.Material.HasEmission() || obj.Shape == nil {
			continue
		}
		sampler, ok := obj.Shape.(shape.SurfaceSampler)
		if !ok {
			continue
		}
		area := sampler.SurfaceArea()
		if area <= 0 || math.IsNaN(area) || math.IsInf(area, 0) {
			continue
		}
		lights = append(lights, areaLight{Object: obj, Sampler: sampler, Area: area})
		totalArea += area
	}
	return lights, totalArea
}

func (h *Handler) buildCameraSubpath(
	renderCamera camera.Camera,
	tree *object.ObjectTree,
	wavelengthNM, wavelengthPDF float64,
	index ...int,
) ([]bdptVertex, optics.Spectrum) {
	ray := &optics.Ray{Geometry: h.SceneGeometry}
	renderCamera.GenerateRay(ray, index...)
	setBDPTWavelength(ray, wavelengthNM, wavelengthPDF)

	beta := unitSpectrum(wavelengthNM)
	emitted := zeroSpectrum(wavelengthNM)
	path := make([]bdptVertex, 0, h.MaxRayLevel+1)
	pendingPDF := 1.0
	previousDelta := true

	for depth := int64(0); depth <= h.MaxRayLevel; depth++ {
		origin := mat.VecDenseCopyOf(ray.Origin)
		direction := mat.VecDenseCopyOf(ray.Direction)
		hit, ok := surfaceHitInGeometry(tree, ray, ray.G())
		if !ok {
			break
		}
		distance2 := squaredDistance(origin, hit.Point)
		pdfArea := pendingPDF * absDot(hit.GeometricNormal, negated(direction)) / math.Max(distance2, utils.EPS)

		si, ok := h.prepareSurfaceInteraction(getMediumRegistry(tree), ray, hit)
		if !ok {
			break
		}
		vertex := bdptVertex{
			Point: hit.Point, GeometricNormal: hit.GeometricNormal, Frame: si.Frame,
			WoLocal: si.WoLocal, Context: si.Context, Object: si.Object,
			Beta: beta, PDFFwdArea: pdfArea,
		}

		if si.Object.Material.HasEmission() && (depth == 0 || previousDelta || !isSampleableAreaLight(si.Object)) {
			le := si.Object.Material.Emission.Emit(si.Context, si.WoLocal)
			emitted = emitted.Add(beta.Mul(le))
		}
		if !si.Object.Material.HasSurface() {
			path = append(path, vertex)
			break
		}

		sample, ok := sampleSurface(si.Object, si.Context, si.WoLocal)
		if !ok {
			path = append(path, vertex)
			break
		}
		vertex.SampledPDF = sample.PDF
		vertex.SampledDelta = sample.Flags&(bxdf.DeltaReflection|bxdf.DeltaTransmission) != 0

		beta = beta.Mul(sample.F).MulScalar(maths.AbsCosTheta(sample.Wi) / sample.PDF)
		if !validSpectrum(beta) {
			path = append(path, vertex)
			break
		}
		survival := bdptSurvivalProbability(beta, depth+1, h.RussianRouletteDepth)
		if survival <= 0 || rand.Float64() >= survival {
			path = append(path, vertex)
			break
		}
		beta = beta.DivScalar(survival)
		vertex.SampledPDF *= survival
		path = append(path, vertex)
		if sample.Flags&bxdf.TransmissionEvent != 0 {
			applyMediumTransmission(getMediumRegistry(tree), ray, si.Context, si.Object.MediumBoundary, sample)
		}
		si.Frame.LocalToWorldInto(ray.Direction, sample.Wi)
		ray.Origin.CopyVec(hit.Point)
		pendingPDF = vertex.SampledPDF
		previousDelta = vertex.SampledDelta
	}
	return path, emitted
}

func (h *Handler) buildLightSubpath(
	tree *object.ObjectTree,
	lights []areaLight,
	totalArea, wavelengthNM, wavelengthPDF float64,
) []bdptVertex {
	target := rand.Float64() * totalArea
	selected := lights[len(lights)-1]
	for _, light := range lights {
		if target < light.Area {
			selected = light
			break
		}
		target -= light.Area
	}
	ss, ok := selected.Sampler.SampleSurface(maths.Sample2D{U: rand.Float64(), V: rand.Float64()})
	if !ok {
		return nil
	}
	selectionPDF := selected.Area / totalArea
	pdfLightArea := selectionPDF * ss.PDFArea
	if pdfLightArea <= 0 || math.IsNaN(pdfLightArea) || math.IsInf(pdfLightArea, 0) {
		return nil
	}
	ctx := bxdf.ShadingContext{
		TransportMode: bxdf.TransportImportance, SpectrumMode: h.SpectrumMode,
		WavelengthNM: wavelengthNM, WavelengthPDF: wavelengthPDF,
		HitPoint:        maths.NewDirectionFromComponents(ss.Point.RawVector().Data),
		GeometricNormal: maths.NewDirectionFromComponents(ss.Normal.RawVector().Data),
		UV:              ss.UV, CurrentIOR: 1,
	}
	if wavelengthNM > 0 {
		ctx.WavelengthsNM = []float64{wavelengthNM}
	}

	normal := mat.VecDenseCopyOf(ss.Normal)
	if rand.Float64() < 0.5 {
		normal.ScaleVec(-1, normal)
	}
	frame, ok := maths.NewFrameFromNormal(normal)
	if !ok {
		return nil
	}
	localDirection := maths.CosineSampleHemisphere(maths.Sample2D{U: rand.Float64(), V: rand.Float64()})
	pdfDirection := 0.5 * maths.CosineHemispherePDF(localDirection)
	if pdfDirection <= 0 {
		return nil
	}
	worldDirection := frame.LocalToWorld(localDirection)
	emitted := selected.Object.Material.Emission.Emit(ctx, localDirection)
	beta := emitted.MulScalar(maths.AbsCosTheta(localDirection) / (pdfLightArea * pdfDirection))

	root := bdptVertex{
		Point: ss.Point, GeometricNormal: ss.Normal, Frame: frame,
		Context: ctx, Object: selected.Object, Beta: unitSpectrum(wavelengthNM).MulScalar(1 / pdfLightArea),
		PDFFwdArea: pdfLightArea, SampledPDF: pdfDirection, LightEndpoint: true,
	}
	path := []bdptVertex{root}
	ray := &optics.Ray{Origin: mat.VecDenseCopyOf(ss.Point), Direction: worldDirection, Geometry: h.SceneGeometry}
	ray.Init()
	ray.Origin.CopyVec(ss.Point)
	ray.Direction.CopyVec(worldDirection)
	setBDPTWavelength(ray, wavelengthNM, wavelengthPDF)
	pendingPDF := pdfDirection

	for depth := int64(1); depth <= h.MaxRayLevel; depth++ {
		origin := mat.VecDenseCopyOf(ray.Origin)
		direction := mat.VecDenseCopyOf(ray.Direction)
		hit, ok := surfaceHitInGeometry(tree, ray, ray.G())
		if !ok {
			break
		}
		distance2 := squaredDistance(origin, hit.Point)
		pdfArea := pendingPDF * absDot(hit.GeometricNormal, negated(direction)) / math.Max(distance2, utils.EPS)
		si, ok := h.prepareSurfaceInteraction(getMediumRegistry(tree), ray, hit)
		if !ok {
			break
		}
		si.Context.TransportMode = bxdf.TransportImportance
		vertex := bdptVertex{
			Point: hit.Point, GeometricNormal: hit.GeometricNormal, Frame: si.Frame,
			WoLocal: si.WoLocal, Context: si.Context, Object: si.Object,
			Beta: beta, PDFFwdArea: pdfArea,
		}
		if !si.Object.Material.HasSurface() {
			path = append(path, vertex)
			break
		}
		sample, ok := sampleSurface(si.Object, si.Context, si.WoLocal)
		if !ok {
			path = append(path, vertex)
			break
		}
		vertex.SampledPDF = sample.PDF
		vertex.SampledDelta = sample.Flags&(bxdf.DeltaReflection|bxdf.DeltaTransmission) != 0
		beta = beta.Mul(sample.F).MulScalar(maths.AbsCosTheta(sample.Wi) / sample.PDF)
		if !validSpectrum(beta) {
			path = append(path, vertex)
			break
		}
		survival := bdptSurvivalProbability(beta, depth+1, h.RussianRouletteDepth)
		if survival <= 0 || rand.Float64() >= survival {
			path = append(path, vertex)
			break
		}
		beta = beta.DivScalar(survival)
		vertex.SampledPDF *= survival
		path = append(path, vertex)
		if sample.Flags&bxdf.TransmissionEvent != 0 {
			applyMediumTransmission(getMediumRegistry(tree), ray, si.Context, si.Object.MediumBoundary, sample)
		}
		si.Frame.LocalToWorldInto(ray.Direction, sample.Wi)
		ray.Origin.CopyVec(hit.Point)
		pendingPDF = vertex.SampledPDF
	}
	return path
}

func (h *Handler) connectBDPTVertices(
	tree *object.ObjectTree,
	lightPath, cameraPath []bdptVertex,
	li, ci int,
) (optics.Spectrum, bool) {
	lv, cv := &lightPath[li], &cameraPath[ci]
	if cv.Object == nil || cv.Object.Material == nil || !cv.Object.Material.HasSurface() {
		return optics.Spectrum{}, false
	}
	toCamera := mat.NewVecDense(lv.Point.Len(), nil)
	toCamera.SubVec(cv.Point, lv.Point)
	distance2 := mat.Dot(toCamera, toCamera)
	if distance2 <= utils.EPS*utils.EPS {
		return optics.Spectrum{}, false
	}
	distance := math.Sqrt(distance2)
	toCamera.ScaleVec(1/distance, toCamera)
	if !visibleSegment(tree, lv.Point, cv.Point, toCamera, distance) {
		return optics.Spectrum{}, false
	}

	toLight := negated(toCamera)
	wiCamera := cv.Frame.WorldToLocal(toLight)
	fCamera := cv.Object.Material.Surface.Eval(cv.Context, wiCamera, cv.WoLocal)
	if fCamera.IsZero() {
		return optics.Spectrum{}, false
	}
	cosLight := absDot(lv.GeometricNormal, toCamera)
	cosCamera := absDot(cv.GeometricNormal, toLight)
	if cosLight <= 0 || cosCamera <= 0 {
		return optics.Spectrum{}, false
	}
	geometryTerm := cosLight * cosCamera / distance2

	var lightFactor optics.Spectrum
	if lv.LightEndpoint {
		woLight := lv.Frame.WorldToLocal(toCamera)
		lightFactor = lv.Object.Material.Emission.Emit(lv.Context, woLight).Mul(lv.Beta)
	} else {
		if lv.Object == nil || lv.Object.Material == nil || !lv.Object.Material.HasSurface() {
			return optics.Spectrum{}, false
		}
		wiLight := lv.Frame.WorldToLocal(toCamera)
		fLight := lv.Object.Material.Surface.Eval(lv.Context, wiLight, lv.WoLocal)
		if fLight.IsZero() {
			return optics.Spectrum{}, false
		}
		lightFactor = lv.Beta.Mul(fLight)
	}

	weight := adjacentPowerMIS(lv, cv, li, ci, toCamera, toLight, cosLight, cosCamera, distance2)
	contribution := lightFactor.Mul(cv.Beta).Mul(fCamera).MulScalar(geometryTerm * weight)
	return contribution, validSpectrum(contribution)
}

func adjacentPowerMIS(
	lv, cv *bdptVertex,
	li, ci int,
	toCamera, toLight *mat.VecDense,
	cosLight, cosCamera, distance2 float64,
) float64 {
	sum := 1.0
	if cv.Object.Material.HasSurface() && lv.PDFFwdArea > 0 {
		pdf := cv.Object.Material.Surface.PDF(cv.Context, cv.Frame.WorldToLocal(toLight), cv.WoLocal)
		pdfArea := pdf * cosLight / distance2
		ratio := pdfArea / lv.PDFFwdArea
		sum += ratio * ratio
	}
	// t=1 (light tracing splatted directly to this pixel) is not implemented,
	// so it must not participate in MIS at the first camera vertex.
	if ci > 0 && cv.PDFFwdArea > 0 {
		var pdf float64
		if lv.LightEndpoint {
			pdf = 0.5 * cosLight / math.Pi
		} else if lv.Object.Material.HasSurface() {
			pdf = lv.Object.Material.Surface.PDF(lv.Context, lv.Frame.WorldToLocal(toCamera), lv.WoLocal)
		}
		pdfArea := pdf * cosCamera / distance2
		ratio := pdfArea / cv.PDFFwdArea
		sum += ratio * ratio
	}
	return 1 / sum
}

func visibleSegment(tree *object.ObjectTree, from, to, direction *mat.VecDense, distance float64) bool {
	if tree == nil || tree.Root == nil {
		return true
	}
	_, ok := tree.GetSurfaceHitRange(from, direction, utils.EPS, distance-utils.EPS)
	return !ok
}

func isSampleableAreaLight(obj *object.Object) bool {
	if obj == nil || obj.Shape == nil {
		return false
	}
	sampler, ok := obj.Shape.(shape.SurfaceSampler)
	return ok && sampler.SurfaceArea() > 0
}

func setBDPTWavelength(ray *optics.Ray, wavelengthNM, wavelengthPDF float64) {
	if wavelengthNM > 0 {
		ray.SetSpectralSample(optics.WavelengthSample{LambdaNM: wavelengthNM, PDF: wavelengthPDF})
	} else {
		ray.DisableSpectralSampling()
	}
}

func unitSpectrum(wavelengthNM float64) optics.Spectrum {
	if wavelengthNM > 0 {
		return optics.NewSampledSpectrum([]float64{1})
	}
	return optics.ConstantSpectrum(1)
}

func zeroSpectrum(wavelengthNM float64) optics.Spectrum {
	if wavelengthNM > 0 {
		return optics.NewSampledSpectrum([]float64{0})
	}
	return optics.Spectrum{}
}

func validSpectrum(s optics.Spectrum) bool {
	return s.IsFinite() && s.IsNonNegative() && !s.IsZero()
}

func bdptSurvivalProbability(beta optics.Spectrum, nextDepth, configuredDepth int64) float64 {
	depth := configuredDepth
	if depth <= 0 {
		depth = 3
	}
	if nextDepth < depth {
		return 1
	}
	return math.Min(0.95, math.Max(minRussianRouletteSurvival, beta.MaxComponent()))
}

func squaredDistance(a, b *mat.VecDense) float64 {
	d := mat.NewVecDense(a.Len(), nil)
	d.SubVec(a, b)
	return mat.Dot(d, d)
}

func negated(v *mat.VecDense) *mat.VecDense {
	result := mat.VecDenseCopyOf(v)
	result.ScaleVec(-1, result)
	return result
}

func absDot(a, b *mat.VecDense) float64 {
	if a == nil || b == nil {
		return 0
	}
	return math.Abs(mat.Dot(a, b))
}
