package geometry

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestEuclideanDefaultAndArcLength(t *testing.T) {
	g := Get(nil)
	if g.Name() != "euclidean" {
		t.Fatalf("default geometry = %q, want euclidean", g.Name())
	}
	if g.Dimension() != 3 {
		t.Fatalf("default dimension = %d, want 3", g.Dimension())
	}

	p := mat.NewVecDense(3, []float64{0, 0, 0})
	d := mat.NewVecDense(3, []float64{2, 0, 0})
	if got := g.ArcLengthFromEmbedT(p, d, 1.25); got != 2.5 {
		t.Fatalf("Euclidean arc length = %v, want 2.5", got)
	}
}

func TestEuclideanExpIsAffine(t *testing.T) {
	g := Euclidean()
	p := mat.NewVecDense(3, []float64{1, 2, 3})
	v := mat.NewVecDense(3, []float64{0, 1, -1})
	out := mat.NewVecDense(3, nil)

	got := g.Exp(p, v, 4, out)
	assertVecApprox(t, got, []float64{1, 6, -1}, 0)
}

func TestKleinDistanceFromOriginIsAtanhRadius(t *testing.T) {
	g := Klein()
	origin := mat.NewVecDense(3, []float64{0, 0, 0})
	dir := mat.NewVecDense(3, []float64{1, 0, 0})

	for _, r := range []float64{0.1, 0.3, 0.7, 0.9} {
		got := g.ArcLengthFromEmbedT(origin, dir, r)
		want := math.Atanh(r)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("Klein arc to radius %v = %.15f, want %.15f", r, got, want)
		}
	}
}

func TestKleinInnerProductUsesMetricTensor(t *testing.T) {
	g := Klein()
	p := mat.NewVecDense(3, []float64{0.5, 0, 0})
	radial := mat.NewVecDense(3, []float64{1, 0, 0})
	transverse := mat.NewVecDense(3, []float64{0, 1, 0})

	if got, want := g.InnerProduct(p, radial, radial), 1/math.Pow(1-0.25, 2); math.Abs(got-want) > 1e-12 {
		t.Fatalf("radial Klein inner product = %.15f, want %.15f", got, want)
	}
	if got, want := g.InnerProduct(p, transverse, transverse), 1/(1-0.25); math.Abs(got-want) > 1e-12 {
		t.Fatalf("transverse Klein inner product = %.15f, want %.15f", got, want)
	}
}

func TestKleinExpRoundTrip(t *testing.T) {
	g := Klein()
	p := mat.NewVecDense(3, []float64{0.1, 0.2, -0.1})
	v := mat.NewVecDense(3, []float64{1, 0, 0})
	q := g.Exp(p, v, 0.5, mat.NewVecDense(3, nil))

	chord := mat.NewVecDense(3, nil)
	chord.SubVec(q, p)
	got := g.ArcLengthFromEmbedT(p, chord, 1)
	if math.Abs(got-0.5) > 1e-6 {
		t.Fatalf("Klein Exp round-trip arc = %.15f, want 0.5", got)
	}
}

func TestKleinEmbeddedRayStopsAtUnitBall(t *testing.T) {
	_, _, tMax := Klein().EmbeddedRay(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
	)
	if math.Abs(tMax-1) > 1e-12 {
		t.Fatalf("Klein tMax = %.15f, want 1", tMax)
	}
}

func TestSphericalProjectTangentSubtractsRadial(t *testing.T) {
	g := Spherical()
	p := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	v := mat.NewVecDense(4, []float64{0.3, 1, 0, 0})
	out := g.ProjectTangent(p, v, mat.NewVecDense(4, nil))

	assertVecApprox(t, out, []float64{0, 1, 0, 0}, 1e-12)
}

func TestSphericalDistanceToAntipodeIsPi(t *testing.T) {
	g := Spherical()
	p := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	dir := mat.NewVecDense(4, []float64{-1, 0, 0, 0})

	if got := g.ArcLengthFromEmbedT(p, dir, 2); math.Abs(got-math.Pi) > 1e-9 {
		t.Fatalf("Spherical arc to antipode = %.15f, want pi", got)
	}
}

func TestSphericalExpAndWrapFollowGreatCircle(t *testing.T) {
	g := Spherical()
	p := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	dir := mat.NewVecDense(4, []float64{0, 1, 0, 0})

	q := g.Exp(p, dir, math.Pi/2, mat.NewVecDense(4, nil))
	assertVecApprox(t, q, []float64{0, 1, 0, 0}, 1e-9)

	newP, newD, ok := g.WrapBeyond(p, dir, math.Pi)
	if !ok {
		t.Fatal("expected spherical wrap to succeed")
	}
	assertVecApprox(t, newP, []float64{-1, 0, 0, 0}, 1e-9)
	assertVecApprox(t, newD, []float64{0, -1, 0, 0}, 1e-9)
	if dot := mat.Dot(newP, newD); math.Abs(dot) > 1e-9 {
		t.Fatalf("wrapped direction is not tangent: dot = %g", dot)
	}
}

func TestWrapBeyondOnlySpherical(t *testing.T) {
	p := mat.NewVecDense(3, []float64{0, 0, 0})
	d := mat.NewVecDense(3, []float64{1, 0, 0})
	if _, _, ok := Euclidean().WrapBeyond(p, d, 1); ok {
		t.Fatal("Euclidean wrap should fail")
	}
	if _, _, ok := Klein().WrapBeyond(p, d, 1); ok {
		t.Fatal("Klein wrap should fail")
	}
}

func assertVecApprox(t *testing.T, got *mat.VecDense, want []float64, tolerance float64) {
	t.Helper()
	if got.Len() != len(want) {
		t.Fatalf("length mismatch: got %d, want %d", got.Len(), len(want))
	}
	for i, w := range want {
		if math.Abs(got.AtVec(i)-w) > tolerance {
			t.Fatalf("component %d = %.15f, want %.15f", i, got.AtVec(i), w)
		}
	}
}
