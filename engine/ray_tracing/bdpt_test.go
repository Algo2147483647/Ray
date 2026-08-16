package ray_tracing

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/emission"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
)

type fixedCamera struct{ camera.Camera }

func (fixedCamera) SetFilm(*camera.Film) {}

func (fixedCamera) GenerateRay(ray *optics.Ray, _ ...int) *optics.Ray {
	ray.Init()
	ray.Origin.CloneFromVec(mat.NewVecDense(3, []float64{0, 0, 0}))
	ray.Direction.CloneFromVec(mat.NewVecDense(3, []float64{0, 0, 1}))
	return ray
}

func TestBDPTConnectsAreaLightToCameraVertex(t *testing.T) {
	diffuse := &object.Object{
		Shape: shape.NewTriangle(
			mat.NewVecDense(3, []float64{-2, -2, 2}),
			mat.NewVecDense(3, []float64{2, -2, 2}),
			mat.NewVecDense(3, []float64{0, 2, 2}),
		),
		Material: &material.Material{
			Surface: bsdf.NewSingle(bxdf.NewLambert(optics.ConstantSpectrum(0.8))),
		},
	}
	light := &object.Object{
		Shape: shape.NewCircle(
			mat.NewVecDense(3, []float64{0.8, 0, 1}),
			mat.NewVecDense(3, []float64{0, 0, 1}),
			0.2,
		),
		Material: &material.Material{
			Emission: emission.NewConstant(optics.ConstantSpectrum(4)),
		},
	}
	tree := (&object.ObjectTree{}).Build()
	tree.AddObject(diffuse)
	tree.AddObject(light)
	tree.Build()

	h := NewHandler()
	h.IntegratorKind = IntegratorBDPT
	h.SceneGeometry = geometry.Euclidean()
	h.SpectrumMode = optics.SpectrumModeRGB
	h.MaxRayLevel = 2

	value := h.traceBidirectionalSample(fixedCamera{}, tree, 0, 0, 0, 0)
	if !value.IsFinite() || !value.IsNonNegative() {
		t.Fatalf("invalid BDPT contribution: %+v", value)
	}
	if value.IsZero() {
		t.Fatal("expected a visible area light connection to contribute")
	}
}

func TestBDPTFallsBackToPathTracingWithoutSampleableLights(t *testing.T) {
	emitter := &object.Object{
		Shape: &shape.Plane{
			A: mat.NewVecDense(3, []float64{0, 0, 1}),
			B: -1,
		},
		Material: &material.Material{
			Emission: emission.NewConstant(optics.ConstantSpectrum(2)),
		},
	}
	tree := &object.ObjectTree{}
	tree.AddObject(emitter)
	tree.Build()

	h := NewHandler()
	h.IntegratorKind = IntegratorBDPT
	h.SceneGeometry = geometry.Euclidean()
	h.SpectrumMode = optics.SpectrumModeRGB
	value := h.traceBidirectionalSample(fixedCamera{}, tree, 0, 0, 0, 0)
	if value.IsZero() {
		t.Fatal("fallback path tracer should see the unsupported infinite emitter")
	}
}

func TestBDPTConnectionEvaluatesUnweightedPathContribution(t *testing.T) {
	cameraFrame, ok := maths.NewFrameFromNormal(mat.NewVecDense(3, []float64{0, 0, 1}))
	if !ok {
		t.Fatal("failed to build camera frame")
	}
	lightFrame, ok := maths.NewFrameFromNormal(mat.NewVecDense(3, []float64{0, 0, -1}))
	if !ok {
		t.Fatal("failed to build light frame")
	}

	cameraObject := &object.Object{Material: &material.Material{
		Surface: bsdf.NewSingle(bxdf.NewLambert(optics.ConstantSpectrum(0.5))),
	}}
	lightObject := &object.Object{Material: &material.Material{
		Emission: emission.NewConstant(optics.ConstantSpectrum(3)),
	}}
	cameraVertex := bdptVertex{
		Point:           mat.NewVecDense(3, []float64{0, 0, 0}),
		GeometricNormal: mat.NewVecDense(3, []float64{0, 0, 1}),
		Frame:           cameraFrame,
		WoLocal:         maths.NewDirection(0, 0, 1),
		Object:          cameraObject,
		Beta:            optics.ConstantSpectrum(1),
	}
	lightVertex := bdptVertex{
		Point:           mat.NewVecDense(3, []float64{0, 0, 1}),
		GeometricNormal: mat.NewVecDense(3, []float64{0, 0, -1}),
		Frame:           lightFrame,
		Object:          lightObject,
		Beta:            optics.ConstantSpectrum(2),
		LightEndpoint:   true,
	}

	got, ok := (&Handler{}).connectBDPTVertices(nil, &lightVertex, &cameraVertex)
	if !ok {
		t.Fatal("expected the baseline connection to contribute")
	}
	want := 3 / math.Pi
	for channel := 0; channel < 3; channel++ {
		if math.Abs(got.RGB[channel]-want) > 1e-12 {
			t.Fatalf("channel %d = %g, want %g", channel, got.RGB[channel], want)
		}
	}
}

