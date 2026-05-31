package camera

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
)

func TestCameraNDimGenerateRay3D(t *testing.T) {
	camera := NewCameraNDim()
	camera.Position = mat.NewVecDense(3, []float64{0, 0, 0})
	camera.Coordinates = []*mat.VecDense{
		mat.NewVecDense(3, []float64{1, 0, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
	}
	camera.Width = []int{100, 100}
	camera.FieldOfView = []float64{90, 90}

	ray := camera.GenerateRay(nil, 50, 50)
	if ray == nil {
		t.Fatal("expected ray to be generated")
	}
	if !mat.Equal(ray.Origin, camera.Position) {
		t.Fatal("expected ray origin to match camera position")
	}
	if norm := mat.Norm(ray.Direction, 2); math.Abs(norm-1.0) > 1e-10 {
		t.Fatalf("expected normalized direction, got norm %f", norm)
	}
	if ray.WaveLength != 0 {
		t.Fatalf("expected camera to leave wavelength sampling to the renderer, got %f", ray.WaveLength)
	}
	if ray.WavelengthPDF != 0 {
		t.Fatalf("expected camera ray wavelength pdf to remain unset, got %f", ray.WavelengthPDF)
	}
	for i := 0; i < 3; i++ {
		if ray.Color[i] != 1 {
			t.Fatalf("expected camera ray throughput to start white, got %v", ray.Color)
		}
	}
}

func TestCameraNDimGenerateRay4D(t *testing.T) {
	camera := NewCameraNDim()
	camera.Position = mat.NewVecDense(4, []float64{0, 0, 0, 0})
	camera.Coordinates = []*mat.VecDense{
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		mat.NewVecDense(4, []float64{0, 0, 1, 0}),
		mat.NewVecDense(4, []float64{0, 0, 0, 1}),
	}
	camera.Width = []int{10, 10, 10}
	camera.FieldOfView = []float64{90, 90, 90}

	ray := camera.GenerateRay(nil, 5, 5, 5)
	if ray == nil {
		t.Fatal("expected ray to be generated")
	}
	if ray.Origin.Len() != 4 {
		t.Fatalf("expected 4D origin, got %dD", ray.Origin.Len())
	}
	if ray.Direction.Len() != 4 {
		t.Fatalf("expected 4D direction, got %dD", ray.Direction.Len())
	}
	if norm := mat.Norm(ray.Direction, 2); math.Abs(norm-1.0) > 1e-10 {
		t.Fatalf("expected normalized direction, got norm %f", norm)
	}
}

func TestCameraNDimOrthoKeepsDirectionAndMovesOrigin(t *testing.T) {
	camera := NewCameraNDim()
	camera.Position = mat.NewVecDense(4, []float64{0, 0, 0, 0})
	camera.Coordinates = []*mat.VecDense{
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		mat.NewVecDense(4, []float64{0, 0, 1, 0}),
		mat.NewVecDense(4, []float64{0, 0, 0, 1}),
	}
	camera.Width = []int{10, 10, 10}
	camera.FieldOfView = []float64{90, 90, 90}
	camera.Ortho = true

	rayA := camera.GenerateRay(nil, 0, 0, 0)
	rayB := camera.GenerateRay(nil, 9, 9, 9)

	assertVecApprox(t, rayA.Direction, rayB.Direction, 1e-12)
	assertVecApprox(t, rayA.Direction, mat.NewVecDense(4, []float64{1, 0, 0, 0}), 1e-12)
	if mat.EqualApprox(rayA.Origin, rayB.Origin, 1e-6) {
		t.Fatalf("expected orthographic camera to move ray origins across the film")
	}
	if math.Abs(rayA.Origin.AtVec(0)) > 1e-12 || math.Abs(rayB.Origin.AtVec(0)) > 1e-12 {
		t.Fatalf("expected orthographic film offsets to stay on camera plane, got %v and %v", rayA.Origin.RawVector().Data, rayB.Origin.RawVector().Data)
	}
}

func TestCameraNDimGenerateRayResetsReusedRayMediumState(t *testing.T) {
	camera := NewCameraNDim()
	camera.Position = mat.NewVecDense(3, []float64{0, 0, 0})
	camera.Coordinates = []*mat.VecDense{
		mat.NewVecDense(3, []float64{1, 0, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
	}
	camera.Width = []int{100, 100}
	camera.FieldOfView = []float64{90, 90}

	ray := &optics.Ray{}
	ray.Init()
	ray.MediumStack.Push(medium.MediumID(42))
	ray.SetSpectralWavelength(610)

	camera.GenerateRay(ray, 50, 50)

	if got := ray.MediumStack.Current(); got != medium.MediumAir {
		t.Fatalf("expected GenerateRay to reset medium stack to air, got %v", got)
	}
	if ray.WaveLength != 0 || ray.WavelengthPDF != 0 {
		t.Fatalf("expected GenerateRay to reset spectral state, got wavelength=%f pdf=%f", ray.WaveLength, ray.WavelengthPDF)
	}
}

func assertVecApprox(t *testing.T, got, want *mat.VecDense, tolerance float64) {
	t.Helper()
	if got.Len() != want.Len() {
		t.Fatalf("length mismatch: got %d want %d", got.Len(), want.Len())
	}
	for i := 0; i < got.Len(); i++ {
		if math.Abs(got.AtVec(i)-want.AtVec(i)) > tolerance {
			t.Fatalf("component %d: got %f want %f", i, got.AtVec(i), want.AtVec(i))
		}
	}
}
