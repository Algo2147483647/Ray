package camera

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"testing"
)

func TestCamera3DPrepareCachesDerivedData(t *testing.T) {
	camera := NewCamera3D()
	camera.Position = mat.NewVecDense(3, []float64{0, 0, 0})
	camera.Coordinates = testCameraCoordinates([]float64{1, 0, 0}, []float64{0, 0, 1})
	camera.FieldOfViews = []float64{60, 67.38013505195957}

	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if len(camera.orthonormalCoordinates) != 3 {
		t.Fatal("expected cached camera basis vectors")
	}
	if math.Abs(camera.halfHeight-math.Tan(60*math.Pi/180/2.0)) > 1e-12 {
		t.Fatalf("unexpected cached half height: %v", camera.halfHeight)
	}
	if math.Abs(camera.halfWidth-math.Tan(67.38013505195957*math.Pi/180/2.0)) > 1e-12 {
		t.Fatalf("unexpected cached half width: %v", camera.halfWidth)
	}

	cachedRight := camera.orthonormalCoordinates[1]
	camera.Film = NewFilm(100, 50)
	ray := camera.GenerateRay(nil, 5, 5)
	if ray == nil {
		t.Fatal("expected ray to be generated")
	}
	if camera.orthonormalCoordinates[1] != cachedRight {
		t.Fatal("expected camera basis cache to be reused")
	}
}

func TestCamera3DGenerateRayUsesFilmDimensions(t *testing.T) {
	camera := NewCamera3D()
	camera.Position = mat.NewVecDense(3, []float64{0, 0, 0})
	camera.Coordinates = testCameraCoordinates([]float64{1, 0, 0}, []float64{0, 0, 1})
	camera.FieldOfViews = []float64{60, 67.38013505195957}

	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	camera.Film = NewFilm(100, 50)
	if ray := camera.GenerateRay(nil, 50, 25); ray == nil {
		t.Fatal("expected ray from first film")
	}
	camera.Film = NewFilm(200, 100)
	if ray := camera.GenerateRay(nil, 100, 50); ray == nil {
		t.Fatal("expected ray from second film")
	}
}
