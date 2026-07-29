package ray_tracing

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/emission"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
)

type fixedCamera struct{}

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
	h.Integrator = IntegratorBDPT
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
	h.Integrator = IntegratorBDPT
	h.SceneGeometry = geometry.Euclidean()
	h.SpectrumMode = optics.SpectrumModeRGB
	value := h.traceBidirectionalSample(fixedCamera{}, tree, 0, 0, 0, 0)
	if value.IsZero() {
		t.Fatal("fallback path tracer should see the unsupported infinite emitter")
	}
}

var _ camera.Camera = fixedCamera{}
