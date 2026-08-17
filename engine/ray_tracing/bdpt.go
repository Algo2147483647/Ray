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
	RRDepth         int64
	LightEndpoint   bool
	MediumStack     medium.Stack
}

type areaLight struct {
	Object  *object.Object
	Sampler shape.SurfaceSampler
	Area    float64
	Weight  float64
}

// bdptSceneState contains immutable data that is prepared once for a render.
// Keeping capability analysis and light discovery out of the sample loop is
// essential for scenes with many objects or high sample counts.
type bdptSceneState struct {
	Lights           []areaLight
	TotalLightWeight float64
	FallbackReason   string
}

func prepareBDPTScene(sceneGeometry geometry.Geometry, tree *object.ObjectTree) *bdptSceneState {
	state := &bdptSceneState{}
	state.FallbackReason = bdptUnsupportedReason(sceneGeometry, tree)
	state.Lights, state.TotalLightWeight = collectAreaLights(tree)
	if state.FallbackReason == "" && (len(state.Lights) == 0 || state.TotalLightWeight <= 0) {
		state.FallbackReason = "scene has no sampleable finite area light"
	}
	return state
}

// traceBidirectionalSample is the direct-call compatibility wrapper. Production
// renders prepare scene capabilities and the light distribution once in
// bdptKernel.Prepare, then call traceBidirectionalPrepared for every sample.
func (h *Handler) traceBidirectionalSample(
	renderCamera camera.RayCamera,
	objTree *object.ObjectTree,
	wavelengthNM, wavelengthPDF float64,
	index ...int,
) optics.Spectrum {
	state := prepareBDPTScene(h.SceneGeometry, objTree)
	result, _ := h.traceBidirectionalPrepared(
		state, renderCamera, objTree, wavelengthNM, wavelengthPDF, index...,
	)
	return result
}

func (h *Handler) traceBidirectionalPrepared(
	state *bdptSceneState,
	renderCamera camera.RayCamera,
	objTree *object.ObjectTree,
	wavelengthNM, wavelengthPDF float64,
	index ...int,
) (optics.Spectrum, []bdptVertex) {
	if state == nil || state.FallbackReason != "" {
		return h.tracePathSampleSpectrum(renderCamera, objTree, wavelengthNM, wavelengthPDF, index...), nil
	}

	cameraPath, emitted := h.buildCameraSubpath(renderCamera, objTree, wavelengthNM, wavelengthPDF, index...)
	lightPath := h.buildLightSubpath(
		objTree, state.Lights, state.TotalLightWeight, wavelengthNM, wavelengthPDF,
	)
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

	return result, lightPath
}

func bdptSupportsScene(sceneGeometry geometry.Geometry, tree *object.ObjectTree) bool {
	return bdptUnsupportedReason(sceneGeometry, tree) == ""
}

func bdptUnsupportedReason(sceneGeometry geometry.Geometry, tree *object.ObjectTree) string {
	if geometry.Get(sceneGeometry).Kind() != geometry.EuclideanKind {
		return "BDPT currently requires three-dimensional Euclidean geometry"
	}
	if tree == nil {
		return ""
	}
	for _, obj := range tree.Objects {
		if obj == nil {
			continue
		}
		if obj.Material != nil && (obj.Material.Metadata.NonReciprocal ||
			(obj.Material.HasSurface() && obj.Material.Surface.DeltaFlags()&bxdf.NonReciprocal != 0)) {
			return "scene contains a non-reciprocal surface"
		}
	}
	return ""
}

func (h *Handler) tracePathSampleSpectrum(
	renderCamera camera.RayCamera,
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
		emitted := obj.Material.Emission.Emit(
			bxdf.ShadingContext{SpectrumMode: optics.SpectrumModeRGB},
			maths.NewDirection(0, 0, 1),
		)
		powerEstimate := emitted.MaxComponent()
		if powerEstimate <= 0 || math.IsNaN(powerEstimate) || math.IsInf(powerEstimate, 0) {
			powerEstimate = 1
		}
		weight := area * powerEstimate
		lights = append(lights, areaLight{
			Object: obj, Sampler: sampler, Area: area, Weight: weight,
		})
		totalWeight += weight
	}
	return lights, totalWeight
}

func (h *Handler) buildCameraSubpath(
	renderCamera camera.RayCamera,
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
			Beta: beta, PDFFwdArea: pdfArea, RRDepth: h.RussianRouletteDepth,
			MediumStack: ray.MediumStack.Clone(),
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
		path = append(path, vertex)
		if survival < 1 {
			if rand.Float64() >= survival {
				break
			}
			beta = beta.DivScalar(survival)
		}
		if sample.Flags&bxdf.TransmissionEvent != 0 {
			applyMediumTransmission(getMediumRegistry(tree), ray, si.Context, si.Object.MediumBoundary, sample)
		}
		si.Frame.LocalToWorldInto(ray.Direction, sample.Wi)
		ray.Origin.CopyVec(hit.Point)
		pendingPDF = vertex.SampledPDF * survival
		previousDelta = vertex.SampledDelta
	}
	return path, emitted
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
	if selectionPDF <= 0 || math.IsNaN(selectionPDF) || math.IsInf(selectionPDF, 0) {
		return areaLight{}, 0, false
	}
	return selected, selectionPDF, true
}

