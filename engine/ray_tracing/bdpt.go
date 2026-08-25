package ray_tracing

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/emission"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type bdptVertexKind uint8

const (
	bdptVertexCamera bdptVertexKind = iota
	bdptVertexLight
	bdptVertexSurface
)

// bdptVertex stores densities in area measure at this vertex. Delta describes
// the sampled outgoing edge; Connectible describes whether the vertex has a
// continuous component that a deterministic connection strategy may evaluate.
type bdptVertex struct {
	Kind            bdptVertexKind
	Point           *mat.VecDense
	GeometricNormal *mat.VecDense
	Frame           maths.Frame
	EmissionFrame   maths.Frame
	WoLocal         maths.Direction
	Context         bxdf.ShadingContext
	Object          *object.Object
	Beta            optics.Spectrum
	PDFFwdArea      float64
	PDFRevArea      float64
	SampledPDF      float64
	SampledDelta    bool
	Connectible     bool
	MediumStack     medium.Stack
	Camera          camera.BidirectionalCamera
}

type areaLight struct {
	Object  *object.Object
	Sampler shape.SurfaceSampler
	Area    float64
	Weight  float64
}

type bdptSceneState struct {
	Lights           []areaLight
	TotalLightWeight float64
}

func (v *bdptVertex) emissionLocal(world *mat.VecDense) maths.Direction {
	if v == nil {
		return nil
	}
	if v.EmissionFrame.Normal != nil {
		return v.EmissionFrame.WorldToLocal(world)
	}
	// Compatibility for manually assembled vertices in tests and external
	// callers created before EmissionFrame was introduced.
	return v.Frame.WorldToLocal(world)
}

// prepareBDPT performs all capability validation before an integrator driver
// or worker is started. Infinite endpoint measures and non-reciprocal
// transmission remain explicit capability errors.
func (h *Handler) prepareBDPT(renderCamera camera.RayCamera, film *camera.Film, tree *object.ObjectTree) (*bdptSceneState, error) {
	if h == nil {
		return nil, fmt.Errorf("BDPT handler is nil")
	}
	if h.Space.G().Kind() != geometry.EuclideanKind {
		return nil, fmt.Errorf("BDPT requires three-dimensional Euclidean geometry")
	}
	if renderCamera == nil || film == nil {
		return nil, fmt.Errorf("BDPT camera or Film is nil")
	}
	if len(film.Shape) != 2 || film.Shape[0] <= 0 || film.Shape[1] <= 0 {
		return nil, fmt.Errorf("BDPT requires a non-empty 2D Film")
	}
	if !film.HasSpectralBins() {
		return nil, fmt.Errorf("BDPT Film spectral bins are not initialized")
	}
	if _, err := camera.NormalizePixelWindows(film.PixelWindows, film.Shape); err != nil {
		return nil, fmt.Errorf("BDPT pixel windows: %w", err)
	}
	if _, ok := renderCamera.(camera.BidirectionalCamera); !ok {
		return nil, fmt.Errorf("BDPT camera %T does not provide endpoint sampling densities", renderCamera)
	}
	endpoint := renderCamera.(camera.BidirectionalCamera).Endpoint()
	if endpoint == nil || endpoint.Len() != 3 {
		return nil, fmt.Errorf("BDPT currently requires a perspective camera with a finite endpoint")
	}

	if tree != nil {
		for _, obj := range tree.Objects {
			if obj == nil {
				continue
			}
			if obj.Material == nil {
				if obj.MediumBoundary.Active() {
					return nil, fmt.Errorf("BDPT medium boundary requires a material surface")
				}
				continue
			}
			if obj.Material.Metadata.NonReciprocal {
				return nil, fmt.Errorf("BDPT does not support physically non-reciprocal materials")
			}
			if obj.Material.HasSurface() {
				if err := validateBDPTSurface(obj.Material.Surface); err != nil {
					return nil, fmt.Errorf("object %q: %w", obj.Material.Metadata.Name, err)
				}
				if obj.MediumBoundary.Active() && obj.Material.Surface.DeltaFlags()&bxdf.TransmissionEvent == 0 {
					return nil, fmt.Errorf("object %q: BDPT medium boundary requires a supported transmission surface", obj.Material.Metadata.Name)
				}
				if obj.MediumBoundary.Active() && obj.MediumBoundary.Thin {
					return nil, fmt.Errorf("object %q: BDPT thin medium boundaries are not implemented", obj.Material.Metadata.Name)
				}
			} else if obj.MediumBoundary.Active() {
				return nil, fmt.Errorf("object %q: BDPT medium boundary requires a material surface", obj.Material.Metadata.Name)
			}
			if obj.Material.HasEmission() {
				if obj.Material.Emission.DirectionFlags()&emission.DirectionDelta != 0 {
					return nil, fmt.Errorf("BDPT delta emitters are not implemented")
				}
				if !isSampleableAreaLight(obj) {
					return nil, fmt.Errorf("BDPT environment or non-sampleable emitter %T is not implemented", obj.Shape)
				}
			}
		}
	}

	state := prepareBDPTScene(tree)
	if len(state.Lights) == 0 || state.TotalLightWeight <= 0 {
		return nil, fmt.Errorf("BDPT scene has no sampleable finite area light")
	}
	return state, nil
}

