package camera

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"gonum.org/v1/gonum/mat"
)

func TestCamera3DProjectPointMapsViewCenter(t *testing.T) {
	camera := &Camera3D{
		Position:     mat.NewVecDense(3, []float64{0, 0, 0}),
		Coordinates:  testCameraCoordinates([]float64{0, 0, -1}, []float64{0, 1, 0}),
		FieldOfViews: []float64{60, 90},
	}
	film := NewFilm(100, 50)
	camera.Film = film
	projection, ok := camera.ProjectPoint(mat.NewVecDense(3, []float64{0, 0, -2}))
	if !ok {
		t.Fatal("view-center point did not project")
	}
	pixel, ok := PixelIndex(projection.Position[0], projection.Position[1], 100, 50)
	if !ok || pixel != 25*100+50 {
		t.Fatalf("center pixel = %d (valid=%v), want %d", pixel, ok, 25*100+50)
	}
	if math.Abs(projection.Position[0]-49.5) > 1e-12 || math.Abs(projection.Position[1]-24.5) > 1e-12 {
		t.Fatalf("center raster position = (%g, %g), want (49.5, 24.5)", projection.Position[0], projection.Position[1])
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
		Coordinates:  testCameraCoordinates([]float64{0, 0, -1}, []float64{0, 1, 0}),
		FieldOfViews: []float64{60, 90},
	}
	film := NewFilm(100, 50)
	camera.Film = film
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
		Coordinates:  testCameraCoordinates([]float64{-4.1, 5.7, -1.63}, []float64{0, 0, 1}),
		FieldOfViews: []float64{48, 60},
	}

	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	for name, axis := range map[string]*mat.VecDense{
		"direction": camera.orthonormalCoordinates[0],
		"right":     camera.orthonormalCoordinates[1],
		"up":        camera.orthonormalCoordinates[2],
	} {
		if got := mat.Norm(axis, 2); math.Abs(got-1) > 1e-12 {
			t.Errorf("%s norm = %g, want 1", name, got)
		}
	}
	for name, got := range map[string]float64{
		"direction dot up":    mat.Dot(camera.orthonormalCoordinates[0], camera.orthonormalCoordinates[2]),
		"direction dot right": mat.Dot(camera.orthonormalCoordinates[0], camera.orthonormalCoordinates[1]),
		"up dot right":        mat.Dot(camera.orthonormalCoordinates[2], camera.orthonormalCoordinates[1]),
	} {
		if math.Abs(got) > 1e-12 {
			t.Errorf("%s = %g, want 0", name, got)
		}
	}
}

func TestCamera3DProjectPointMapsOpticalAxisToCenterWithWorldUp(t *testing.T) {
	camera := &Camera3D{
		Position:     mat.NewVecDense(3, []float64{3.35, -5.65, 2.25}),
		Coordinates:  testCameraCoordinates([]float64{-4.1, 5.7, -1.63}, []float64{0, 0, 1}),
		FieldOfViews: []float64{48, 60},
	}
	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	point := mat.VecDenseCopyOf(camera.Position)
	point.AddScaledVec(point, 4, camera.orthonormalCoordinates[0])
	film := NewFilm(800, 800)
	camera.Film = film
	projection, ok := camera.ProjectPoint(point)
	if !ok {
		t.Fatal("point on optical axis did not project")
	}
	want := (800/2)*800 + 800/2
	pixel, ok := PixelIndex(projection.Position[0], projection.Position[1], 800, 800)
	if !ok || pixel != want {
		t.Fatalf("optical-axis pixel = %d (valid=%v), want %d", pixel, ok, want)
	}
}

func TestCamera3DGenerateRayProjectPointRoundTripWithWorldUp(t *testing.T) {
	camera := &Camera3D{
		Position:     mat.NewVecDense(3, []float64{3.35, -5.65, 2.25}),
		Coordinates:  testCameraCoordinates([]float64{-4.1, 5.7, -1.63}, []float64{0, 0, 1}),
		FieldOfViews: []float64{48, 60},
	}
	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	film := NewFilm(80, 80)
	camera.Film = film

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
			want := y*80 + x
			projectedPixel, ok := PixelIndex(projection.Position[0], projection.Position[1], 80, 80)
			if !ok || projectedPixel != want {
				t.Fatalf("ray from pixel (%d, %d) projected to %d (valid=%v), want %d", x, y, projectedPixel, ok, want)
			}
		}
	}
}

func testCameraCoordinates(direction, up []float64) []*mat.VecDense {
	forward := mat.NewVecDense(3, direction)
	right := maths.Cross2(forward, mat.NewVecDense(3, up))
	return []*mat.VecDense{forward, right, mat.NewVecDense(3, up)}
}

func TestPixelIndexDerivesPixelsFromCenterCoordinates(t *testing.T) {
	for _, test := range []struct {
		name     string
		position [2]float64
		want     int
	}{
		{name: "left top boundary", position: [2]float64{-0.5, -0.5}, want: 0},
		{name: "first center", position: [2]float64{0, 0}, want: 0},
		{name: "even film center", position: [2]float64{49.5, 24.5}, want: 25*100 + 50},
		{name: "last center", position: [2]float64{99, 49}, want: 49*100 + 99},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := PixelIndex(test.position[0], test.position[1], 100, 50)
			if !ok || got != test.want {
				t.Fatalf("PixelIndex = %d (valid=%v), want %d", got, ok, test.want)
			}
		})
	}
	if _, ok := PixelIndex(99.5, 0, 100, 50); ok {
		t.Fatal("right Film boundary must be outside")
	}
}
