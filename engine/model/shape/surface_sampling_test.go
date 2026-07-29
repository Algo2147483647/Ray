package shape

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"gonum.org/v1/gonum/mat"
)

func TestFiniteShapeSurfaceAreas(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(3, nil), 2)
	if got, want := sphere.SurfaceArea(), 16*math.Pi; math.Abs(got-want) > 1e-12 {
		t.Fatalf("sphere area = %g, want %g", got, want)
	}

	triangle := NewTriangle(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{2, 0, 0}),
		mat.NewVecDense(3, []float64{0, 3, 0}),
	)
	if got, want := triangle.SurfaceArea(), 3.0; math.Abs(got-want) > 1e-12 {
		t.Fatalf("triangle area = %g, want %g", got, want)
	}

	box := NewCuboid(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 2, 3}),
	)
	if got, want := box.SurfaceArea(), 22.0; math.Abs(got-want) > 1e-12 {
		t.Fatalf("cuboid area = %g, want %g", got, want)
	}
}

func TestSurfaceSamplesHaveAreaPDFAndLieOnShape(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(3, []float64{1, 2, 3}), 2)
	sample, ok := sphere.SampleSurface(maths.Sample2D{U: 0.37, V: 0.81})
	if !ok {
		t.Fatal("expected sphere to be sampleable")
	}
	offset := mat.NewVecDense(3, nil)
	offset.SubVec(sample.Point, sphere.center)
	if math.Abs(mat.Norm(offset, 2)-sphere.R) > 1e-12 {
		t.Fatalf("sample is not on sphere: radius %g", mat.Norm(offset, 2))
	}
	if math.Abs(sample.PDFArea-1/sphere.SurfaceArea()) > 1e-12 {
		t.Fatalf("unexpected area PDF %g", sample.PDFArea)
	}
	if math.Abs(mat.Norm(sample.Normal, 2)-1) > 1e-12 {
		t.Fatalf("sample normal is not normalized: %g", mat.Norm(sample.Normal, 2))
	}
}

func TestUnsupportedDimensionalSphereIsNotAreaSampleable(t *testing.T) {
	sphere := NewSphere(mat.NewVecDense(4, nil), 1)
	if sphere.SurfaceArea() != 0 {
		t.Fatal("BDPT surface measure must reject non-3D sphere")
	}
	if _, ok := sphere.SampleSurface(maths.Sample2D{U: 0.5, V: 0.5}); ok {
		t.Fatal("expected non-3D sphere sampling to fail")
	}
}
