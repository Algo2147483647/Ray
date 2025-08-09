package math_lib

import "gonum.org/v1/gonum/mat"

func normalize(v *mat.VecDense) {
	norm := mat.Norm(v, 2)
	if norm == 0 {
		return
	}
	v.ScaleVec(1/norm, v)
}
