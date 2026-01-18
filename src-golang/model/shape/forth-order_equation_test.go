package shape

import (
	"fmt"
	"testing"
)

func TestFourOrderEquation(t *testing.T) {
	t.Run("(x - 1)^4 + y^4 + z^4 = 1", func(t *testing.T) {
		a := math_lib.NewTensor[float64]([]int{4, 4, 4, 4})

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
		a := math_lib.NewTensor[float64]([]int{4, 4, 4, 4})

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