func validateBDPTSurface(surface bxdf.Scattering) error {
	if surface == nil {
		return nil
	}
	flags := surface.DeltaFlags()
	if flags&bxdf.NonReciprocal != 0 {
		return fmt.Errorf("surface is not bidirectionally evaluable")
	}
	if !isBDPTSurface(surface) {
		return fmt.Errorf("BDPT currently supports reciprocal reflection and ideal dielectric transmission, got %T", surface)
	}
	return nil
}

func isBDPTSurface(surface bxdf.Scattering) bool {
	return isBDPTScattering(surface)
}

func isBDPTMixture(mixture bsdf.WeightedMixture) bool {
	hasComponent := false
	for _, component := range mixture.Components {
		if component.Scattering == nil || component.Weight <= 0 {
			continue
		}
		hasComponent = true
		if !isBDPTScattering(component.Scattering) {
			return false
		}
	}
	return hasComponent
}

func isBDPTScattering(scattering bxdf.Scattering) bool {
	switch value := scattering.(type) {
	case bsdf.WeightedMixture:
		return isBDPTMixture(value)
	case *bsdf.WeightedMixture:
		return value != nil && isBDPTMixture(*value)
	case bxdf.Lambert, *bxdf.Lambert,
		bxdf.SpecularReflection, *bxdf.SpecularReflection,
		bxdf.SpecularDielectric, *bxdf.SpecularDielectric,
		bxdf.RoughConductor, *bxdf.RoughConductor,
		bxdf.RoughDielectricReflection, *bxdf.RoughDielectricReflection:
		return true
	default:
		return false
	}
}

func prepareBDPTScene(tree *object.ObjectTree) *bdptSceneState {
	lights, total := collectAreaLights(tree)
	return &bdptSceneState{Lights: lights, TotalLightWeight: total}
}

func (h *Handler) traceBidirectionalPrepared(
	state *bdptSceneState,
	renderCamera camera.RayCamera,
	filmShape []int,
	objTree *object.ObjectTree,
	wavelengthNM, wavelengthPDF float64,
	index ...int,
) (optics.Spectrum, []FilmSplat) {
	bdCamera, ok := renderCamera.(camera.BidirectionalCamera)
	if state == nil || !ok {
		return zeroSpectrum(wavelengthNM), nil
	}
	cameraPath := h.buildCameraSubpath(renderCamera, filmShape, objTree, wavelengthNM, wavelengthPDF, index...)
	lightPath := h.buildLightSubpath(objTree, state.Lights, state.TotalLightWeight, wavelengthNM, wavelengthPDF)
	result := zeroSpectrum(wavelengthNM)
	splats := make([]FilmSplat, 0, len(lightPath))

	for t := 1; t <= len(cameraPath); t++ {
		for s := 0; s <= len(lightPath); s++ {
			depth := s + t - 2
			if (s == 1 && t == 1) || depth < 0 || int64(depth) > h.MaxRayLevel {
				continue
			}
			value, projection, isSplat, valid := h.connectBDPTStrategy(
				state, bdCamera, filmShape, objTree, lightPath, cameraPath, s, t,
			)
			if !valid {
				continue
			}
			weight := bdptMISWeight(state, bdCamera, lightPath, cameraPath, s, t)
			if weight <= 0 {
				continue
			}
			value = value.MulScalar(weight)
			if isSplat {
				splats = append(splats, FilmSplat{
					WavelengthNM: wavelengthNM, WavelengthPDF: wavelengthPDF,
					Value: value, projection: projection,
				})
			} else {
				result = result.Add(value)
			}
		}
	}
	return result, splats
}

