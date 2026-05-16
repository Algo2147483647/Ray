package camera

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/go/internal/model/optics"
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
	if ray.WaveLength <= optics.WavelengthMin || ray.WaveLength >= optics.WavelengthMax {
		t.Fatalf("expected camera ray wavelength inside visible range, got %f", ray.WaveLength)
	}
	if math.Abs(ray.WavelengthPDF-optics.UniformWavelengthPDF()) > 1e-12 {
		t.Fatalf("unexpected wavelength pdf: got %f", ray.WavelengthPDF)
	}
	for i := 0; i < 3; i++ {
		if math.IsNaN(ray.Color.AtVec(i)) || math.IsInf(ray.Color.AtVec(i), 0) || ray.Color.AtVec(i) < 0 {
			t.Fatalf("expected finite non-negative spectral reconstruction color, got %v", ray.Color.RawVector().Data)
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
