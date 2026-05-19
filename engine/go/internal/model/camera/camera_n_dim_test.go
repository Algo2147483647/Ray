package camera

import (
	"math"
	"testing"

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
		if ray.Color.AtVec(i) != 1 {
			t.Fatalf("expected camera ray throughput to start white, got %v", ray.Color.RawVector().Data)
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