func (h *Handler) makeLightEndpoint(
	selected areaLight,
	selectionPDF float64,
	ss shape.SurfaceSample,
	wavelengthNM, wavelengthPDF float64,
) (bdptVertex, bool) {
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
		PDFFwdArea: pdfLightArea,
		RRDepth:    h.RussianRouletteDepth, LightEndpoint: true,
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
			Beta: beta, PDFFwdArea: pdfArea, RRDepth: h.RussianRouletteDepth,
			MediumStack: ray.MediumStack.Clone(),
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
		path = append(path, vertex)
		if survival < 1 {
			if rand.Float64() >= survival {
				break
			}
			beta = beta.DivScalar(survival)
		}
		if sample.Flags&bxdf.TransmissionEvent != 0 {
			applyMediumTransmission(getMediumRegistry(tree), ray, si.Context, si.Object.MediumBoundary, sample)
		}
		si.Frame.LocalToWorldInto(ray.Direction, sample.Wi)
		ray.Origin.CopyVec(hit.Point)
		pendingPDF = vertex.SampledPDF * survival
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
	view := bdptPathView{light: lightPath, camera: cameraPath, li: li, ci: ci}
	if view.len() < 2 || li < 0 || ci < 0 {
		return 0
	}

	// Discrete events use probability masses rather than solid-angle PDFs. The
	// continuous t>=2 strategy family must not mix those measures before reverse
	// discrete densities and eta factors exist. Delta caustics are handled by the
	// separate t=1 film-splat estimator in bdptKernel.
	if view.containsSampledDelta() {
		return 0
	}
	return bdptStrategyWeight(view, li+1)
}

func bdptStrategyWeight(view bdptPathView, currentStrategy int) float64 {
	logPDFs := make([]float64, 0, view.len()-1)
	for strategy := 1; strategy < view.len(); strategy++ {
		logPDFs = append(logPDFs, bdptStrategyLogPDFView(view.len(), view.vertex, strategy))
	}
	weights := bdptPowerHeuristicWeights(logPDFs)
	if currentStrategy <= 0 || currentStrategy > len(weights) {
		return 0
	}
	return weights[currentStrategy-1]
}

type bdptPathView struct {
	light, camera []bdptVertex
	li, ci        int
}

func (v bdptPathView) len() int { return v.li + v.ci + 2 }

func (v bdptPathView) vertex(index int) *bdptVertex {
	if index < 0 || index >= v.len() {
		return nil
	}
	if index <= v.li {
		return &v.light[index]
	}
	cameraIndex := v.ci - (index - v.li - 1)
	return &v.camera[cameraIndex]
}

