package camera

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"testing"
)

func TestCamera3DPrepareCachesDerivedData(t *testing.T) {
	camera := NewCamera3D()
	camera.Position = mat.NewVecDense(3, []float64{0, 0, 0})
	camera.Direction = mat.NewVecDense(3, []float64{1, 0, 0})
	camera.Up = mat.NewVecDense(3, []float64{0, 0, 1})
	camera.Width = 100
	camera.Height = 50
	camera.FieldOfViews = []float64{60, 67.38013505195957}

	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if camera.dir == nil || camera.up == nil || camera.right == nil {
		t.Fatal("expected cached camera basis vectors")
	}
	if math.Abs(camera.halfHeight-math.Tan(60*math.Pi/180/2.0)) > 1e-12 {
		t.Fatalf("unexpected cached half height: %v", camera.halfHeight)
	}
	if math.Abs(camera.halfWidth-math.Tan(67.38013505195957*math.Pi/180/2.0)) > 1e-12 {
		t.Fatalf("unexpected cached half width: %v", camera.halfWidth)
	}

	cachedRight := camera.right
	ray := camera.GenerateRay(nil, 5, 5)
	if ray == nil {
		t.Fatal("expected ray to be generated")
	}
	if camera.right != cachedRight {
		t.Fatal("expected camera basis cache to be reused")
	}
}

func TestCamera3DPrepareRefreshesWhenDimensionsChange(t *testing.T) {
	camera := NewCamera3D()
	camera.Position = mat.NewVecDense(3, []float64{0, 0, 0})
	camera.Direction = mat.NewVecDense(3, []float64{1, 0, 0})
	camera.Up = mat.NewVecDense(3, []float64{0, 0, 1})
	camera.Width = 100
	camera.Height = 50
	camera.FieldOfViews = []float64{60, 67.38013505195957}

	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	cachedInvWidth2 := camera.invWidth2

	camera.Width = 200
	if err := camera.Prepare(); err != nil {
		t.Fatalf("Prepare returned error after width change: %v", err)
	}
	if camera.invWidth2 == cachedInvWidth2 {
		t.Fatal("expected cached width scale to be refreshed")
	}
}
