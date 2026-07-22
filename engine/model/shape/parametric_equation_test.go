package shape

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestParametricPlanePatchHitReturnsInteractionData(t *testing.T) {
	plane := NewParametricEquation(
		func(u, v float64) *mat.VecDense {
			return mat.NewVecDense(3, []float64{u, v, 0})
		},
		[2]float64{-1, 1},
		[2]float64{-1, 1},
	)
	plane.Derivative = func(u, v float64, du, dv *mat.VecDense) (*mat.VecDense, *mat.VecDense) {
		du.SetVec(0, 1)
		dv.SetVec(1, 1)
		return du, dv
	}
	plane.SamplesU = 8
	plane.SamplesV = 8

	interaction, ok := plane.IntersectRange(
		mat.NewVecDense(3, []float64{0.25, -0.5, -2}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1e-6,
		math.MaxFloat64,
	)

	if !ok {
		t.Fatal("expected plane patch hit")
	}
	if math.Abs(interaction.Distance-2) > 1e-6 {
		t.Fatalf("expected distance 2, got %.12f", interaction.Distance)
	}
	if math.Abs(interaction.Point.AtVec(0)-0.25) > 1e-6 || math.Abs(interaction.Point.AtVec(1)+0.5) > 1e-6 {
		t.Fatalf("unexpected point: %v", interaction.Point.RawVector().Data)
	}
	if interaction.DPDU == nil || interaction.DPDV == nil {
		t.Fatal("expected parametric derivatives")
	}
	if math.Abs(mat.Norm(interaction.GeometricNormal, 2)-1) > 1e-9 {
		t.Fatalf("expected unit normal, got %v", interaction.GeometricNormal.RawVector().Data)
	}
	if math.Abs(interaction.UV[0]-0.625) > 1e-6 || math.Abs(interaction.UV[1]-0.25) > 1e-6 {
		t.Fatalf("unexpected UV: %v", interaction.UV)
	}
}

func TestParametricPlanePatchRejectsOutsideParameterRange(t *testing.T) {
	plane := NewParametricEquation(
		func(u, v float64) *mat.VecDense {
			return mat.NewVecDense(3, []float64{u, v, 0})
		},
		[2]float64{-1, 1},
		[2]float64{-1, 1},
	)
	plane.Derivative = func(u, v float64, du, dv *mat.VecDense) (*mat.VecDense, *mat.VecDense) {
		du.SetVec(0, 1)
		dv.SetVec(1, 1)
		return du, dv
	}

	if _, ok := plane.IntersectRange(
		mat.NewVecDense(3, []float64{2, 0, -2}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1e-6,
		math.MaxFloat64,
	); ok {
		t.Fatal("expected miss outside parameter range")
	}
}

func TestParametricTorusReturnsNearestHit(t *testing.T) {
	majorRadius := 2.0
	minorRadius := 0.5
	torus := NewParametricEquation(
		func(u, v float64) *mat.VecDense {
			ring := majorRadius + minorRadius*math.Cos(v)
			return mat.NewVecDense(3, []float64{
				ring * math.Cos(u),
				ring * math.Sin(u),
				minorRadius * math.Sin(v),
			})
		},
		[2]float64{0, 2 * math.Pi},
		[2]float64{0, 2 * math.Pi},
	)
	torus.Derivative = func(u, v float64, du, dv *mat.VecDense) (*mat.VecDense, *mat.VecDense) {
		ring := majorRadius + minorRadius*math.Cos(v)
		du.SetVec(0, -ring*math.Sin(u))
		du.SetVec(1, ring*math.Cos(u))
		du.SetVec(2, 0)
		dv.SetVec(0, -minorRadius*math.Sin(v)*math.Cos(u))
		dv.SetVec(1, -minorRadius*math.Sin(v)*math.Sin(u))
		dv.SetVec(2, minorRadius*math.Cos(v))
		return du, dv
	}
	torus.SamplesU = 48
	torus.SamplesV = 24

	interaction, ok := torus.IntersectRange(
		mat.NewVecDense(3, []float64{0, -3, 0}),
		mat.NewVecDense(3, []float64{0, 1, 0}),
		1e-6,
		math.MaxFloat64,
	)

	if !ok {
		t.Fatal("expected torus hit")
	}
	if math.Abs(interaction.Distance-0.5) > 1e-5 {
		t.Fatalf("expected nearest distance 0.5, got %.12f", interaction.Distance)
	}
}
