package ray_tracing

import (
	"math"
	randv2 "math/rand/v2"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model"
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

type nonBidirectionalCamera struct{ Film *camera.Film }

func (nonBidirectionalCamera) GenerateRay(ray *optics.Ray, _ []int, _ ...int) *optics.Ray {
	ray.Init()
	ray.Origin.CloneFromVec(mat.NewVecDense(3, []float64{0, 0, 0}))
	ray.Direction.CloneFromVec(mat.NewVecDense(3, []float64{0, 0, 1}))
	return ray
}

func (nonBidirectionalCamera) RasterDimension() int { return 2 }

type bdptTestCamera struct {
	*camera.Camera3D
	Film *camera.Film
}

func newBDPTTestCamera(t testing.TB, width, height int) *bdptTestCamera {
	t.Helper()
	result := &bdptTestCamera{Camera3D: camera.NewCamera3D(), Film: camera.NewFilm(width, height)}
	result.Position = mat.NewVecDense(3, []float64{0, 0, 0})
	result.Coordinates = []*mat.VecDense{
		mat.NewVecDense(3, []float64{0, 0, 1}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
	}
	result.FieldOfViews = []float64{45, 45}
	result.Film.InitSpectralBins(3, 400, 700)
	if err := result.Prepare(); err != nil {
		t.Fatalf("prepare camera: %v", err)
	}
	return result
}

func bdptTarget(c *bdptTestCamera) model.RenderTarget {
	return model.RenderTarget{Camera: c, Film: c.Film, Output: "test.bin"}
}

func addTestAreaLight(tree *object.ObjectTree, center []float64) *object.Object {
	light := &object.Object{
		Shape: shape.NewCircle(
			mat.NewVecDense(3, center),
			mat.NewVecDense(3, []float64{0, 0, -1}),
			0.2,
		),
		Material: &material.Material{Emission: emission.NewConstant(optics.ConstantSpectrum(4))},
	}
	tree.AddObject(light)
	return light
}

func newBDPTTestHandler() *Handler {
	h := NewHandler(geometry.DefaultSceneSpace())
	h.IntegratorKind = IntegratorBDPT
	h.WavelengthSamples = 1
	h.ThreadNum = 1
	h.MaxRayLevel = 3
	return h
}

func (h *Handler) traceBidirectionalSample(
	renderCamera camera.RayCamera,
	objTree *object.ObjectTree,
	wavelengthNM, wavelengthPDF float64,
	index ...int,
) float64 {
	film := camera.NewFilm(1, 1)
	film.InitSpectralBins(3, 400, 700)
	state, err := h.prepareBDPT(renderCamera, film, objTree)
	if err != nil {
		return 0
	}
	result, _ := h.traceBidirectionalPrepared(state, renderCamera, film.Shape, objTree, wavelengthNM, wavelengthPDF, index...)
	return result
}

func TestBDPTPreflightValidatesBeforeFilmIndexing(t *testing.T) {
	h := newBDPTTestHandler()
	tree := (&object.ObjectTree{}).Build()
	addTestAreaLight(tree, []float64{0, 0, 2})
	tree.Build()
	bad := newBDPTTestCamera(t, 1, 1)
	bad.Film.Shape = []int{1}
	if _, err := h.prepareBDPT(bad, bad.Film, tree); err == nil {
		t.Fatal("expected one-dimensional Film to fail BDPT preflight")
	}
	unsupported := nonBidirectionalCamera{Film: camera.NewFilm(1, 1)}
	if _, err := h.prepareBDPT(unsupported, unsupported.Film, tree); err == nil {
		t.Fatal("expected camera without bidirectional endpoint PDFs to fail")
	}
}

func TestBDPTPreflightAcceptsFiniteCylinderAreaLight(t *testing.T) {
	tree := (&object.ObjectTree{}).Build()
	tree.AddObject(&object.Object{
		Shape: shape.NewFiniteCylinder(
			mat.NewVecDense(3, []float64{0, 0, 2}),
			mat.NewVecDense(3, []float64{0, 0, 1}),
			0.25,
			3,
		),
		Material: &material.Material{Emission: emission.NewConstant(optics.ConstantSpectrum(4))},
	})
	tree.Build()

	cam := newBDPTTestCamera(t, 1, 1)
	state, err := newBDPTTestHandler().prepareBDPT(cam, cam.Film, tree)
	if err != nil {
		t.Fatalf("finite cylinder area light should pass BDPT preflight: %v", err)
	}
	if len(state.Lights) != 1 || state.TotalLightWeight <= 0 {
		t.Fatalf("finite cylinder was not collected as an area light: %+v", state)
	}
}

func TestBDPTUnsupportedSceneFailsWithoutChangingIntegrator(t *testing.T) {
	tree := (&object.ObjectTree{}).Build()
	tree.AddObject(&object.Object{
		Shape:    testLinearPolynomial(t, [3]float64{0, 0, 1}, -1),
		Material: &material.Material{Emission: emission.NewConstant(optics.ConstantSpectrum(2))},
	})
	tree.Build()

	camera3D := newBDPTTestCamera(t, 1, 1)
	h := newBDPTTestHandler()
	if err := h.TraceScene(bdptTarget(camera3D), tree, 1); err == nil {
		t.Fatal("BDPT must report the unsupported emitter")
	}
	if h.IntegratorKind != IntegratorBDPT || camera3D.Film.Samples != 0 {
		t.Fatalf("failed BDPT changed execution state: integrator=%q samples=%d", h.IntegratorKind, camera3D.Film.Samples)
	}
}

func TestBDPTPreflightAcceptsIdealTransmissionAndMediumBoundary(t *testing.T) {
	registry := medium.NewRegistry()
	glassID, err := registry.RegisterHomogeneous("glass", medium.NewConstant(1.5))
	if err != nil {
		t.Fatal(err)
	}
	tree := (&object.ObjectTree{Media: registry}).Build()
	addTestAreaLight(tree, []float64{0, 0, 3})
	tree.AddObject(&object.Object{
		Shape: shape.NewSphere(mat.NewVecDense(3, []float64{0, 0, 1}), 0.25),
		Material: &material.Material{Surface: bxdf.NewSpecularDielectricConstant(
			optics.ConstantSpectrum(1), optics.ConstantSpectrum(1), 1, 1.5,
		)},
		MediumBoundary: medium.NewBoundary(medium.MediumAir, glassID),
	})
	tree.Build()
	cam := newBDPTTestCamera(t, 1, 1)
	if _, err := newBDPTTestHandler().prepareBDPT(cam, cam.Film, tree); err != nil {
		t.Fatalf("ideal dielectric boundary should be supported: %v", err)
	}
}

func TestBDPTPreflightAcceptsReciprocalRoughReflection(t *testing.T) {
	tree := (&object.ObjectTree{}).Build()
	addTestAreaLight(tree, []float64{0, 0, 3})
	tree.AddObject(&object.Object{
		Shape: shape.NewSphere(mat.NewVecDense(3, []float64{0, 0, 1}), 0.25),
		Material: &material.Material{Surface: bsdf.NewWeightedMixture(
			bsdf.WeightedScattering{Weight: 0.5, Scattering: bxdf.NewLambert(optics.ConstantSpectrum(0.7))},
			bsdf.WeightedScattering{Weight: 0.5, Scattering: bxdf.NewRoughDielectricReflection(
				optics.ConstantSpectrum(1), 1, 1.5, 0.2,
			)},
		)},
	})
	tree.Build()
	cam := newBDPTTestCamera(t, 1, 1)
	if _, err := newBDPTTestHandler().prepareBDPT(cam, cam.Film, tree); err != nil {
		t.Fatalf("reciprocal rough reflection should be supported: %v", err)
	}
}

func TestBDPTRandomWalkTransmitsAndAbsorbsInsideMedium(t *testing.T) {
	registry := medium.NewRegistry()
	glassID, err := registry.RegisterHomogeneousWithCoefficients(
		"absorbing-glass", medium.NewConstant(1), medium.ConstantCoefficient(0.7),
	)
	if err != nil {
		t.Fatal(err)
	}
	tree := (&object.ObjectTree{Media: registry}).Build()
	boundary := medium.NewBoundary(medium.MediumAir, glassID)
	dielectric := &material.Material{Surface: bxdf.NewSpecularDielectricConstant(
		optics.ConstantSpectrum(1), optics.ConstantSpectrum(1), 1, 1,
	)}
	tree.AddObject(&object.Object{
		Shape: shape.NewTriangle(
			mat.NewVecDense(3, []float64{-2, -2, 1}),
			mat.NewVecDense(3, []float64{0, 2, 1}),
			mat.NewVecDense(3, []float64{2, -2, 1}),
		),
		Material: dielectric, MediumBoundary: boundary,
	})
	tree.AddObject(&object.Object{
		Shape: shape.NewTriangle(
			mat.NewVecDense(3, []float64{-2, -2, 2}),
			mat.NewVecDense(3, []float64{2, -2, 2}),
			mat.NewVecDense(3, []float64{0, 2, 2}),
		),
		Material: dielectric, MediumBoundary: boundary,
	})
	addTestAreaLight(tree, []float64{10, 0, 3})
	tree.Build()

	h := newBDPTTestHandler()
	ray := &optics.Ray{Space: h.Space}
	ray.Init()
	ray.SetSpectralWavelength(550)
	ray.Origin.CopyVec(mat.NewVecDense(3, []float64{0, 0, 0}))
	ray.Direction.CopyVec(mat.NewVecDense(3, []float64{0, 0, 1}))
	path := []bdptVertex{{
		Kind: bdptVertexCamera, Point: mat.NewVecDense(3, []float64{0, 0, 0}),
		Beta: 1, PDFFwdArea: 1, Connectible: true,
		MediumStack: medium.NewStack(medium.MediumAir),
	}}
	path = h.randomWalk(tree, ray, 1, 1, bxdf.TransportRadiance, 3, path)
	if len(path) < 3 {
		t.Fatalf("camera path did not cross both interfaces: %d vertices", len(path))
	}
	exit := path[2]
	if exit.Context.IncidentMedium != glassID || exit.Context.TransmitMedium != medium.MediumAir || exit.Context.Entering {
		t.Fatalf("incorrect exit medium context: incident=%d transmit=%d entering=%v",
			exit.Context.IncidentMedium, exit.Context.TransmitMedium, exit.Context.Entering)
	}
	want := math.Exp(-0.7)
	if got := exit.Beta; math.Abs(got-want) > 1e-10 {
		t.Fatalf("slab transmittance = %g, want %g", got, want)
	}
}

func TestBDPTBuildsRealCameraAndLightEndpoints(t *testing.T) {
	tree := (&object.ObjectTree{}).Build()
	tree.AddObject(&object.Object{
		Shape: shape.NewTriangle(
			mat.NewVecDense(3, []float64{-4, -4, 2}),
			mat.NewVecDense(3, []float64{0, 4, 2}),
			mat.NewVecDense(3, []float64{4, -4, 2}),
		),
		Material: &material.Material{Surface: bxdf.NewLambert(optics.ConstantSpectrum(0.8))},
	})
	addTestAreaLight(tree, []float64{2, 0, 1})
	tree.Build()
	h := newBDPTTestHandler()
	cam := newBDPTTestCamera(t, 1, 1)
	state, err := h.prepareBDPT(cam, cam.Film, tree)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	cameraPath := h.buildCameraSubpath(cam, cam.Film.Shape, tree, 550, optics.UniformWavelengthPDF(), 0, 0)
	lightPath := h.buildLightSubpath(tree, state.Lights, state.TotalLightWeight, 550, optics.UniformWavelengthPDF())
	if len(cameraPath) < 2 || cameraPath[0].Kind != bdptVertexCamera || cameraPath[0].Camera == nil {
		t.Fatalf("camera path has no real endpoint: %+v", cameraPath)
	}
	if len(lightPath) == 0 || lightPath[0].Kind != bdptVertexLight {
		t.Fatalf("light path has no real endpoint: %+v", lightPath)
	}
	if cameraPath[1].PDFFwdArea <= 0 {
		t.Fatalf("camera first-edge area PDF was not cached: %g", cameraPath[1].PDFFwdArea)
	}
}

func TestBDPTS0CameraHitEmissionIsEnumerated(t *testing.T) {
	tree := (&object.ObjectTree{}).Build()
	tree.AddObject(&object.Object{
		Shape: shape.NewTriangle(
			mat.NewVecDense(3, []float64{-3, -3, 2}),
			mat.NewVecDense(3, []float64{0, 3, 2}),
			mat.NewVecDense(3, []float64{3, -3, 2}),
		),
		Material: &material.Material{Emission: emission.NewConstant(optics.ConstantSpectrum(3))},
	})
	tree.Build()
	value := newBDPTTestHandler().traceBidirectionalSample(newBDPTTestCamera(t, 1, 1), tree, 550, optics.UniformWavelengthPDF(), 0, 0)
	if !validPower(value) {
		t.Fatalf("s=0 camera-hit emission did not contribute: %+v", value)
	}
}

func TestBDPTSupportedSceneRunsUnifiedSplatDriver(t *testing.T) {
	tree := (&object.ObjectTree{}).Build()
	tree.AddObject(&object.Object{
		Shape: shape.NewTriangle(
			mat.NewVecDense(3, []float64{-3, -3, 2}),
			mat.NewVecDense(3, []float64{0, 3, 2}),
			mat.NewVecDense(3, []float64{3, -3, 2}),
		),
		Material: &material.Material{Emission: emission.NewConstant(optics.ConstantSpectrum(3))},
	})
	tree.Build()
	cam := newBDPTTestCamera(t, 2, 2)
	h := newBDPTTestHandler()
	if err := h.TraceScene(bdptTarget(cam), tree, 4); err != nil {
		t.Fatalf("supported BDPT render failed: %v", err)
	}
	if cam.Film.Samples != 4 {
		t.Fatalf("BDPT sample accounting = %d, want 4", cam.Film.Samples)
	}
	positive := false
	for _, plane := range cam.Film.SpectralBins {
		for _, value := range plane.Data {
			positive = positive || value > 0
		}
	}
	if !positive {
		t.Fatal("supported BDPT render produced no radiance")
	}
}

func TestBDPTFiniteAreaLightConnectsToLambertCameraVertex(t *testing.T) {
	tree := (&object.ObjectTree{}).Build()
	tree.AddObject(&object.Object{
		Shape: shape.NewTriangle(
			mat.NewVecDense(3, []float64{-3, -3, 2}),
			mat.NewVecDense(3, []float64{3, -3, 2}),
			mat.NewVecDense(3, []float64{0, 3, 2}),
		),
		Material: &material.Material{Surface: bxdf.NewLambert(optics.ConstantSpectrum(0.8))},
	})
	addTestAreaLight(tree, []float64{1.5, 0, 1})
	tree.Build()
	value := newBDPTTestHandler().traceBidirectionalSample(newBDPTTestCamera(t, 1, 1), tree, 550, optics.UniformWavelengthPDF(), 0, 0)
	if !maths.IsFinite(value) || value < 0 {
		t.Fatalf("invalid BDPT contribution: %+v", value)
	}
}

func TestBDPTCachedMISWeightsSamePathAsPartition(t *testing.T) {
	cam := newBDPTTestCamera(t, 4, 4)
	cam.Position = mat.NewVecDense(3, []float64{1, 0, 1})
	cam.Coordinates = []*mat.VecDense{
		mat.NewVecDense(3, []float64{-1, 0, -1}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
		mat.NewVecDense(3, []float64{-1, 0, 1}),
	}
	if err := cam.Prepare(); err != nil {
		t.Fatal(err)
	}
	lightObject := &object.Object{
		Shape:    shape.NewCircle(mat.NewVecDense(3, []float64{0, 0, 2}), mat.NewVecDense(3, []float64{0, 0, -1}), 1),
		Material: &material.Material{Emission: emission.NewConstant(optics.ConstantSpectrum(1))},
	}
	surfaceObject := &object.Object{Material: &material.Material{
		Surface: bxdf.NewLambert(optics.ConstantSpectrum(0.8)),
	}}
	lightNormal := mat.NewVecDense(3, []float64{0, 0, -1})
	surfaceNormal := mat.NewVecDense(3, []float64{0, 0, 1})
	lightFrame, _ := maths.NewFrameFromNormal(lightNormal)
	surfaceFrame, _ := maths.NewFrameFromNormal(surfaceNormal)
	lightRoot := bdptVertex{
		Kind: bdptVertexLight, Point: mat.NewVecDense(3, []float64{0, 0, 2}),
		GeometricNormal: lightNormal, Frame: lightFrame, Object: lightObject,
		PDFFwdArea: 1 / math.Pi, Connectible: true,
	}
	cameraRoot := bdptVertex{
		Kind: bdptVertexCamera, Point: cam.Endpoint(), PDFFwdArea: 1,
		Connectible: true, Camera: cam,
	}
	surface := bdptVertex{
		Kind: bdptVertexSurface, Point: mat.NewVecDense(3, []float64{0, 0, 0}),
		GeometricNormal: surfaceNormal, Frame: surfaceFrame,
		Object: surfaceObject, Connectible: true,
	}
	surface.WoLocal = surfaceFrame.WorldToLocal(directionBetween(surface.Point, cam.Endpoint()))
	lightSurface := surface
	cameraSurface := surface
	lightSurface.PDFFwdArea = bdptVertexPDF(&lightRoot, nil, &lightSurface, cam)
	cameraSurface.PDFFwdArea = bdptVertexPDF(&cameraRoot, nil, &cameraSurface, cam)
	state := &bdptSceneState{
		Lights:           []areaLight{{Object: lightObject, Sampler: lightObject.Shape.(shape.SurfaceSampler), Area: math.Pi, Weight: math.Pi}},
		TotalLightWeight: math.Pi,
	}
	w12 := bdptMISWeight(state, cam, []bdptVertex{lightRoot}, []bdptVertex{cameraRoot, cameraSurface}, 1, 2)
	w21 := bdptMISWeight(state, cam, []bdptVertex{lightRoot, lightSurface}, []bdptVertex{cameraRoot}, 2, 1)
	cameraLight := lightRoot
	cameraLight.Kind = bdptVertexSurface
	cameraLight.PDFFwdArea = bdptVertexPDF(&cameraSurface, &cameraRoot, &cameraLight, cam)
	w03 := bdptMISWeight(state, cam, nil, []bdptVertex{cameraRoot, cameraSurface, cameraLight}, 0, 3)
	if w03 <= 0 || w12 <= 0 || w21 <= 0 || math.Abs(w03+w12+w21-1) > 1e-10 {
		t.Fatalf("same-path MIS weights = [%g %g %g], sum=%g", w03, w12, w21, w03+w12+w21)
	}
}

func TestBDPTMISPartitionAcrossRandomizedDirectPaths(t *testing.T) {
	rng := randv2.New(randv2.NewPCG(0x2147483647, 0x5eed))
	lambertObject := &object.Object{Material: &material.Material{
		Surface: bxdf.NewLambert(optics.ConstantSpectrum(0.8)),
	}}
	for sample := 0; sample < 128; sample++ {
		lightPoint := mat.NewVecDense(3, []float64{
			-0.75 + 1.5*rng.Float64(),
			-0.25 + 0.5*rng.Float64(),
			1.5 + rng.Float64(),
		})
		cameraPoint := mat.NewVecDense(3, []float64{
			0.5 + rng.Float64(),
			-0.25 + 0.5*rng.Float64(),
			0.5 + rng.Float64(),
		})
		surfacePoint := mat.NewVecDense(3, []float64{0, 0, 0})
		toLight := directionBetween(surfacePoint, lightPoint)
		toCamera := directionBetween(surfacePoint, cameraPoint)
		surfaceNormal := mat.NewVecDense(3, nil)
		surfaceNormal.AddVec(toLight, toCamera)
		surfaceNormal.ScaleVec(1/mat.Norm(surfaceNormal, 2), surfaceNormal)
		lightNormal := negated(toLight)
		surfaceFrame, _ := maths.NewFrameFromNormal(surfaceNormal)
		lightFrame, _ := maths.NewFrameFromNormal(lightNormal)

		cam := camera.NewCamera3D()
		cam.Position = cameraPoint
		cam.Coordinates = []*mat.VecDense{
			directionBetween(cameraPoint, surfacePoint),
			mat.NewVecDense(3, []float64{0, 1, 0}),
			mat.NewVecDense(3, []float64{1, 0, -1}),
		}
		cam.FieldOfViews = []float64{45, 45}
		if err := cam.Prepare(); err != nil {
			t.Fatalf("sample %d camera: %v", sample, err)
		}

		radius := 0.2 + 0.3*rng.Float64()
		lightShape := shape.NewCircle(lightPoint, lightNormal, radius)
		lightObject := &object.Object{
			Shape:    lightShape,
			Material: &material.Material{Emission: emission.NewConstant(optics.ConstantSpectrum(1))},
		}
		area := lightShape.SurfaceArea()
		lightRoot := bdptVertex{
			Kind: bdptVertexLight, Point: lightPoint, GeometricNormal: lightNormal,
			Frame: lightFrame, Object: lightObject, PDFFwdArea: 1 / area,
			Connectible: true,
		}
		cameraRoot := bdptVertex{
			Kind: bdptVertexCamera, Point: cameraPoint, PDFFwdArea: 1,
			Connectible: true, Camera: cam,
		}
		surface := bdptVertex{
			Kind: bdptVertexSurface, Point: surfacePoint, GeometricNormal: surfaceNormal,
			Frame: surfaceFrame, WoLocal: surfaceFrame.WorldToLocal(toCamera),
			Object: lambertObject, Connectible: true,
		}
		lightSurface, cameraSurface := surface, surface
		lightSurface.PDFFwdArea = bdptVertexPDF(&lightRoot, nil, &lightSurface, cam)
		cameraSurface.PDFFwdArea = bdptVertexPDF(&cameraRoot, nil, &cameraSurface, cam)
		cameraLight := lightRoot
		cameraLight.Kind = bdptVertexSurface
		cameraLight.PDFFwdArea = bdptVertexPDF(&cameraSurface, &cameraRoot, &cameraLight, cam)
		state := &bdptSceneState{
			Lights:           []areaLight{{Object: lightObject, Sampler: lightShape, Area: area, Weight: area}},
			TotalLightWeight: area,
		}

		weights := []float64{
			bdptMISWeight(state, cam, nil, []bdptVertex{cameraRoot, cameraSurface, cameraLight}, 0, 3),
			bdptMISWeight(state, cam, []bdptVertex{lightRoot}, []bdptVertex{cameraRoot, cameraSurface}, 1, 2),
			bdptMISWeight(state, cam, []bdptVertex{lightRoot, lightSurface}, []bdptVertex{cameraRoot}, 2, 1),
		}
		sum := weights[0] + weights[1] + weights[2]
		if math.Abs(sum-1) > 1e-10 {
			t.Fatalf("sample %d MIS partition = %v, sum=%g", sample, weights, sum)
		}
	}
}

func TestBDPTDeltaOnlyDisablesAdjacentStrategies(t *testing.T) {
	light := []bdptVertex{
		{Kind: bdptVertexLight, PDFFwdArea: 1, PDFRevArea: 1},
		{Kind: bdptVertexSurface, PDFFwdArea: 0, PDFRevArea: 0, SampledDelta: true},
		{Kind: bdptVertexSurface, PDFFwdArea: 0, PDFRevArea: 1, Connectible: true},
	}
	cameraPath := []bdptVertex{{Kind: bdptVertexCamera, PDFFwdArea: 1, Connectible: true}}
	weight := bdptMISWeight(nil, nil, light, cameraPath, 3, 1)
	if weight <= 0 || math.IsNaN(weight) {
		t.Fatalf("a generated path containing an earlier Delta event was rejected: %g", weight)
	}
}

func TestBDPTUnifiedT1ProjectsPathAfterDeltaEvent(t *testing.T) {
	cam := newBDPTTestCamera(t, 4, 4)
	rootNormal := mat.NewVecDense(3, []float64{0, 0, 1})
	rootFrame, _ := maths.NewFrameFromNormal(rootNormal)
	rootObject := &object.Object{
		Shape:    shape.NewCircle(mat.NewVecDense(3, []float64{1, 0, 0}), rootNormal, 0.2),
		Material: &material.Material{Emission: emission.NewConstant(optics.ConstantSpectrum(1))},
	}
	root := bdptVertex{
		Kind: bdptVertexLight, Point: mat.NewVecDense(3, []float64{1, 0, 0}),
		GeometricNormal: rootNormal, Frame: rootFrame, Object: rootObject,
		Context: bxdf.ShadingContext{WavelengthNM: 550, WavelengthsNM: []float64{550}},
		Beta:    1, PDFFwdArea: 1, Connectible: true,
		MediumStack: medium.NewStack(medium.MediumAir),
	}
	delta := bdptVertex{
		Kind: bdptVertexSurface, Point: mat.NewVecDense(3, []float64{1, 0, 1}),
		PDFFwdArea: 0, PDFRevArea: 0, SampledDelta: true,
	}
	screenNormal := mat.NewVecDense(3, []float64{0, 0, -1})
	screenFrame, _ := maths.NewFrameFromNormal(screenNormal)
	screenObject := &object.Object{Material: &material.Material{
		Surface: bxdf.NewLambert(optics.ConstantSpectrum(0.8)),
	}}
	screen := bdptVertex{
		Kind: bdptVertexSurface, Point: mat.NewVecDense(3, []float64{0, 0, 2}),
		GeometricNormal: screenNormal, Frame: screenFrame,
		WoLocal: screenFrame.WorldToLocal(directionBetween(
			mat.NewVecDense(3, []float64{0, 0, 2}),
			mat.NewVecDense(3, []float64{1, 0, 1}),
		)),
		Context: bxdf.ShadingContext{WavelengthNM: 550, WavelengthsNM: []float64{550}},
		Object:  screenObject, Beta: 1,
		PDFFwdArea: 1, Connectible: true,
		MediumStack: medium.NewStack(medium.MediumAir),
	}
	cameraRoot := bdptVertex{
		Kind: bdptVertexCamera, Point: cam.Endpoint(), PDFFwdArea: 1,
		Connectible: true, Camera: cam, MediumStack: medium.NewStack(medium.MediumAir),
	}
	lightPath := []bdptVertex{root, delta, screen}
	cameraPath := []bdptVertex{cameraRoot}
	value, _, isSplat, ok := newBDPTTestHandler().connectBDPTStrategy(
		&bdptSceneState{}, cam, []int{4, 4}, nil, lightPath, cameraPath, 3, 1,
	)
	if !ok || !isSplat || !validPower(value) {
		t.Fatalf("unified t=1 strategy failed after Delta event: value=%+v splat=%v ok=%v", value, isSplat, ok)
	}
	if weight := bdptMISWeight(nil, cam, lightPath, cameraPath, 3, 1); weight <= 0 {
		t.Fatalf("unified t=1 MIS rejected Delta path: %g", weight)
	}
}

func TestBDPTSegmentMediumUsesOutgoingSide(t *testing.T) {
	inside := medium.MediumID(7)
	vertex := bdptVertex{
		Kind:            bdptVertexSurface,
		GeometricNormal: mat.NewVecDense(3, []float64{0, 0, 1}),
		Object:          &object.Object{MediumBoundary: medium.NewBoundary(medium.MediumAir, inside)},
		MediumStack:     medium.NewStack(medium.MediumAir),
	}
	if got := bdptSegmentMedium(&vertex, mat.NewVecDense(3, []float64{0, 0, -1})); got != inside {
		t.Fatalf("inward segment medium = %d, want %d", got, inside)
	}
	if got := bdptSegmentMedium(&vertex, mat.NewVecDense(3, []float64{0, 0, 1})); got != medium.MediumAir {
		t.Fatalf("outward segment medium = %d, want air", got)
	}
}

func TestBDPTReconstructionFilterDoesNotRenormalizeAtCrop(t *testing.T) {
	splat := FilmSplat{
		Value:      8,
		projection: camera.FilmProjection{Position: []float64{1.25, 1.75}},
	}
	fullMask := make([]bool, 16)
	for i := range fullMask {
		fullMask[i] = true
	}
	full := filterBDPTSplat(splat, 4, 4, fullMask)
	if len(full) != 1 {
		t.Fatalf("box-filtered splat count = %d, want 1", len(full))
	}
	var fullSum float64
	for _, item := range full {
		fullSum += item.Value
	}
	if math.Abs(fullSum-8) > 1e-12 {
		t.Fatalf("full filter energy = %g, want 8", fullSum)
	}

	cropMask := make([]bool, 16)
	cropMask[2*4+1] = true
	cropped := filterBDPTSplat(splat, 4, 4, cropMask)
	if len(cropped) != 1 {
		t.Fatalf("cropped splat count = %d, want 1", len(cropped))
	}
	if got := cropped[0].Value; math.Abs(got-8) > 1e-12 {
		t.Fatalf("box-filtered crop value = %g, want 8", got)
	}
	cropMask[2*4+1] = false
	cropMask[1*4+1] = true
	if got := filterBDPTSplat(splat, 4, 4, cropMask); len(got) != 0 {
		t.Fatalf("splat outside active crop was redistributed: %+v", got)
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
		t.Fatalf("invalid light distribution: count=%d weight=%g", len(lights), totalWeight)
	}
	if ratio := lights[1].Weight / lights[0].Weight; math.Abs(ratio-10) > 1e-12 {
		t.Fatalf("power weight ratio = %g, want 10", ratio)
	}
}