func collectAreaLights(tree *object.ObjectTree) ([]areaLight, float64) {
	if tree == nil {
		return nil, 0
	}
	lights := make([]areaLight, 0)
	totalWeight := 0.0
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
		exitance := obj.Material.Emission.ExitanceEstimate(bxdf.ShadingContext{
			SpectrumMode:    optics.SpectrumModeRGB,
			GeometricNormal: maths.NewDirection(0, 0, 1),
		})
		powerEstimate := exitance.MaxComponent()
		if powerEstimate <= 0 || math.IsNaN(powerEstimate) || math.IsInf(powerEstimate, 0) {
			powerEstimate = 1
		}
		weight := area * powerEstimate
		lights = append(lights, areaLight{Object: obj, Sampler: sampler, Area: area, Weight: weight})
		totalWeight += weight
	}
	return lights, totalWeight
}

func (h *Handler) buildCameraSubpath(
	renderCamera camera.RayCamera,
	filmShape []int,
	tree *object.ObjectTree,
	wavelengthNM, wavelengthPDF float64,
	index ...int,
) []bdptVertex {
	bdCamera, ok := renderCamera.(camera.BidirectionalCamera)
	if !ok || bdCamera.Endpoint() == nil {
		return nil
	}
	ray := &optics.Ray{Space: h.Space}
	renderCamera.GenerateRay(ray, filmShape, index...)
	setBDPTWavelength(ray, wavelengthNM, wavelengthPDF)
	directionPDF := bdCamera.PDFDirection(ray.Direction)
	if directionPDF <= 0 {
		return nil
	}
	path := []bdptVertex{{
		Kind: bdptVertexCamera, Point: bdCamera.Endpoint(), Beta: unitSpectrum(wavelengthNM),
		PDFFwdArea: 1, Connectible: true, Camera: bdCamera,
		MediumStack: medium.NewStack(medium.MediumAir),
	}}
	return h.randomWalk(
		tree, ray, unitSpectrum(wavelengthNM), directionPDF,
		bxdf.TransportRadiance, int(h.MaxRayLevel)+2, path,
	)
}

func (h *Handler) sampleLightEndpoint(
	lights []areaLight,
	totalWeight, wavelengthNM, wavelengthPDF float64,
) (bdptVertex, bool) {
	selected, selectionPDF, ok := selectAreaLight(lights, totalWeight)
	if !ok {
		return bdptVertex{}, false
	}
	ss, ok := selected.Sampler.SampleSurface(maths.Sample2D{U: rand.Float64(), V: rand.Float64()})
	if !ok {
		return bdptVertex{}, false
	}
	return h.makeLightEndpoint(selected, selectionPDF, ss, wavelengthNM, wavelengthPDF)
}

func selectAreaLight(lights []areaLight, totalWeight float64) (areaLight, float64, bool) {
	if len(lights) == 0 || totalWeight <= 0 || math.IsNaN(totalWeight) || math.IsInf(totalWeight, 0) {
		return areaLight{}, 0, false
	}
	target := rand.Float64() * totalWeight
	selected := lights[len(lights)-1]
	for _, light := range lights {
		if target < light.Weight {
			selected = light
			break
		}
		target -= light.Weight
	}
	selectionPDF := selected.Weight / totalWeight
	return selected, selectionPDF, selectionPDF > 0 && isFinitePDF(selectionPDF)
}

