package shape

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestCuboidIntersectRangeParallelAxisDoesNotDivideByZero(t *testing.T) {
	cuboid := NewCuboid(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 1, 1}),
	)

	interaction, ok := cuboid.IntersectRange(
		mat.NewVecDense(3, []float64{0.5, 0.5, -1}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1e-6,
		math.MaxFloat64,
	)
	if !ok {
		t.Fatal("expected ray parallel to x/y axes to hit cuboid")
	}
	if math.IsNaN(interaction.Distance) || math.IsInf(interaction.Distance, 0) {
		t.Fatalf("expected finite distance, got %f", interaction.Distance)
	}
	if math.Abs(interaction.Distance-1) > 1e-9 {
		t.Fatalf("unexpected distance: got %f want 1", interaction.Distance)
	}
}

func TestCuboidNormalUsesPminAndPmaxDirection(t *testing.T) {
	cuboid := NewCuboid(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 1, 1}),
	)

	minNormal := cuboid.GetNormalVector(mat.NewVecDense(3, []float64{0, 0.5, 0.5}), mat.NewVecDense(3, nil))
	maxNormal := cuboid.GetNormalVector(mat.NewVecDense(3, []float64{1, 0.5, 0.5}), mat.NewVecDense(3, nil))

	if minNormal.AtVec(0) != -1 || minNormal.AtVec(1) != 0 || minNormal.AtVec(2) != 0 {
		t.Fatalf("unexpected pmin normal: %v", minNormal.RawVector().Data)
	}
	if maxNormal.AtVec(0) != 1 || maxNormal.AtVec(1) != 0 || maxNormal.AtVec(2) != 0 {
		t.Fatalf("unexpected pmax normal: %v", maxNormal.RawVector().Data)
	}
}

func TestHypercuboidIntersectRange4D(t *testing.T) {
	cuboid := NewCuboid(
		mat.NewVecDense(4, []float64{-1, -1, -1, -1}),
		mat.NewVecDense(4, []float64{1, 1, 1, 1}),
	)

	interaction, ok := cuboid.IntersectRange(
		mat.NewVecDense(4, []float64{-3, 0, 0, 0}),
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		1e-6,
		math.MaxFloat64,
	)

	if !ok {
		t.Fatal("expected 4D hypercuboid hit")
	}
	if interaction.Point.Len() != 4 || interaction.GeometricNormal.Len() != 4 {
		t.Fatalf("expected 4D interaction, got point=%d normal=%d", interaction.Point.Len(), interaction.GeometricNormal.Len())
	}
	if math.Abs(interaction.Distance-2) > 1e-9 {
		t.Fatalf("expected nearest distance 2, got %f", interaction.Distance)
	}
	if interaction.GeometricNormal.AtVec(0) != -1 {
		t.Fatalf("expected hit on negative x face, got normal %v", interaction.GeometricNormal.RawVector().Data)
	}
}
