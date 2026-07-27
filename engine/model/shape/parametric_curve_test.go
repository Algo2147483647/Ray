package shape

import (
	"math"
	"sync/atomic"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestParametricCurveStraightTubeHit(t *testing.T) {
	curve := NewParametricCurve(
		func(t float64) *mat.VecDense {
			return mat.NewVecDense(3, []float64{0, t, 0})
		},
		func(t float64) float64 {
			return 0.25
		},
		[2]float64{-1, 1},
	)
	curve.Derivative = func(t float64, res *mat.VecDense) *mat.VecDense {
		res.SetVec(1, 1)
		return res
	}
	curve.Samples = 64

	interaction, ok := curve.IntersectAffine(
		mat.NewVecDense(3, []float64{-1, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		NewIntersectOptions(1e-6, math.MaxFloat64),
	)

	if !ok {
		t.Fatal("expected tube hit")
	}
	if math.Abs(interaction.Distance-0.75) > 1e-5 {
		t.Fatalf("expected distance 0.75, got %.12f", interaction.Distance)
	}
	if math.Abs(interaction.Point.AtVec(0)+0.25) > 1e-5 {
		t.Fatalf("unexpected hit point: %v", interaction.Point.RawVector().Data)
	}
	if interaction.GeometricNormal == nil || math.Abs(interaction.GeometricNormal.AtVec(0)+1) > 1e-5 {
		t.Fatalf("expected normal toward negative x, got %v", interaction.GeometricNormal)
	}
	if interaction.DPDU == nil || math.Abs(interaction.DPDU.AtVec(1)-1) > 1e-9 {
		t.Fatalf("expected curve tangent, got %v", interaction.DPDU)
	}
}

func TestParametricCurveEndpointCapHit(t *testing.T) {
	curve := NewParametricCurve(
		func(t float64) *mat.VecDense {
			return mat.NewVecDense(3, []float64{0, t, 0})
		},
		func(t float64) float64 {
			return 0.25
		},
		[2]float64{0, 1},
	)
	curve.Samples = 64

	interaction, ok := curve.IntersectAffine(
		mat.NewVecDense(3, []float64{-1, -0.2, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		NewIntersectOptions(1e-6, math.MaxFloat64),
	)

	if !ok {
		t.Fatal("expected endpoint cap hit")
	}
	expected := 1 - math.Sqrt(0.25*0.25-0.2*0.2)
	if math.Abs(interaction.Distance-expected) > 1e-4 {
		t.Fatalf("expected cap distance %.12f, got %.12f", expected, interaction.Distance)
	}
}

func TestParametricCurveMiss(t *testing.T) {
	curve := NewParametricCurve(
		func(t float64) *mat.VecDense {
			return mat.NewVecDense(3, []float64{0, t, 0})
		},
		func(t float64) float64 {
			return 0.1
		},
		[2]float64{-1, 1},
	)

	if _, ok := curve.IntersectAffine(
		mat.NewVecDense(3, []float64{-1, 0, 0.5}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		NewIntersectOptions(1e-6, math.MaxFloat64),
	); ok {
		t.Fatal("expected ray to miss thin curve")
	}
}

func TestParametricCurveMissUsesSegmentBVHCoarseCulling(t *testing.T) {
	var evals int64
	curve := NewParametricCurve(
		func(t float64) *mat.VecDense {
			atomic.AddInt64(&evals, 1)
			return mat.NewVecDense(3, []float64{
				0.35 * math.Cos(t),
				0.35 * math.Sin(t),
				0.2 * math.Sin(3*t),
			})
		},
		func(t float64) float64 {
			return 0.025
		},
		[2]float64{0, 2 * math.Pi},
	)
	curve.Samples = 256

	if err := curve.BuildAcceleration(); err != nil {
		t.Fatalf("build acceleration: %v", err)
	}
	buildEvals := atomic.LoadInt64(&evals)

	for i := 0; i < 200; i++ {
		y := -0.9 + float64(i)*1.8/199
		_, _ = curve.IntersectAffine(
			mat.NewVecDense(3, []float64{-1, y, 0.55}),
			mat.NewVecDense(3, []float64{1, 0, 0}),
			NewIntersectOptions(1e-6, 3),
		)
	}

	queryEvals := atomic.LoadInt64(&evals) - buildEvals
	if queryEvals > 64 {
		t.Fatalf("expected BVH coarse culling to avoid broad curve evaluation, got %d query evals", queryEvals)
	}
}
