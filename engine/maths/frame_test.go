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

	assertNear(t, negated.X, -local.X)
	assertNear(t, negated.Y, -local.Y)
	assertNear(t, negated.Z, -local.Z)
}

func TestFrameRejectsNon3DNormal(t *testing.T) {
	_, ok := NewFrameFromNormal(mat.NewVecDense(2, []float64{0, 1}))
	if ok {
		t.Fatal("expected non-3D normal to be rejected")
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