func (h *Handler) makeLightEndpoint(
	selected areaLight,
	selectionPDF float64,
	ss shape.SurfaceSample,
	wavelengthNM, wavelengthPDF float64,
) (bdptVertex, bool) {
	pdfLightArea := selectionPDF * ss.PDFArea
	if pdfLightArea <= 0 || !isFinitePDF(pdfLightArea) {
		return bdptVertex{}, false
	}
	ctx := bxdf.ShadingContext{
		TransportMode: bxdf.TransportImportance, SpectrumMode: optics.SpectrumModeSpectral,
		WavelengthNM: wavelengthNM, WavelengthPDF: wavelengthPDF,
		HitPoint:        maths.NewDirectionFromComponents(ss.Point.RawVector().Data),
		GeometricNormal: maths.NewDirectionFromComponents(ss.Normal.RawVector().Data),
		UV:              ss.UV,
	}
	if wavelengthNM > 0 {
		ctx.WavelengthsNM = []float64{wavelengthNM}
	}
	frame, ok := maths.NewFrameFromNormal(ss.Normal)
	if !ok {
		return bdptVertex{}, false
	}
	return bdptVertex{
		Kind: bdptVertexLight, Point: ss.Point, GeometricNormal: ss.Normal,
		Frame: frame, EmissionFrame: frame,
		Context: ctx, Object: selected.Object,
		Beta:        unitSpectrum(wavelengthNM).MulScalar(1 / pdfLightArea),
		PDFFwdArea:  pdfLightArea,
		Connectible: selected.Object.Material.Emission.DirectionFlags()&emission.DirectionDelta == 0,
		MediumStack: medium.NewStack(medium.MediumAir),
	}, true
}

func (h *Handler) buildLightSubpath(
	tree *object.ObjectTree,
	lights []areaLight,
	totalWeight, wavelengthNM, wavelengthPDF float64,
) []bdptVertex {
	root, ok := h.sampleLightEndpoint(lights, totalWeight, wavelengthNM, wavelengthPDF)
	if !ok {
		return nil
	}
	directionSample := root.Object.Material.Emission.SampleDirection(root.Context, maths.Sample2D{
		U: rand.Float64(), V: rand.Float64(),
	})
	if directionSample.PDF <= 0 || directionSample.Wo.Len() != root.GeometricNormal.Len() ||
		!directionSample.Le.IsFinite() || !directionSample.Le.IsNonNegative() {
		return nil
	}
	worldDirection := root.EmissionFrame.LocalToWorld(directionSample.Wo)
	beta := root.Beta.Mul(directionSample.Le).MulScalar(
		maths.AbsCosTheta(directionSample.Wo) / directionSample.PDF,
	)
	root.SampledPDF = directionSample.PDF
	root.SampledDelta = directionSample.Flags&emission.DirectionDelta != 0
	path := []bdptVertex{root}
	ray := &optics.Ray{
		Origin: mat.VecDenseCopyOf(root.Point), Direction: mat.VecDenseCopyOf(worldDirection),
		Space: h.Space,
	}
	ray.Init()
	ray.Origin.CopyVec(root.Point)
	ray.Direction.CopyVec(worldDirection)
	setBDPTWavelength(ray, wavelengthNM, wavelengthPDF)
	return h.randomWalk(
		tree, ray, beta, directionSample.PDF,
		bxdf.TransportImportance, int(h.MaxRayLevel)+1, path,
	)
}

