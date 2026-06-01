package camera

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"gonum.org/v1/gonum/mat"
)

func TestHyperbolicCameraPrepareBuildsKleinOrthonormalFrame(t *testing.T) {
	camera := NewHyperbolicCamera()
	camera.Position = mat.NewVecDense(3, []float64{-0.88, 0, 0.10})
	camera.Direction = mat.NewVecDense(3, []float64{1.8, 0, -0.02})
	camera.Up = mat.NewVecDense(3, []float64{0, 0, 1})
	camera.Width = 100
	camera.Height = 100
	camera.AspectRatio = 1
	camera.FieldOfView = 80

	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	g := geometry.Klein()
	assertKleinInnerApprox(t, g, camera.Position, camera.dir, camera.dir, 1, 1e-12)
	assertKleinInnerApprox(t, g, camera.Position, camera.up, camera.up, 1, 1e-12)
	assertKleinInnerApprox(t, g, camera.Position, camera.right, camera.right, 1, 1e-12)
	assertKleinInnerApprox(t, g, camera.Position, camera.dir, camera.up, 0, 1e-12)
	assertKleinInnerApprox(t, g, camera.Position, camera.dir, camera.right, 0, 1e-12)
	assertKleinInnerApprox(t, g, camera.Position, camera.up, camera.right, 0, 1e-12)
}

func TestHyperbolicCameraGenerateRayUsesKleinUnitDirection(t *testing.T) {
	camera := NewHyperbolicCamera()
	camera.Position = mat.NewVecDense(3, []float64{-0.7, 0.2, 0.1})
	camera.Direction = mat.NewVecDense(3, []float64{1, -0.1, 0})
	camera.Up = mat.NewVecDense(3, []float64{0, 0, 1})
	camera.Width = 64
	camera.Height = 64
	camera.AspectRatio = 1
	camera.FieldOfView = 70

	ray := camera.GenerateRay(nil, 32, 32)

	if ray.Geometry != geometry.Klein() {
		t.Fatal("expected generated ray to carry Klein geometry")
	}
	assertKleinInnerApprox(t, geometry.Klein(), ray.Origin, ray.Direction, ray.Direction, 1, 1e-12)
}

func TestHyperbolicCameraRejectsPositionOutsideKleinBall(t *testing.T) {
	camera := NewHyperbolicCamera()
	camera.Position = mat.NewVecDense(3, []float64{1.1, 0, 0})
	camera.Direction = mat.NewVecDense(3, []float64{1, 0, 0})
	camera.Up = mat.NewVecDense(3, []float64{0, 0, 1})
	camera.Width = 64
	camera.Height = 64
	camera.AspectRatio = 1
	camera.FieldOfView = 70

	if err := camera.Prepare(); err == nil {
		t.Fatal("expected camera outside Klein unit ball to be rejected")
	}
}

func assertKleinInnerApprox(
	t *testing.T,
	g geometry.Geometry,
	p, u, v *mat.VecDense,
	want, tolerance float64,
) {
	t.Helper()
	got := g.InnerProduct(p, u, v)
	if math.Abs(got-want) > tolerance {
		t.Fatalf("unexpected Klein inner product: got %.15f want %.15f", got, want)
	}
}
