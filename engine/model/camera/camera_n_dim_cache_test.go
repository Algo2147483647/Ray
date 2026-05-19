package camera

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"testing"
)

func TestCameraNDimPrepareCachesDerivedData(t *testing.T) {
	camera := NewCameraNDim()
	camera.Position = mat.NewVecDense(4, []float64{0, 0, 0, 0})
	camera.Coordinates = []*mat.VecDense{
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		mat.NewVecDense(4, []float64{0, 0, 1, 0}),
		mat.NewVecDense(4, []float64{0, 0, 0, 1}),
	}
	camera.Width = []int{10, 10, 10}
	camera.FieldOfView = []float64{45, 60, 90}

	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if len(camera.orthonormalCoordinates) != 4 {
		t.Fatalf("expected 4 cached basis vectors, got %d", len(camera.orthonormalCoordinates))
	}
	if len(camera.fovTangents) != 3 {
		t.Fatalf("expected 3 cached fov tangents, got %d", len(camera.fovTangents))
	}
	if math.Abs(camera.fovTangents[0]-math.Tan(45*math.Pi/180/2.0)) > 1e-12 {
		t.Fatalf("unexpected cached tangent: %v", camera.fovTangents[0])
	}

	cachedBasis := camera.orthonormalCoordinates
	cachedTangents := camera.fovTangents

	ray := camera.GenerateRay(nil, 5, 5, 5)
	if ray == nil {
		t.Fatal("expected ray to be generated")
	}
	if camera.orthonormalCoordinates[0] != cachedBasis[0] {
		t.Fatal("expected orthonormal basis cache to be reused")
	}
	if &camera.fovTangents[0] != &cachedTangents[0] {
		t.Fatal("expected field-of-view tangent cache to be reused")
	}
}
