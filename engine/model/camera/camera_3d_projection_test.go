package camera

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestCamera3DProjectPointMapsViewCenter(t *testing.T) {
	camera := &Camera3D{
		Position:     mat.NewVecDense(3, []float64{0, 0, 0}),
		Direction:    mat.NewVecDense(3, []float64{0, 0, -1}),
		Up:           mat.NewVecDense(3, []float64{0, 1, 0}),
		Width:        100,
		Height:       50,
		FieldOfViews: []float64{60, 90},
	}
	projection, ok := camera.ProjectPoint(mat.NewVecDense(3, []float64{0, 0, -2}))
	if !ok {
		t.Fatal("view-center point did not project")
	}
	if projection.Pixel != 25*100+50 {
		t.Fatalf("center pixel = %d, want %d", projection.Pixel, 25*100+50)
	}
	if math.Abs(projection.Distance-2) > 1e-12 {
		t.Fatalf("distance = %g, want 2", projection.Distance)
	}
	if got := mat.Dot(projection.ToCamera, mat.NewVecDense(3, []float64{0, 0, 1})); math.Abs(got-1) > 1e-12 {
		t.Fatalf("to-camera direction dot = %g, want 1", got)
	}
}

func TestCamera3DProjectPointRejectsOutsideFilm(t *testing.T) {
	camera := &Camera3D{
		Position:     mat.NewVecDense(3, []float64{0, 0, 0}),
		Direction:    mat.NewVecDense(3, []float64{0, 0, -1}),
		Up:           mat.NewVecDense(3, []float64{0, 1, 0}),
		Width:        100,
		Height:       50,
		FieldOfViews: []float64{60, 90},
	}
	if _, ok := camera.ProjectPoint(mat.NewVecDense(3, []float64{3, 0, -2})); ok {
		t.Fatal("point outside horizontal FOV projected onto film")
	}
	if _, ok := camera.ProjectPoint(mat.NewVecDense(3, []float64{0, 0, 1})); ok {
		t.Fatal("point behind camera projected onto film")
	}
}