// randomWalk is shared by camera and light subpaths. It records the forward
// area density of every generated vertex and the reverse area density of the
// preceding vertex at the moment the outgoing edge is sampled.
func (h *Handler) randomWalk(
	tree *object.ObjectTree,
	ray *optics.Ray,
	beta optics.Spectrum,
	pendingDirectionPDF float64,
	mode bxdf.TransportMode,
	maxVertices int,
	path []bdptVertex,
) []bdptVertex {
	for len(path) < maxVertices {
		origin := mat.VecDenseCopyOf(ray.Origin)
		direction := mat.VecDenseCopyOf(ray.Direction)
		hit, ok := surfaceHitInGeometry(tree, ray, ray.G())
		if !ok {
			break
		}
		segmentLength := hit.ArcLength
		if segmentLength <= 0 {
			segmentLength = ray.G().ArcLengthFromEmbedT(ray.Origin, ray.Direction, hit.Distance)
		}
		beta = evaluateSegmentTransmittance(
			getMediumRegistry(tree), ray.MediumStack.Current(), segmentLength, h.newShadingContext(ray),
		).ApplyToSpectrum(beta)
		if !validSpectrum(beta) {
			break
		}
		distance2 := squaredDistance(origin, hit.Point)
		pdfArea := pendingDirectionPDF * absDot(hit.GeometricNormal, negated(direction)) / math.Max(distance2, utils.EPS)
		si, ok := h.prepareSurfaceInteraction(getMediumRegistry(tree), ray, hit)
		if !ok {
			break
		}
		si.Context.TransportMode = mode
		vertex := bdptVertex{
			Kind: bdptVertexSurface, Point: hit.Point, GeometricNormal: hit.GeometricNormal,
			Frame: si.Frame, EmissionFrame: si.EmissionFrame,
			WoLocal: si.WoLocal, Context: si.Context, Object: si.Object,
			Beta: beta, PDFFwdArea: pdfArea, MediumStack: ray.MediumStack.Clone(),
		}
		if si.Object.Material != nil && si.Object.Material.HasSurface() {
			vertex.Connectible = !si.Object.Material.Surface.RoughnessInfo(si.Context).IsDelta
		}
		path = append(path, vertex)
		currentIndex := len(path) - 1
		if si.Object.Material == nil || !si.Object.Material.HasSurface() {
			break
		}
		sample, ok := sampleSurface(si.Object, si.Context, si.WoLocal)
		if !ok {
			break
		}
		path[currentIndex].SampledPDF = sample.PDF
		path[currentIndex].SampledDelta = sample.Flags&(bxdf.DeltaReflection|bxdf.DeltaTransmission) != 0

		reverseContext := si.Context
		if mode == bxdf.TransportRadiance {
			reverseContext.TransportMode = bxdf.TransportImportance
		} else {
			reverseContext.TransportMode = bxdf.TransportRadiance
		}
		forwardPDF := sample.PDF
		reversePDF := si.Object.Material.Surface.PDF(reverseContext, si.WoLocal, sample.Wi)
		if path[currentIndex].SampledDelta {
			forwardPDF = 0
			reversePDF = 0
		}
		path[currentIndex-1].PDFRevArea = convertBDPTDensity(reversePDF, &path[currentIndex], &path[currentIndex-1])

		beta = beta.Mul(sample.F).MulScalar(maths.AbsCosTheta(sample.Wi) / sample.PDF)
		if !validSpectrum(beta) {
			break
		}
		if sample.Flags&bxdf.TransmissionEvent != 0 {
			applyMediumTransmission(ray, si.Context, si.Object.MediumBoundary, sample)
		}
		si.Frame.LocalToWorldInto(ray.Direction, sample.Wi)
		ray.Origin.CopyVec(hit.Point)
		pendingDirectionPDF = forwardPDF
	}
	return path
}