func (v bdptPathView) containsSampledDelta() bool {
	for i := 0; i <= v.li; i++ {
		if v.light[i].SampledDelta {
			return true
		}
	}
	for i := 0; i <= v.ci; i++ {
		if v.camera[i].SampledDelta {
			return true
		}
	}
	return false
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

func bdptStrategyLogPDFView(
	pathLength int,
	vertex func(int) *bdptVertex,
	strategy int,
) float64 {
	root := vertex(0)
	rootPDF := 0.0
	if root != nil {
		rootPDF = root.PDFFwdArea
	}
	if strategy <= 0 || strategy >= pathLength || root == nil ||
		rootPDF <= 0 || !isFinitePDF(rootPDF) {
		return math.Inf(-1)
	}
	logPDF := math.Log(rootPDF)

	for source := 0; source < strategy-1; source++ {
		pdfArea := bdptEdgeAreaPDFView(
			pathLength, vertex, source, source+1, source-1, bxdf.TransportImportance,
		)
		if pdfArea <= 0 || !isFinitePDF(pdfArea) {
			return math.Inf(-1)
		}
		logPDF += math.Log(pdfArea)
	}
	for source := pathLength - 1; source > strategy; source-- {
		pdfArea := bdptEdgeAreaPDFView(
			pathLength, vertex, source, source-1, source+1, bxdf.TransportRadiance,
		)
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
	return bdptEdgeAreaPDFView(len(path), func(index int) *bdptVertex {
		if index < 0 || index >= len(path) {
			return nil
		}
		return &path[index]
	}, sourceIndex, destinationIndex, previousIndex, mode)
}

func bdptEdgeAreaPDFView(
	pathLength int,
	vertex func(int) *bdptVertex,
	sourceIndex, destinationIndex, previousIndex int,
	mode bxdf.TransportMode,
) float64 {
	source := vertex(sourceIndex)
	destination := vertex(destinationIndex)
	if source == nil || destination == nil {
		return 0
	}
	var toDestination [3]float64
	distance2, ok := directionBetween3(source.Point, destination.Point, &toDestination)
	if !ok {
		return 0
	}

	var pdfDirection float64
	if source.LightEndpoint {
		pdfDirection = 0.5 * absDotDirection3(source.GeometricNormal, &toDestination) / math.Pi
	} else {
		if source.Object == nil || source.Object.Material == nil ||
			!source.Object.Material.HasSurface() {
			return 0
		}
		var wiStorage [3]float64
		wi, ok := frameWorldToLocal3(source.Frame, &toDestination, &wiStorage)
		if !ok {
			return 0
		}
		wo := source.WoLocal
		if previousIndex >= 0 && previousIndex < pathLength {
			previous := vertex(previousIndex)
			if previous == nil {
				return 0
			}
			var toPrevious, woStorage [3]float64
			if _, ok := directionBetween3(source.Point, previous.Point, &toPrevious); !ok {
				return 0
			}
			wo, ok = frameWorldToLocal3(source.Frame, &toPrevious, &woStorage)
			if !ok {
				return 0
			}
		}
		ctx := source.Context
		ctx.TransportMode = mode
		pdfDirection = source.Object.Material.Surface.PDF(ctx, wi, wo)
	}
	if pdfDirection <= 0 || !isFinitePDF(pdfDirection) {
		return 0
	}
	pdfDirection *= bdptEdgeSurvivalProbability(
		source, pathLength, sourceIndex, destinationIndex,
	)

	if distance2 <= utils.EPS*utils.EPS {
		return 0
	}
	return pdfDirection * absDotDirection3(destination.GeometricNormal, &toDestination) / distance2
}

func directionBetween3(from, to *mat.VecDense, result *[3]float64) (float64, bool) {
	if from == nil || to == nil || result == nil || from.Len() != 3 || to.Len() != 3 {
		return 0, false
	}
	distance2 := 0.0
	for i := range 3 {
		component := to.AtVec(i) - from.AtVec(i)
		result[i] = component
		distance2 += component * component
	}
	if distance2 <= utils.EPS*utils.EPS || math.IsNaN(distance2) || math.IsInf(distance2, 0) {
		return 0, false
	}
	inverseDistance := 1 / math.Sqrt(distance2)
	for i := range 3 {
		result[i] *= inverseDistance
	}
	return distance2, true
}

func frameWorldToLocal3(
	frame maths.Frame,
	direction *[3]float64,
	storage *[3]float64,
) (maths.Direction, bool) {
	if direction == nil || storage == nil || frame.Normal == nil || frame.Normal.Len() != 3 ||
		len(frame.Tangents) != 2 || frame.Tangents[0] == nil || frame.Tangents[1] == nil {
		return nil, false
	}
	for tangent := range 2 {
		storage[tangent] = dotVecDirection3(frame.Tangents[tangent], direction)
	}
	storage[2] = dotVecDirection3(frame.Normal, direction)
	return maths.Direction(storage[:]), true
}

func dotVecDirection3(vector *mat.VecDense, direction *[3]float64) float64 {
	if vector == nil || direction == nil || vector.Len() != 3 {
		return 0
	}
	return vector.AtVec(0)*direction[0] + vector.AtVec(1)*direction[1] + vector.AtVec(2)*direction[2]
}

func absDotDirection3(vector *mat.VecDense, direction *[3]float64) float64 {
	return math.Abs(dotVecDirection3(vector, direction))
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

const bdptRussianRouletteSurvival = 0.8

func bdptSurvivalProbability(_ optics.Spectrum, nextDepth, configuredDepth int64) float64 {
	depth := configuredDepth
	if depth <= 0 {
		depth = 3
	}
	if nextDepth < depth {
		return 1
	}
	// A throughput-dependent probability cannot be reconstructed for alternative
	// BDPT strategies without replaying their complete throughput. A fixed
	// probability is still unbiased, bounds each roulette amplification, and can
	// be included exactly in every strategy density.
	return bdptRussianRouletteSurvival
}

func bdptEdgeSurvivalProbability(
	source *bdptVertex,
	pathLength, sourceIndex, destinationIndex int,
) float64 {
	if source == nil || source.LightEndpoint {
		return 1
	}
	var sourceDepth int64
	switch destinationIndex {
	case sourceIndex + 1:
		sourceDepth = int64(sourceIndex)
	case sourceIndex - 1:
		sourceDepth = int64(pathLength - 1 - sourceIndex)
	default:
		return 1
	}
	return bdptSurvivalProbability(optics.Spectrum{}, sourceDepth+1, source.RRDepth)
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
