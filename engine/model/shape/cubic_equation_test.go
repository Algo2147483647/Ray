package shape

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"gonum.org/v1/gonum/mat"
)

func TestCubicEquationIntersectsSimpleCubic(t *testing.T) {
	cubic := NewCubicEquation(cubicCoefficients(map[[3]int]float64{
		[3]int{1, 1, 1}: 1,
		[3]int{0, 0, 0}: -1,
	}))

	interaction, ok := cubic.IntersectAffine(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		NewIntersectOptions(0, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected ray to hit x^3 - 1 = 0")
	}
	if math.Abs(interaction.Distance-1) > 1e-9 {
		t.Fatalf("expected distance 1, got %f", interaction.Distance)
	}
}

func TestCubicEquationIntersectsDegenerateQuadratic(t *testing.T) {
	cubic := NewCubicEquation(cubicCoefficients(map[[3]int]float64{
		[3]int{1, 1, 0}: 1,
		[3]int{0, 0, 0}: -1,
	}))

	interaction, ok := cubic.IntersectAffine(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		NewIntersectOptions(0, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected degenerate quadratic to hit")
	}
	if math.Abs(interaction.Distance-1) > 1e-9 {
		t.Fatalf("expected distance 1, got %f", interaction.Distance)
	}
}

func TestCubicEquationChoosesClosestOfThreeRealRoots(t *testing.T) {
	cubic := NewCubicEquation(cubicCoefficients(map[[3]int]float64{
		[3]int{1, 1, 1}: 1,
		[3]int{1, 1, 0}: -6,
		[3]int{1, 0, 0}: 11,
		[3]int{0, 0, 0}: -6,
	}))

	interaction, ok := cubic.IntersectAffine(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		NewIntersectOptions(0, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected ray to hit one of three roots")
	}
	if math.Abs(interaction.Distance-1) > 1e-8 {
		t.Fatalf("expected closest root at distance 1, got %f", interaction.Distance)
	}
}

func TestBoundedCubicEquationCanChooseLaterRootInsideBounds(t *testing.T) {
	cubic := NewCubicEquation(cubicCoefficients(map[[3]int]float64{
		[3]int{1, 1, 1}: 1,
		[3]int{1, 1, 0}: -6,
		[3]int{1, 0, 0}: 11,
		[3]int{0, 0, 0}: -6,
	}))
	bounded := NewBoundedShape(cubic, NewCuboid(
		mat.NewVecDense(3, []float64{1.5, -1, -1}),
		mat.NewVecDense(3, []float64{2.5, 1, 1}),
	))

	interaction, ok := bounded.IntersectAffine(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		NewIntersectOptions(0, math.MaxFloat64),
	)
	if !ok {
		t.Fatal("expected bounded cubic to hit the root inside bounds")
	}
	if math.Abs(interaction.Distance-2) > 1e-8 {
		t.Fatalf("expected bounded hit at distance 2, got %f", interaction.Distance)
	}
}

func TestCubicEquationNormal(t *testing.T) {
	cubic := NewCubicEquation(cubicCoefficients(map[[3]int]float64{
		[3]int{1, 1, 1}: 1,
		[3]int{2, 2, 2}: 1,
		[3]int{3, 3, 3}: 1,
		[3]int{0, 0, 0}: -1,
	}))

	normal := cubic.GetNormalVector(
		mat.NewVecDense(3, []float64{1, 0, 0}),
		mat.NewVecDense(3, nil),
	)
	if math.Abs(normal.AtVec(0)-1) > 1e-9 || math.Abs(normal.AtVec(1)) > 1e-9 || math.Abs(normal.AtVec(2)) > 1e-9 {
		t.Fatalf("unexpected normal: %v", normal.RawVector().Data)
	}
}

func TestCubicEquationCalculateStorageCachesNonZeroTerms(t *testing.T) {
	a := maths.NewTensor[float64]([]int{4, 4, 4})
	a.Set(3, 0, 0, 0)
	a.Set(-2, 1, 1, 0)
	a.Set(5, 2, 3, 0)
	a.Set(7, 0, 1, 1)
	cubic := NewCubicEquation(a.Data)

	if len(cubic.Mem.Terms) != 3 {
		t.Fatalf("expected three cached non-zero terms, got %d", len(cubic.Mem.Terms))
	}
	got := map[[3]int]float64{}
	for _, term := range cubic.Mem.Terms {
		got[term.Powers] = term.Value
	}
	for powers, expected := range map[[3]int]float64{
		[3]int{0, 0, 0}: 3,
		[3]int{2, 0, 0}: 5,
		[3]int{0, 1, 1}: 5,
	} {
		if got[powers] != expected {
			t.Fatalf("expected cached term %v=%g, got %g", powers, expected, got[powers])
		}
	}
}

func cubicCoefficients(values map[[3]int]float64) []float64 {
	coeffs := make([]float64, 64)
	for index, value := range values {
		coeffs[(index[0]*4+index[1])*4+index[2]] = value
	}
	return coeffs
}