func (h *Handler) connectBDPTStrategy(
	state *bdptSceneState,
	renderCamera camera.BidirectionalCamera,
	filmShape []int,
	tree *object.ObjectTree,
	lightPath, cameraPath []bdptVertex,
	s, t int,
) (optics.Spectrum, camera.FilmProjection, bool, bool) {
	if t <= 0 || t > len(cameraPath) || s < 0 || s > len(lightPath) {
		return optics.Spectrum{}, camera.FilmProjection{}, false, false
	}
	if s == 0 {
		pt := &cameraPath[t-1]
		if pt.Kind != bdptVertexSurface || pt.Object == nil || pt.Object.Material == nil || !pt.Object.Material.HasEmission() {
			return optics.Spectrum{}, camera.FilmProjection{}, false, false
		}
		if t < 2 {
			return optics.Spectrum{}, camera.FilmProjection{}, false, false
		}
		previous := &cameraPath[t-2]
		toPrevious := directionBetween(pt.Point, previous.Point)
		if toPrevious == nil {
			return optics.Spectrum{}, camera.FilmProjection{}, false, false
		}
		wo := pt.emissionLocal(toPrevious)
		value := pt.Beta.Mul(pt.Object.Material.Emission.Eval(pt.Context, wo))
		return value, camera.FilmProjection{}, false, validSpectrum(value)
	}
	if t == 1 {
		if s < 2 {
			return optics.Spectrum{}, camera.FilmProjection{}, true, false
		}
		value, projection, ok := projectLightVertex(renderCamera, filmShape, tree, &lightPath[s-1])
		return value, projection, true, ok
	}
	value, ok := h.connectBDPTVertices(tree, &lightPath[s-1], &cameraPath[t-1])
	return value, camera.FilmProjection{}, false, ok
}

func (h *Handler) connectBDPTVertices(tree *object.ObjectTree, lv, cv *bdptVertex) (optics.Spectrum, bool) {
	if lv == nil || cv == nil || cv.Kind != bdptVertexSurface || !cv.Connectible ||
		cv.Object == nil || cv.Object.Material == nil || !cv.Object.Material.HasSurface() {
		return optics.Spectrum{}, false
	}
	toCamera := directionBetween(lv.Point, cv.Point)
	if toCamera == nil {
		return optics.Spectrum{}, false
	}
	distance2 := squaredDistance(lv.Point, cv.Point)
	distance := math.Sqrt(distance2)
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
	if lv.Kind == bdptVertexLight {
		woLight := lv.emissionLocal(toCamera)
		lightFactor = lv.Object.Material.Emission.Eval(lv.Context, woLight).Mul(lv.Beta)
	} else {
		if lv.Kind != bdptVertexSurface || !lv.Connectible || lv.Object == nil ||
			lv.Object.Material == nil || !lv.Object.Material.HasSurface() {
			return optics.Spectrum{}, false
		}
		wiLight := lv.Frame.WorldToLocal(toCamera)
		fLight := lv.Object.Material.Surface.Eval(lv.Context, wiLight, lv.WoLocal)
		if fLight.IsZero() {
			return optics.Spectrum{}, false
		}
		lightFactor = lv.Beta.Mul(fLight)
	}
	mediumID := bdptSegmentMedium(lv, toCamera)
	transmittance := evaluateSegmentTransmittance(getMediumRegistry(tree), mediumID, distance, lv.Context)
	contribution := transmittance.ApplyToSpectrum(lightFactor.Mul(cv.Beta).Mul(fCamera)).MulScalar(geometryTerm)
	return contribution, validSpectrum(contribution)
}

func bdptSegmentMedium(vertex *bdptVertex, outgoing *mat.VecDense) medium.MediumID {
	if vertex == nil {
		return medium.MediumAir
	}
	stack := vertex.MediumStack.Clone()
	if vertex.Kind != bdptVertexSurface || vertex.Object == nil || outgoing == nil {
		return stack.Current()
	}
	boundary := vertex.Object.MediumBoundary
	if !boundary.Active() || boundary.Thin || vertex.GeometricNormal == nil {
		return stack.Current()
	}
	if mat.Dot(vertex.GeometricNormal, outgoing) < 0 {
		if !stack.Contains(boundary.Inside) {
			stack.EnterBoundary(boundary)
		}
	} else if stack.Contains(boundary.Inside) {
		stack.ExitBoundary(boundary)
	}
	return stack.Current()
}

