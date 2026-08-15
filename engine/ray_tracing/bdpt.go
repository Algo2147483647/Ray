package ray_tracing

import (
	"math"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
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
	MediumStack     medium.Stack
}

type areaLight struct {
	Object  *object.Object
	Sampler shape.SurfaceSampler
	Area    float64
}

// traceBidirectionalSample builds camera and light subpaths, connects every
// continuous surface pair, and combines all enabled t>=2 strategies for each
// complete path with one global power-heuristic denominator.
//
// Delta, non-reciprocal, participating-media, and non-Euclidean scenes fall
// back to the path integrator until their reverse densities and measures are
// represented explicitly.
func (h *Handler) traceBidirectionalSample(
	renderCamera camera.Camera,
	objTree *object.ObjectTree,
	wavelengthNM, wavelengthPDF float64,
	index ...int,
) optics.Spectrum {
	if !bdptSupportsScene(h.SceneGeometry, objTree) {
		return h.tracePathSampleSpectrum(renderCamera, objTree, wavelengthNM, wavelengthPDF, index...)
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
			if int64(li+ci+1) > h.MaxRayLevel {
				continue
			}
			weight := bdptMISWeight(lightPath, cameraPath, li, ci)
			if weight <= 0 {
				continue
			}
			contribution, ok := h.connectBDPTVertices(objTree, &lightPath[li], &cameraPath[ci])
			if ok {
				result = result.Add(contribution.MulScalar(weight))
			}
		}
	}
	return result
}

