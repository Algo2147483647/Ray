package shape

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"gonum.org/v1/gonum/mat"
)

var (
	_ SurfaceSampler = (*Sphere)(nil)
	_ SurfaceSampler = (*Circle)(nil)
	_ SurfaceSampler = (*Cuboid)(nil)
	_ SurfaceSampler = (*Triangle)(nil)
	_ SurfaceSampler = (*BoundedShape)(nil)
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

func TestSurfaceSamplerReferenceFallbackUsesAreaDistribution(t *testing.T) {
	sampler := SurfaceSampler(NewSphere(mat.NewVecDense(3, nil), 2))
	u := maths.Sample2D{U: 0.37, V: 0.81}
	reference := mat.NewVecDense(3, []float64{4, 5, 6})

	areaSample, ok := sampler.SampleSurface(u)
	if !ok {
		t.Fatal("expected area sample")
	}
	referenceSample, ok := SampleSurfaceFrom(sampler, reference, u)
	if !ok {
		t.Fatal("expected reference fallback sample")
	}
	if !mat.EqualApprox(areaSample.Point, referenceSample.Point, 0) {
		t.Fatalf("reference fallback point differs: area=%v reference=%v", areaSample.Point, referenceSample.Point)
	}
	if got, want := SurfacePDF(sampler, areaSample.Point), areaSample.PDFArea; got != want {
		t.Fatalf("surface PDF = %g, want %g", got, want)
	}
	if got, want := SurfacePDFFrom(sampler, reference, referenceSample.Point), referenceSample.PDFArea; got != want {
		t.Fatalf("reference fallback PDF = %g, want %g", got, want)
	}
}

func TestTriangleReferenceSamplingHasUniformSolidAngleDensity(t *testing.T) {
	triangle := NewTriangle(
		mat.NewVecDense(3, []float64{-1, -1, 2}),
		mat.NewVecDense(3, []float64{1, -1, 2}),
		mat.NewVecDense(3, []float64{0, 1, 2}),
	)
	reference := mat.NewVecDense(3, []float64{0, 0, 0})
	directions := [3][3]float64{}
	for index, point := range []*mat.VecDense{triangle.P1, triangle.P2, triangle.P3} {
		directions[index] = vecDenseXYZ(point)
		if !normalize3(&directions[index]) {
			t.Fatal("failed to normalize triangle direction")
		}
	}
	determinant := math.Abs(dot3(directions[0], cross3(directions[1], directions[2])))
	solidAngle := 2 * math.Atan2(
		determinant,
		1+dot3(directions[0], directions[1])+dot3(directions[1], directions[2])+dot3(directions[2], directions[0]),
	)

	for index := range 100 {
		u := maths.Sample2D{
			U: (float64(index) + 0.37) / 100,
			V: math.Mod(float64(index)*0.6180339887498949+0.23, 1),
		}
		sample, ok := SampleSurfaceFrom(triangle, reference, u)
		if !ok {
			t.Fatalf("reference sample %d failed", index)
		}
		if sample.UV[0] < 0 || sample.UV[1] < 0 || sample.UV[0]+sample.UV[1] > 1+1e-12 {
			t.Fatalf("sample %d lies outside triangle: uv=%v", index, sample.UV)
		}
		distance2 := mat.Dot(sample.Point, sample.Point)
		toSample := mat.VecDenseCopyOf(sample.Point)
		toSample.ScaleVec(1/math.Sqrt(distance2), toSample)
		cosine := math.Abs(mat.Dot(sample.Normal, toSample))
		want := cosine / (solidAngle * distance2)
		if relative := math.Abs(sample.PDFArea-want) / want; relative > 1e-10 {
			t.Fatalf("sample %d area PDF = %g, want %g (relative error %g)", index, sample.PDFArea, want, relative)
		}
		if evaluated := SurfacePDFFrom(triangle, reference, sample.Point); math.Abs(evaluated-sample.PDFArea)/want > 1e-10 {
			t.Fatalf("sample %d evaluated PDF = %g, sampled PDF %g", index, evaluated, sample.PDFArea)
		}
	}
}