// bdptMISWeight uses cached edge densities and only recomputes the densities
// adjacent to the connection seam. sample counts are included explicitly even
// though the current driver allocates one sample to every (s,t) strategy.
func bdptMISWeight(
	state *bdptSceneState,
	renderCamera camera.BidirectionalCamera,
	lightPath, cameraPath []bdptVertex,
	s, t int,
) float64 {
	if s < 0 || t <= 0 || s > len(lightPath) || t > len(cameraPath) || s+t < 2 {
		return 0
	}
	if s+t == 2 {
		return 1
	}
	lights := append([]bdptVertex(nil), lightPath[:s]...)
	cameras := append([]bdptVertex(nil), cameraPath[:t]...)
	var qs, qsMinus, pt, ptMinus *bdptVertex
	if s > 0 {
		qs = &lights[s-1]
		qs.SampledDelta = false
	}
	if s > 1 {
		qsMinus = &lights[s-2]
	}
	if t > 0 {
		pt = &cameras[t-1]
		pt.SampledDelta = false
	}
	if t > 1 {
		ptMinus = &cameras[t-2]
	}

	if pt != nil {
		if s > 0 {
			pt.PDFRevArea = bdptVertexPDF(qs, qsMinus, pt, renderCamera)
		} else {
			pt.PDFRevArea = bdptLightOriginPDF(state, pt)
		}
	}
	if ptMinus != nil {
		if s > 0 {
			ptMinus.PDFRevArea = bdptVertexPDF(pt, qs, ptMinus, renderCamera)
		} else {
			ptMinus.PDFRevArea = bdptLightEmissionPDF(pt, ptMinus)
		}
	}
	if qs != nil {
		qs.PDFRevArea = bdptVertexPDF(pt, ptMinus, qs, renderCamera)
	}
	if qsMinus != nil {
		qsMinus.PDFRevArea = bdptVertexPDF(qs, pt, qsMinus, renderCamera)
	}

	sum := 0.0
	ratioSquared := 1.0
	currentS, currentT := s, t
	for i := t - 1; i > 0; i-- {
		alternativeS, alternativeT := currentS+1, currentT-1
		ratio := remapBDPTPDF(cameras[i].PDFRevArea) / remapBDPTPDF(cameras[i].PDFFwdArea)
		ratio *= bdptStrategySampleCount(alternativeS, alternativeT) / bdptStrategySampleCount(currentS, currentT)
		ratioSquared *= ratio * ratio
		if !cameras[i].SampledDelta && !cameras[i-1].SampledDelta && bdptStrategyValid(alternativeS, alternativeT) {
			sum += ratioSquared
		}
		currentS, currentT = alternativeS, alternativeT
	}

	ratioSquared = 1
	currentS, currentT = s, t
	for i := s - 1; i >= 0; i-- {
		alternativeS, alternativeT := currentS-1, currentT+1
		ratio := remapBDPTPDF(lights[i].PDFRevArea) / remapBDPTPDF(lights[i].PDFFwdArea)
		ratio *= bdptStrategySampleCount(alternativeS, alternativeT) / bdptStrategySampleCount(currentS, currentT)
		ratioSquared *= ratio * ratio
		deltaPrevious := i > 0 && lights[i-1].SampledDelta
		if !lights[i].SampledDelta && !deltaPrevious && bdptStrategyValid(alternativeS, alternativeT) {
			sum += ratioSquared
		}
		currentS, currentT = alternativeS, alternativeT
	}
	if math.IsNaN(sum) || math.IsInf(sum, 0) || sum < 0 {
		return 0
	}
	return 1 / (1 + sum)
}

func bdptStrategyValid(s, t int) bool {
	return s >= 0 && t >= 1 && s+t >= 2 && !(s == 1 && t == 1)
}

func bdptStrategySampleCount(s, t int) float64 {
	if !bdptStrategyValid(s, t) {
		return 0
	}
	return 1
}

func remapBDPTPDF(pdf float64) float64 {
	if pdf == 0 {
		return 1
	}
	return pdf
}