func TestBDPTGlobalMISWeightsFormPartitionOfUnity(t *testing.T) {
	logPDFs := []float64{
		math.Log(2),
		math.Log(2),
		math.Inf(-1), // Disabled strategies must not enter the denominator.
		math.Log(2),
	}
	weights := bdptPowerHeuristicWeights(logPDFs)
	if len(weights) != len(logPDFs) {
		t.Fatalf("weight count = %d, want %d", len(weights), len(logPDFs))
	}
	sum := 0.0
	for _, weight := range weights {
		sum += weight
	}
	if math.Abs(sum-1) > 1e-12 {
		t.Fatalf("MIS weights sum to %g, want 1", sum)
	}
	for _, index := range []int{0, 1, 3} {
		if math.Abs(weights[index]-1.0/3.0) > 1e-12 {
			t.Fatalf("weight %d = %g, want 1/3", index, weights[index])
		}
	}
	if weights[2] != 0 {
		t.Fatalf("disabled strategy weight = %g, want 0", weights[2])
	}
}

func TestBDPTStrategyDensitiesNormalizeAcrossSameCompletePath(t *testing.T) {
	lambertObject := &object.Object{Material: &material.Material{
		Surface: bsdf.NewSingle(bxdf.NewLambert(optics.ConstantSpectrum(0.8))),
	}}
	makeSurfaceVertex := func(point, normal []float64) bdptVertex {
		frame, ok := maths.NewFrameFromNormal(mat.NewVecDense(3, normal))
		if !ok {
			t.Fatalf("failed to build frame for normal %v", normal)
		}
		return bdptVertex{
			Point:           mat.NewVecDense(3, point),
			GeometricNormal: mat.NewVecDense(3, normal),
			Frame:           frame,
			WoLocal:         maths.NewDirection(0, 0, 1),
			Object:          lambertObject,
		}
	}

	lightFrame, ok := maths.NewFrameFromNormal(mat.NewVecDense(3, []float64{0, 0, 1}))
	if !ok {
		t.Fatal("failed to build light frame")
	}
	z0 := bdptVertex{
		Point:           mat.NewVecDense(3, []float64{-1, 0, 2}),
		GeometricNormal: mat.NewVecDense(3, []float64{0, 0, 1}),
		Frame:           lightFrame,
		PDFFwdArea:      0.25,
		LightAreaPDF:    0.25,
		LightEndpoint:   true,
		Object: &object.Object{Shape: shape.NewTriangle(
			mat.NewVecDense(3, []float64{-1, 0, 2}),
			mat.NewVecDense(3, []float64{1, -1, 2}),
			mat.NewVecDense(3, []float64{1, 1, 2}),
		)},
	}
	z1 := makeSurfaceVertex([]float64{0, 0, 1}, []float64{0, 0, 1})
	z2 := makeSurfaceVertex([]float64{1, 0, 2}, []float64{0, 0, -1})
	z3 := makeSurfaceVertex([]float64{2, 0, 1}, []float64{0, 0, 1})

	lightPath := []bdptVertex{z0, z1, z2}
	cameraPath := []bdptVertex{z3, z2, z1}
	weights := []float64{
		bdptMISWeight(lightPath[:1], cameraPath, 0, 2),
		bdptMISWeight(lightPath[:2], cameraPath[:2], 1, 1),
		bdptMISWeight(lightPath, cameraPath[:1], 2, 0),
		bdptMISWeightReferenceDirect(lightPath[:1], cameraPath, 2),
	}
	sum := 0.0
	for strategy, weight := range weights {
		if weight <= 0 || weight >= 1 {
			t.Fatalf("strategy %d weight = %g, want a finite interior weight", strategy+1, weight)
		}
		sum += weight
	}
	if math.Abs(sum-1) > 1e-12 {
		t.Fatalf("same-path strategy weights sum to %g, want 1", sum)
	}
}

