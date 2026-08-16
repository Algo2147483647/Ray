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

func TestCamera3DPrepareOrthogonalizesWorldUp(t *testing.T) {
	camera := &Camera3D{
		Position:     mat.NewVecDense(3, []float64{3.35, -5.65, 2.25}),
		Direction:    mat.NewVecDense(3, []float64{-4.1, 5.7, -1.63}),
		Up:           mat.NewVecDense(3, []float64{0, 0, 1}),
		Width:        800,
		Height:       800,
		FieldOfViews: []float64{48, 60},
	}

	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	for name, axis := range map[string]*mat.VecDense{
		"direction": camera.dir,
		"up":        camera.up,
		"right":     camera.right,
	} {
		if got := mat.Norm(axis, 2); math.Abs(got-1) > 1e-12 {
			t.Errorf("%s norm = %g, want 1", name, got)
		}
	}
	for name, got := range map[string]float64{
		"direction dot up":    mat.Dot(camera.dir, camera.up),
		"direction dot right": mat.Dot(camera.dir, camera.right),
		"up dot right":        mat.Dot(camera.up, camera.right),
	} {
		if math.Abs(got) > 1e-12 {
			t.Errorf("%s = %g, want 0", name, got)
		}
	}
}

func TestCamera3DProjectPointMapsOpticalAxisToCenterWithWorldUp(t *testing.T) {
	camera := &Camera3D{
		Position:     mat.NewVecDense(3, []float64{3.35, -5.65, 2.25}),
		Direction:    mat.NewVecDense(3, []float64{-4.1, 5.7, -1.63}),
		Up:           mat.NewVecDense(3, []float64{0, 0, 1}),
		Width:        800,
		Height:       800,
		FieldOfViews: []float64{48, 60},
	}
	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	point := mat.VecDenseCopyOf(camera.Position)
	point.AddScaledVec(point, 4, camera.dir)
	projection, ok := camera.ProjectPoint(point)
	if !ok {
		t.Fatal("point on optical axis did not project")
	}
	want := (camera.Height/2)*camera.Width + camera.Width/2
	if projection.Pixel != want {
		t.Fatalf("optical-axis pixel = %d, want %d", projection.Pixel, want)
	}
}

func TestCamera3DGenerateRayProjectPointRoundTripWithWorldUp(t *testing.T) {
	camera := &Camera3D{
		Position:     mat.NewVecDense(3, []float64{3.35, -5.65, 2.25}),
		Direction:    mat.NewVecDense(3, []float64{-4.1, 5.7, -1.63}),
		Up:           mat.NewVecDense(3, []float64{0, 0, 1}),
		Width:        80,
		Height:       80,
		FieldOfViews: []float64{48, 60},
	}
	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	for _, pixel := range [][2]int{{0, 0}, {79, 0}, {0, 79}, {79, 79}, {40, 40}, {17, 63}} {
		x, y := pixel[0], pixel[1]
		for sample := 0; sample < 16; sample++ {
			ray := camera.GenerateRay(nil, x, y)
			point := mat.VecDenseCopyOf(ray.Origin)
			point.AddScaledVec(point, 3, ray.Direction)
			projection, ok := camera.ProjectPoint(point)
			if !ok {
				t.Fatalf("ray from pixel (%d, %d) did not project", x, y)
			}
			want := y*camera.Width + x
			if projection.Pixel != want {
				t.Fatalf("ray from pixel (%d, %d) projected to %d, want %d", x, y, projection.Pixel, want)
			}
		}
	}
}
