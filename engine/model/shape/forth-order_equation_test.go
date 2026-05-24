package shape

import (
	"fmt"
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"gonum.org/v1/gonum/mat"
)

func TestFourOrderEquation(t *testing.T) {
	t.Run("(x - 1)^4 + y^4 + z^4 = 1", func(t *testing.T) {
		a := maths.NewTensor[float64]([]int{4, 4, 4, 4})

		a.Set(1, 0, 0, 0, 0)
		a.Set(-1, 1, 1, 1, 1)
		a.Set(-1, 2, 2, 2, 2)
		a.Set(-1, 3, 3, 3, 3)

		a.Set(a.Get(1, 1, 1, 1)-1, 1, 1, 1, 1)
		a.Set(a.Get(1, 1, 1, 0)+4, 1, 1, 1, 0)
		a.Set(a.Get(1, 1, 0, 0)-6, 1, 1, 0, 0)
		a.Set(a.Get(1, 0, 0, 0)+4, 1, 0, 0, 0)
		a.Set(a.Get(0, 0, 0, 0)-1, 0, 0, 0, 0)

		for i, v := range a.Data {
			if i%64 == 0 {
				println()
				println()
			} else if i%16 == 0 {
				println()
			} else if i%4 == 0 {
				print("   ")
			}

			print(int(v))
			print(", ")
		}
		println()
	})

	t.Run("Tanglecube: sum_{i=1}^3 (x_i^4 - 5 x_i^2 ) + 11.8 = 0, (x - 1.5), ", func(t *testing.T) {
		a := maths.NewTensor[float64]([]int{4, 4, 4, 4})

		a.Set(11.8, 0, 0, 0, 0)
		a.Set(1, 1, 1, 1, 1)
		a.Set(1, 2, 2, 2, 2)
		a.Set(1, 3, 3, 3, 3)
		a.Set(-5, 1, 1, 0, 0)
		a.Set(-5, 2, 2, 0, 0)
		a.Set(-5, 3, 3, 0, 0)

		for i, v := range a.Data {
			if i%64 == 0 {
				println()
				println()
			} else if i%16 == 0 {
				println()
			} else if i%4 == 0 {
				print("   ")
			}

			fmt.Printf("%.2f,", v)
		}
		println()
	})
}

func TestFourOrderEquationIntersectsQuarticWithShiftedRoots(t *testing.T) {
	quartic := NewFourOrderEquation(fourOrderCoefficients(map[[4]int]float64{
		[4]int{1, 1, 1, 1}: 1,
		[4]int{1, 1, 1, 0}: -12,
		[4]int{1, 1, 0, 0}: 54,
		[4]int{1, 0, 0, 0}: -108,
		[4]int{0, 0, 0, 0}: 80,
	}))

	interaction, ok := quartic.IntersectRange(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		0,
		math.MaxFloat64,
	)
	if !ok {
		t.Fatal("expected ray to hit (x - 3)^4 - 1 = 0")
	}
	if math.Abs(interaction.Distance-2) > 1e-8 {
		t.Fatalf("expected closest hit at distance 2, got %f", interaction.Distance)
	}
}

func fourOrderCoefficients(values map[[4]int]float64) []float64 {
	coeffs := make([]float64, 256)
	for index, value := range values {
		coeffs[((index[0]*4+index[1])*4+index[2])*4+index[3]] = value
	}
	return coeffs
}