func bdptSupportsScene(sceneGeometry geometry.Geometry, tree *object.ObjectTree) bool {
	if geometry.Get(sceneGeometry).Kind() != geometry.EuclideanKind {
		return false
	}
	if tree == nil {
		return true
	}
	for _, obj := range tree.Objects {
		if obj == nil {
			continue
		}
		if obj.MediumBoundary.Active() {
			return false
		}
		if obj.Material != nil && obj.Material.HasSurface() &&
			obj.Material.Surface.DeltaFlags() != bxdf.DeltaNone {
			return false
		}
		if obj.Material != nil && obj.Material.Metadata.NonReciprocal {
			return false
		}
	}
	return true
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
		segmentLength := hit.ArcLength
		if segmentLength <= 0 {
			segmentLength = ray.G().ArcLengthFromEmbedT(ray.Origin, ray.Direction, hit.Distance)
		}
		beta = evaluateSegmentTransmittance(
			getMediumRegistry(tree),
			ray.MediumStack.Current(),
			segmentLength,
			h.newShadingContext(ray),
		).ApplyToSpectrum(beta)
		if !validSpectrum(beta) {
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
			Beta: beta, PDFFwdArea: pdfArea, MediumStack: ray.MediumStack.Clone(),
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

func (h *Handler) sampleLightEndpoint(
	lights []areaLight,
	totalArea, wavelengthNM, wavelengthPDF float64,
) (bdptVertex, bool) {
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
		return bdptVertex{}, false
	}
	selectionPDF := selected.Area / totalArea
	pdfLightArea := selectionPDF * ss.PDFArea
	if pdfLightArea <= 0 || math.IsNaN(pdfLightArea) || math.IsInf(pdfLightArea, 0) {
		return bdptVertex{}, false
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

	frame, ok := maths.NewFrameFromNormal(ss.Normal)
	if !ok {
		return bdptVertex{}, false
	}
	return bdptVertex{
		Point: ss.Point, GeometricNormal: ss.Normal, Frame: frame,
		Context: ctx, Object: selected.Object, Beta: unitSpectrum(wavelengthNM).MulScalar(1 / pdfLightArea),
		PDFFwdArea: pdfLightArea, LightEndpoint: true,
		MediumStack: medium.NewStack(medium.MediumAir),
	}, true
}

func (h *Handler) buildLightSubpath(
	tree *object.ObjectTree,
	lights []areaLight,
	totalArea, wavelengthNM, wavelengthPDF float64,
) []bdptVertex {
	root, ok := h.sampleLightEndpoint(lights, totalArea, wavelengthNM, wavelengthPDF)
	if !ok {
		return nil
	}

	emissionNormal := mat.VecDenseCopyOf(root.GeometricNormal)
	if rand.Float64() < 0.5 {
		emissionNormal.ScaleVec(-1, emissionNormal)
	}
	emissionFrame, ok := maths.NewFrameFromNormal(emissionNormal)
	if !ok {
		return nil
	}
	localDirection := maths.CosineSampleHemisphere(maths.Sample2D{U: rand.Float64(), V: rand.Float64()})
	pdfDirection := 0.5 * maths.CosineHemispherePDF(localDirection)
	if pdfDirection <= 0 {
		return nil
	}
	worldDirection := emissionFrame.LocalToWorld(localDirection)
	emitted := root.Object.Material.Emission.Emit(root.Context, localDirection)
	beta := emitted.MulScalar(
		maths.AbsCosTheta(localDirection) / (root.PDFFwdArea * pdfDirection),
	)
	root.Frame = emissionFrame
	root.SampledPDF = pdfDirection
	path := []bdptVertex{root}

	ray := &optics.Ray{
		Origin:    mat.VecDenseCopyOf(root.Point),
		Direction: mat.VecDenseCopyOf(worldDirection),
		Geometry:  h.SceneGeometry,
	}
	ray.Init()
	ray.Origin.CopyVec(root.Point)
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
		segmentLength := hit.ArcLength
		if segmentLength <= 0 {
			segmentLength = ray.G().ArcLengthFromEmbedT(ray.Origin, ray.Direction, hit.Distance)
		}
		media := getMediumRegistry(tree)
		beta = evaluateSegmentTransmittance(
			media,
			ray.MediumStack.Current(),
			segmentLength,
			h.newShadingContext(ray),
		).ApplyToSpectrum(beta)
		if !validSpectrum(beta) {
			break
		}
		distance2 := squaredDistance(origin, hit.Point)
		pdfArea := pendingPDF * absDot(hit.GeometricNormal, negated(direction)) /
			math.Max(distance2, utils.EPS)
		si, ok := h.prepareSurfaceInteraction(getMediumRegistry(tree), ray, hit)
		if !ok {
			break
		}
		si.Context.TransportMode = bxdf.TransportImportance
		vertex := bdptVertex{
			Point: hit.Point, GeometricNormal: hit.GeometricNormal, Frame: si.Frame,
			WoLocal: si.WoLocal, Context: si.Context, Object: si.Object,
			Beta: beta, PDFFwdArea: pdfArea, MediumStack: ray.MediumStack.Clone(),
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
	lv, cv *bdptVertex,
) (optics.Spectrum, bool) {
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

	transmittance := evaluateSegmentTransmittance(
		getMediumRegistry(tree),
		lv.MediumStack.Current(),
		distance,
		lv.Context,
	)
	contribution := transmittance.ApplyToSpectrum(
		lightFactor.Mul(cv.Beta).Mul(fCamera),
	).MulScalar(geometryTerm)
	return contribution, validSpectrum(contribution)
}

func bdptMISWeight(lightPath, cameraPath []bdptVertex, li, ci int) float64 {
	path := make([]bdptVertex, 0, li+ci+2)
	path = append(path, lightPath[:li+1]...)
	for i := ci; i >= 0; i-- {
		path = append(path, cameraPath[i])
	}
	currentStrategy := li + 1
	logPDFs := bdptStrategyLogPDFs(path)
	if currentStrategy <= 0 || currentStrategy >= len(path) ||
		math.IsInf(logPDFs[currentStrategy-1], -1) {
		return 0
	}
	weights := bdptPowerHeuristicWeights(logPDFs)
	if len(weights) != len(logPDFs) {
		return 0
	}
	return weights[currentStrategy-1]
}

func bdptPowerHeuristicWeights(logPDFs []float64) []float64 {
	weights := make([]float64, len(logPDFs))
	maxLogTerm := math.Inf(-1)
	for _, logPDF := range logPDFs {
		if !math.IsInf(logPDF, -1) {
			maxLogTerm = math.Max(maxLogTerm, 2*logPDF)
		}
	}
	if math.IsInf(maxLogTerm, -1) {
		return weights
	}
	denominator := 0.0
	for _, logPDF := range logPDFs {
		if !math.IsInf(logPDF, -1) {
			denominator += math.Exp(2*logPDF - maxLogTerm)
		}
	}
	if denominator <= 0 || math.IsNaN(denominator) || math.IsInf(denominator, 0) {
		return weights
	}
	for i, logPDF := range logPDFs {
		if !math.IsInf(logPDF, -1) {
			weights[i] = math.Exp(2*logPDF-maxLogTerm) / denominator
		}
	}
	return weights
}

// bdptStrategyLogPDFs returns the path density for every enabled split s in
// [1,len(path)-1]. The camera endpoint is fixed to the current pixel, so t=1
// and s=0 strategies are intentionally outside this strategy family.
func bdptStrategyLogPDFs(path []bdptVertex) []float64 {
	if len(path) < 2 {
		return nil
	}
	result := make([]float64, len(path)-1)
	for strategy := 1; strategy < len(path); strategy++ {
		result[strategy-1] = bdptStrategyLogPDF(path, strategy)
	}
	return result
}

func bdptStrategyLogPDF(path []bdptVertex, strategy int) float64 {
	if strategy <= 0 || strategy >= len(path) ||
		path[0].PDFFwdArea <= 0 || !isFinitePDF(path[0].PDFFwdArea) {
		return math.Inf(-1)
	}
	logPDF := math.Log(path[0].PDFFwdArea)

	for source := 0; source < strategy-1; source++ {
		pdfArea := bdptEdgeAreaPDF(path, source, source+1, source-1, bxdf.TransportImportance)
		if pdfArea <= 0 || !isFinitePDF(pdfArea) {
			return math.Inf(-1)
		}
		logPDF += math.Log(pdfArea)
	}
	for source := len(path) - 1; source > strategy; source-- {
		pdfArea := bdptEdgeAreaPDF(path, source, source-1, source+1, bxdf.TransportRadiance)
		if pdfArea <= 0 || !isFinitePDF(pdfArea) {
			return math.Inf(-1)
		}
		logPDF += math.Log(pdfArea)
	}
	return logPDF
}

func bdptEdgeAreaPDF(
	path []bdptVertex,
	sourceIndex, destinationIndex, previousIndex int,
	mode bxdf.TransportMode,
) float64 {
	source := &path[sourceIndex]
	destination := &path[destinationIndex]
	toDestination := directionBetween(source.Point, destination.Point)
	if toDestination == nil {
		return 0
	}

	var pdfDirection float64
	if source.LightEndpoint {
		pdfDirection = 0.5 * absDot(source.GeometricNormal, toDestination) / math.Pi
	} else {
		if source.Object == nil || source.Object.Material == nil ||
			!source.Object.Material.HasSurface() {
			return 0
		}
		wi := source.Frame.WorldToLocal(toDestination)
		wo := source.WoLocal
		if previousIndex >= 0 && previousIndex < len(path) {
			toPrevious := directionBetween(source.Point, path[previousIndex].Point)
			if toPrevious == nil {
				return 0
			}
			wo = source.Frame.WorldToLocal(toPrevious)
		}
		ctx := source.Context
		ctx.TransportMode = mode
		pdfDirection = source.Object.Material.Surface.PDF(ctx, wi, wo)
	}
	if pdfDirection <= 0 || !isFinitePDF(pdfDirection) {
		return 0
	}

	distance2 := squaredDistance(source.Point, destination.Point)
	if distance2 <= utils.EPS*utils.EPS {
		return 0
	}
	toSource := negated(toDestination)
	return pdfDirection * absDot(destination.GeometricNormal, toSource) / distance2
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