func TestBDPTCapabilityGateAllowsDeltaAndHomogeneousAbsorption(t *testing.T) {
	tree := &object.ObjectTree{}
	if bdptSupportsScene(geometry.Spherical(), tree) {
		t.Fatal("non-Euclidean geometry must use the path integrator fallback")
	}

	registry := medium.NewRegistry()
	absorbingID, err := registry.RegisterHomogeneousWithCoefficients(
		"absorbing",
		medium.NewConstant(1.2),
		medium.ConstantCoefficient(0.25),
		nil,
	)
	if err != nil {
		t.Fatalf("register medium: %v", err)
	}
	tree.Media = registry
	tree.AddObject(&object.Object{
		MediumBoundary: medium.NewBoundary(medium.MediumAir, absorbingID),
	})
	if !bdptSupportsScene(geometry.Euclidean(), tree) {
		t.Fatal("homogeneous absorption should remain in the BDPT/light-splat path family")
	}

	deltaTree := &object.ObjectTree{}
	deltaTree.AddObject(&object.Object{Material: &material.Material{
		Surface: bsdf.NewSingle(bxdf.NewSpecularReflection(optics.ConstantSpectrum(1))),
	}})
	if !bdptSupportsScene(geometry.Euclidean(), deltaTree) {
		t.Fatal("delta surfaces should be sampled in subpaths and handled by t=1 caustic splats")
	}
}

func TestBDPTProjectsDeltaCausticWithT1Strategy(t *testing.T) {
	renderCamera := camera.NewCamera3D()
	renderCamera.Position = mat.NewVecDense(3, []float64{0, 0, 0})
	renderCamera.Coordinates = []*mat.VecDense{mat.NewVecDense(3, []float64{0, 0, 1}), mat.NewVecDense(3, []float64{-1, 0, 0}), mat.NewVecDense(3, []float64{0, 1, 0})}
	renderCamera.FieldOfViews = []float64{60, 60}
	if err := renderCamera.Prepare(); err != nil {
		t.Fatalf("prepare camera: %v", err)
	}

	normal := mat.NewVecDense(3, []float64{0, 0, -1})
	frame, ok := maths.NewFrameFromNormal(normal)
	if !ok {
		t.Fatal("failed to build screen frame")
	}
	screen := &object.Object{Material: &material.Material{
		Surface: bsdf.NewSingle(bxdf.NewLambert(optics.ConstantSpectrum(0.8))),
	}}
	path := []bdptVertex{
		{SampledDelta: true},
		{
			Point: mat.NewVecDense(3, []float64{0, 0, 1}), GeometricNormal: normal,
			Frame: frame, WoLocal: maths.NewDirection(0, 0, 1), Object: screen,
			Beta: optics.ConstantSpectrum(1), MediumStack: medium.NewStack(medium.MediumAir),
		},
	}
	renderCamera.Film = camera.NewFilm(1, 1)
	splats := NewHandler().projectBDPTDeltaCaustics(renderCamera, nil, path, 0, 0)
	if len(splats) != 1 {
		t.Fatalf("delta t=1 splat count = %d, want 1", len(splats))
	}
	if splats[0].Pixel != 0 || !validSpectrum(splats[0].Value) {
		t.Fatalf("invalid delta t=1 splat: %+v", splats[0])
	}
}