func bdptVertexPDF(source, previous, next *bdptVertex, renderCamera camera.BidirectionalCamera) float64 {
	if source == nil || next == nil {
		return 0
	}
	toNext := directionBetween(source.Point, next.Point)
	if toNext == nil {
		return 0
	}
	var pdfDirection float64
	switch source.Kind {
	case bdptVertexCamera:
		if renderCamera == nil {
			renderCamera = source.Camera
		}
		if renderCamera == nil {
			return 0
		}
		pdfDirection = renderCamera.PDFDirection(toNext)
	case bdptVertexLight:
		if source.GeometricNormal == nil || source.Object == nil || source.Object.Material == nil ||
			!source.Object.Material.HasEmission() {
			return 0
		}
		wo := source.emissionLocal(toNext)
		pdfDirection = source.Object.Material.Emission.PDFDirection(source.Context, wo)
	case bdptVertexSurface:
		if source.Object == nil || source.Object.Material == nil || !source.Object.Material.HasSurface() || previous == nil {
			return 0
		}
		toPrevious := directionBetween(source.Point, previous.Point)
		if toPrevious == nil {
			return 0
		}
		wi := source.Frame.WorldToLocal(toNext)
		wo := source.Frame.WorldToLocal(toPrevious)
		pdfDirection = source.Object.Material.Surface.PDF(source.Context, wi, wo)
	default:
		return 0
	}
	return convertBDPTDensity(pdfDirection, source, next)
}

func bdptLightOriginPDF(state *bdptSceneState, lightVertex *bdptVertex) float64 {
	if state == nil || lightVertex == nil || lightVertex.Object == nil || state.TotalLightWeight <= 0 {
		return 0
	}
	for _, light := range state.Lights {
		if light.Object == lightVertex.Object && light.Area > 0 {
			return (light.Weight / state.TotalLightWeight) / light.Area
		}
	}
	return 0
}

func bdptLightEmissionPDF(lightVertex, next *bdptVertex) float64 {
	if lightVertex == nil || next == nil || lightVertex.GeometricNormal == nil ||
		lightVertex.Object == nil || lightVertex.Object.Material == nil ||
		!lightVertex.Object.Material.HasEmission() {
		return 0
	}
	direction := directionBetween(lightVertex.Point, next.Point)
	if direction == nil {
		return 0
	}
	wo := lightVertex.emissionLocal(direction)
	pdfDirection := lightVertex.Object.Material.Emission.PDFDirection(lightVertex.Context, wo)
	return convertBDPTDensity(pdfDirection, lightVertex, next)
}

func convertBDPTDensity(pdfDirection float64, source, destination *bdptVertex) float64 {
	if pdfDirection <= 0 || !isFinitePDF(pdfDirection) || source == nil || destination == nil {
		return 0
	}
	direction := directionBetween(source.Point, destination.Point)
	if direction == nil {
		return 0
	}
	distance2 := squaredDistance(source.Point, destination.Point)
	if distance2 <= utils.EPS*utils.EPS {
		return 0
	}
	if destination.Kind == bdptVertexSurface || destination.Kind == bdptVertexLight {
		if destination.GeometricNormal == nil {
			return 0
		}
		pdfDirection *= absDot(destination.GeometricNormal, direction)
	}
	return pdfDirection / distance2
}

func isFinitePDF(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
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

func squaredDistance(a, b *mat.VecDense) float64 {
	if a == nil || b == nil || a.Len() != b.Len() {
		return 0
	}
	d := mat.NewVecDense(a.Len(), nil)
	d.SubVec(a, b)
	return mat.Dot(d, d)
}

func directionBetween(from, to *mat.VecDense) *mat.VecDense {
	if from == nil || to == nil || from.Len() != to.Len() {
		return nil
	}
	result := mat.NewVecDense(from.Len(), nil)
	result.SubVec(to, from)
	length := mat.Norm(result, 2)
	if length <= utils.EPS || math.IsNaN(length) || math.IsInf(length, 0) {
		return nil
	}
	result.ScaleVec(1/length, result)
	return result
}

func negated(v *mat.VecDense) *mat.VecDense {
	if v == nil {
		return nil
	}
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
