package object

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
)

func TestBuildUsesBinnedSAHToIsolateFarCluster(t *testing.T) {
	tree := &ObjectTree{}
	for i := 0; i < 10; i++ {
		x := float64(i) * 0.08
		tree.AddObject(&Object{Shape: testBox(x, 0, 0, x+0.02, 0.02, 0.02)})
	}
	tree.AddObject(&Object{Shape: testBox(20, 0, 0, 20.02, 0.02, 0.02)})

	tree.Build()

	if tree.Root == nil || tree.Root.Children[0] == nil || tree.Root.Children[1] == nil {
		t.Fatal("expected a binary BVH root")
	}
	leftLeaves := leafCount(tree.Root.Children[0])
	rightLeaves := leafCount(tree.Root.Children[1])
	if leftLeaves != 1 && rightLeaves != 1 {
		t.Fatalf("expected binned SAH split to isolate far cluster, got child leaf counts %d and %d", leftLeaves, rightLeaves)
	}
}

func TestRefitRefreshesDynamicBoundsWithoutChangingTopology(t *testing.T) {
	moving := testBox(0, 0, 0, 1, 1, 1)
	static := testBox(3, 0, 0, 4, 1, 1)
	tree := &ObjectTree{}
	tree.AddObject(&Object{Shape: moving})
	tree.AddObject(&Object{Shape: static})
	tree.Build()

	rootBefore := tree.Root
	moving.Pmin = mat.NewVecDense(3, []float64{10, 0, 0})
	moving.Pmax = mat.NewVecDense(3, []float64{11, 1, 1})
	tree.Refit()

	if tree.Root != rootBefore {
		t.Fatal("expected refit to preserve existing BVH topology")
	}
	if got := tree.Root.BoundBox.Pmax.AtVec(0); got != 11 {
		t.Fatalf("expected refit root max x to update to 11, got %f", got)
	}

	distance, obj := tree.GetIntersection(
		mat.NewVecDense(3, []float64{9, 0.5, 0.5}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		tree.Root,
	)
	if math.Abs(distance-1) > 1e-9 || obj == nil || obj.Shape != moving {
		t.Fatalf("expected ray to hit moved cuboid at distance 1, got distance=%f obj=%v", distance, obj)
	}
}

func TestUpdateAutoRebuildsWhenObjectCountChanges(t *testing.T) {
	tree := &ObjectTree{}
	tree.AddObject(&Object{Shape: testBox(0, 0, 0, 1, 1, 1)})
	tree.Build()

	tree.AddObject(&Object{Shape: testBox(3, 0, 0, 4, 1, 1)})
	tree.Update(BVHUpdateAuto)

	if got := leafCount(tree.Root); got != 2 {
		t.Fatalf("expected auto update to rebuild for changed object count, got %d leaves", got)
	}
}

func TestSurfaceHitCarriesInteractionData(t *testing.T) {
	tree := &ObjectTree{}
	first := tree.AddObject(&Object{Shape: testBox(3, 0, 0, 4, 1, 1)})
	second := tree.AddObject(&Object{Shape: testBox(0, 0, 0, 1, 1, 1)})
	tree.Build()

	hit, ok := tree.GetSurfaceHit(
		mat.NewVecDense(3, []float64{-1, 0.5, 0.5}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
	)
	if !ok {
		t.Fatal("expected surface hit")
	}
	if hit.Object != second || hit.Object == first {
		t.Fatalf("unexpected hit object: %+v", hit.Object)
	}
	if hit.PrimitiveID != 1 {
		t.Fatalf("expected primitive id 1, got %d", hit.PrimitiveID)
	}
	if hit.Point == nil || hit.GeometricNormal == nil || hit.ShadingNormal == nil {
		t.Fatal("expected complete surface interaction vectors")
	}
}

func TestSurfaceHitDoesNotMutateShapeOwnedNormal(t *testing.T) {
	triangle := shape.NewTriangle(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
	)
	tree := &ObjectTree{}
	tree.AddObject(&Object{Shape: triangle})
	tree.Build()

	original := mat.VecDenseCopyOf(triangle.Mem.Normal)
	hit, ok := tree.GetSurfaceHit(
		mat.NewVecDense(3, []float64{0.25, 0.25, 1}),
		mat.NewVecDense(3, []float64{0, 0, -1}),
	)
	if !ok {
		t.Fatal("expected triangle hit")
	}

	hit.GeometricNormal.SetVec(2, 42)
	if !mat.EqualApprox(triangle.Mem.Normal, original, 1e-12) {
		t.Fatalf("surface hit normal mutation leaked into triangle normal: got %v want %v", triangle.Mem.Normal.RawVector().Data, original.RawVector().Data)
	}
}

func TestSurfaceHitRangeHonorsTMax(t *testing.T) {
	tree := &ObjectTree{}
	sphere := shape.NewSphere(mat.NewVecDense(3, []float64{2, 0, 0}), 0.25)
	tree.AddObject(&Object{Shape: sphere})
	tree.Build()

	_, ok := tree.GetSurfaceHitRange(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		1e-6,
		1,
	)
	if ok {
		t.Fatal("did not expect a hit beyond tMax")
	}

	hit, ok := tree.GetSurfaceHitRange(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		1e-6,
		math.MaxFloat64,
	)
	if !ok {
		t.Fatal("expected hit when tMax includes the sphere")
	}
	if math.Abs(hit.Distance-1.75) > 1e-9 {
		t.Fatalf("unexpected hit distance: got %f want 1.75", hit.Distance)
	}
}

func TestSphericalSurfaceHitUsesGreatCirclePoint(t *testing.T) {
	const (
		centerArc = math.Pi / 3
		radius    = 0.1
	)
	center := mat.NewVecDense(4, []float64{math.Cos(centerArc), math.Sin(centerArc), 0, 0})
	tree := &ObjectTree{}
	tree.AddObject(&Object{Shape: shape.NewSphere(center, radius)})
	tree.Build()

	start := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	dir := mat.NewVecDense(4, []float64{0, 1, 0, 0})
	hit, ok := tree.GetSphericalSurfaceHit(start, dir, 1e-6, math.Pi)
	if !ok {
		t.Fatal("expected spherical geodesic hit")
	}

	wantArc := centerArc - 2*math.Asin(radius/2)
	if math.Abs(hit.ArcLength-wantArc) > 5e-4 {
		t.Fatalf("unexpected spherical arc: got %.9f want %.9f", hit.ArcLength, wantArc)
	}
	if math.Abs(mat.Norm(hit.Point, 2)-1) > 1e-9 {
		t.Fatalf("expected hit point to remain on S3, got norm %.12f", mat.Norm(hit.Point, 2))
	}
	if math.Abs(hit.Point.AtVec(0)-math.Cos(hit.ArcLength)) > 1e-6 ||
		math.Abs(hit.Point.AtVec(1)-math.Sin(hit.ArcLength)) > 1e-6 {
		t.Fatalf("hit point is not on the traced great circle: %v", hit.Point.RawVector().Data)
	}
}

func TestSphericalSurfaceHitStopsAtAntipode(t *testing.T) {
	tree := &ObjectTree{}
	tree.AddObject(&Object{Shape: shape.NewSphere(mat.NewVecDense(4, []float64{0, -1, 0, 0}), 0.05)})
	tree.Build()

	_, ok := tree.GetSphericalSurfaceHit(
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		1e-6,
		math.Pi,
	)
	if ok {
		t.Fatal("did not expect the first S3 half-circle to see beyond the antipode")
	}
}

func testBox(x0, y0, z0, x1, y1, z1 float64) *shape.Cuboid {
	return shape.NewCuboid(
		mat.NewVecDense(3, []float64{x0, y0, z0}),
		mat.NewVecDense(3, []float64{x1, y1, z1}),
	)
}