func TestBDPTDeltaReconstructionFilterPreservesEnergy(t *testing.T) {
	mask := make([]bool, 16)
	for index := range mask {
		mask[index] = true
	}
	splat := FilmSplat{
		Pixel: 5, Value: optics.ConstantSpectrum(8),
		projection: camera.FilmProjection{Raster: camera.RasterPosition{X: 1.25, Y: 1.75}},
	}
	filtered := filterBDPTDeltaSplat(splat, 4, 4, mask)
	if len(filtered) != 4 {
		t.Fatalf("filtered splat count = %d, want 4", len(filtered))
	}
	var sum optics.Spectrum
	for _, item := range filtered {
		sum = sum.Add(item.Value)
	}
	for channel := range 3 {
		if math.Abs(sum.RGB[channel]-8) > 1e-12 {
			t.Fatalf("filtered energy channel %d = %g, want 8", channel, sum.RGB[channel])
		}
	}
}

func TestBDPTContinuousMISRejectsDeltaMeasure(t *testing.T) {
	lightPath := []bdptVertex{{PDFFwdArea: 1, SampledDelta: true}}
	cameraPath := []bdptVertex{{}}
	if weight := bdptMISWeight(lightPath, cameraPath, 0, 0); weight != 0 {
		t.Fatalf("continuous MIS weight for a delta path = %g, want 0", weight)
	}
}

func TestCollectAreaLightsUsesPowerWeightedDistribution(t *testing.T) {
	tree := &object.ObjectTree{}
	for index, radiance := range []float64{1, 10} {
		tree.AddObject(&object.Object{
			Shape: shape.NewCircle(
				mat.NewVecDense(3, []float64{float64(index), 0, 1}),
				mat.NewVecDense(3, []float64{0, 0, 1}),
				1,
			),
			Material: &material.Material{Emission: emission.NewConstant(optics.ConstantSpectrum(radiance))},
		})
	}
	lights, totalWeight := collectAreaLights(tree)
	if len(lights) != 2 || totalWeight <= 0 {
		t.Fatalf("invalid prepared light distribution: count=%d weight=%g", len(lights), totalWeight)
	}
	if ratio := lights[1].Weight / lights[0].Weight; math.Abs(ratio-10) > 1e-12 {
		t.Fatalf("power weight ratio = %g, want 10", ratio)
	}
}

func TestBDPTRussianRouletteUsesConfiguredDepth(t *testing.T) {
	beta := optics.ConstantSpectrum(0.2)
	if got := bdptSurvivalProbability(beta, 2, 3); got != 1 {
		t.Fatalf("survival before configured depth = %g, want 1", got)
	}
	if got := bdptSurvivalProbability(beta, 3, 3); math.Abs(got-bdptRussianRouletteSurvival) > 1e-12 {
		t.Fatalf("survival at configured depth = %g, want %g", got, bdptRussianRouletteSurvival)
	}
}

func TestBDPTEdgePDFIncludesReconstructibleRussianRouletteProbability(t *testing.T) {
	lambertObject := &object.Object{Material: &material.Material{
		Surface: bsdf.NewSingle(bxdf.NewLambert(optics.ConstantSpectrum(0.8))),
	}}
	makeVertex := func(z float64) bdptVertex {
		normal := mat.NewVecDense(3, []float64{0, 0, 1})
		frame, ok := maths.NewFrameFromNormal(normal)
		if !ok {
			t.Fatal("failed to build test frame")
		}
		return bdptVertex{
			Point: mat.NewVecDense(3, []float64{0, 0, z}), GeometricNormal: normal,
			Frame: frame, WoLocal: maths.NewDirection(0, 0, 1), Object: lambertObject,
			RRDepth: 2,
		}
	}
	path := []bdptVertex{makeVertex(0), makeVertex(1), makeVertex(2), makeVertex(3)}
	withoutRR := 1 / math.Pi
	got := bdptEdgeAreaPDF(path, 1, 2, -1, bxdf.TransportImportance)
	if math.Abs(got-withoutRR*bdptRussianRouletteSurvival) > 1e-12 {
		t.Fatalf("edge area PDF = %g, want %g", got, withoutRR*bdptRussianRouletteSurvival)
	}
}

var _ camera.RayCamera = fixedCamera{}
