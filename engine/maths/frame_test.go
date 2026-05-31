package maths

import (
	"math"
	"testing"

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
