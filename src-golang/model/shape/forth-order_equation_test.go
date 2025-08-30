package shape

import (
	"src-golang/math_lib"
	"testing"
)

func TestFourOrderEquation(t *testing.T) {
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
}
