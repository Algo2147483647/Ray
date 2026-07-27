package maths

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"gonum.org/v1/gonum/mat"
)

func TestFrameFromNormalIsOrthonormal(t *testing.T) {
	frame, ok := NewFrameFromNormal(mat.NewVecDense(3, []float64{0, 0, 2}))
	if !ok {
		t.Fatal("expected a frame for a 3D normal")
	}

	assertNear(t, mat.Norm(frame.Tangent, 2), 1)
	assertNear(t, mat.Norm(frame.Bitangent, 2), 1)
	assertNear(t, mat.Norm(frame.Normal, 2), 1)
	assertNear(t, mat.Dot(frame.Tangent, frame.Bitangent), 0)
	assertNear(t, mat.Dot(frame.Tangent, frame.Normal), 0)
	assertNear(t, mat.Dot(frame.Bitangent, frame.Normal), 0)
}

func TestFrameWorldLocalRoundTrip(t *testing.T) {
	frame, ok := NewFrameFromNormal(mat.NewVecDense(3, []float64{0, 0, 1}))
	if !ok {
		t.Fatal("expected a frame for a 3D normal")
	}

	world := mat.NewVecDense(3, []float64{0.25, -0.5, 0.75})
	local := frame.WorldToLocal(world)
	roundTrip := frame.LocalToWorld(local)

	assertVecNear(t, roundTrip, world)
}

func TestFrameWorldToLocalNegated(t *testing.T) {
	frame, ok := NewFrameFromNormal(mat.NewVecDense(3, []float64{0, 0, 1}))
	if !ok {
		t.Fatal("expected a frame for a 3D normal")
	}

	world := mat.NewVecDense(3, []float64{0.25, -0.5, 0.75})
	local := frame.WorldToLocal(world)
	negated := frame.WorldToLocalNegated(world)

	assertNear(t, negated.Component(0), -local.Component(0))
	assertNear(t, negated.Component(1), -local.Component(1))
	assertNear(t, negated.Component(2), -local.Component(2))
}

func TestFrameSupports4DNormal(t *testing.T) {
	frame, ok := NewFrameFromNormal(mat.NewVecDense(4, []float64{0, 0, 0, 1}))
	if ok {
		if len(frame.Tangents) != 3 {
			t.Fatalf("expected three 4D tangent vectors, got %d", len(frame.Tangents))
		}
		for i, tangent := range frame.Tangents {
			assertNear(t, mat.Norm(tangent, 2), 1)
			assertNear(t, mat.Dot(tangent, frame.Normal), 0)
			for j := 0; j < i; j++ {
				assertNear(t, mat.Dot(tangent, frame.Tangents[j]), 0)
			}
		}
		return
	}
	t.Fatal("expected a frame for a 4D normal")
}

func TestKleinFrameIsMetricOrthonormalAndRoundTrips(t *testing.T) {
	g := geometry.Klein()
	p := mat.NewVecDense(3, []float64{0.6, 0.2, 0.1})
	gradient := mat.NewVecDense(3, []float64{1, 2, -0.5})
	normal := g.IntrinsicNormal(p, gradient, mat.NewVecDense(3, nil))
	frame, ok := NewFrameFromNormalInGeometry(g, p, normal)
	if !ok {
		t.Fatal("expected Klein metric frame")
	}

	basis := append(append([]*mat.VecDense{}, frame.Tangents...), frame.Normal)
	for i, left := range basis {
		for j, right := range basis {
			want := 0.0
			if i == j {
				want = 1
			}
			if got := g.InnerProduct(p, left, right); math.Abs(got-want) > 1e-10 {
				t.Fatalf("basis inner product (%d,%d)=%g want %g", i, j, got, want)
			}
		}
	}

	world := mat.NewVecDense(3, []float64{0.2, -0.4, 0.3})
	roundTrip := frame.LocalToWorld(frame.WorldToLocal(world))
	if !mat.EqualApprox(roundTrip, world, 1e-10) {
		t.Fatalf("Klein frame round trip got %v want %v", roundTrip.RawVector().Data, world.RawVector().Data)
	}
}

func TestKleinFrameSpecularReflectionPreservesMetricAngle(t *testing.T) {
	g := geometry.Klein()
	p := mat.NewVecDense(3, []float64{0.55, -0.2, 0.1})
	gradient := mat.NewVecDense(3, []float64{1, 0.4, -0.3})
	normal := g.IntrinsicNormal(p, gradient, mat.NewVecDense(3, nil))
	frame, ok := NewFrameFromNormalInGeometry(g, p, normal)
	if !ok {
		t.Fatal("expected Klein metric frame")
	}

	incoming := mat.NewVecDense(3, nil)
	incoming.AddScaledVec(incoming, 0.8, frame.Tangent)
	incoming.AddScaledVec(incoming, -0.6, frame.Normal)
	wo := frame.WorldToLocalNegated(incoming)
	reflectedLocal := NewDirection(-wo.Component(0), -wo.Component(1), wo.Component(2))
	reflected := frame.LocalToWorld(reflectedLocal)

	if got := g.InnerProduct(p, reflected, reflected); math.Abs(got-1) > 1e-10 {
		t.Fatalf("reflected Klein length squared=%g want 1", got)
	}
	inCos := math.Abs(g.InnerProduct(p, incoming, frame.Normal))
	outCos := math.Abs(g.InnerProduct(p, reflected, frame.Normal))
	if math.Abs(inCos-outCos) > 1e-10 {
		t.Fatalf("reflection angle mismatch: incoming=%g outgoing=%g", inCos, outCos)
	}
}

func assertVecNear(t *testing.T, got, want *mat.VecDense) {
	t.Helper()
	if got.Len() != want.Len() {
		t.Fatalf("length mismatch: got %d, want %d", got.Len(), want.Len())
	}
	for i := 0; i < got.Len(); i++ {
		assertNear(t, got.AtVec(i), want.AtVec(i))
	}
}

func assertNear(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %g, want %g", got, want)
	}
}
