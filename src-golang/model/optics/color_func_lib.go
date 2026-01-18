package optics

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/utils"
)

type ColorFunc func(ray *Ray, norm *mat.VecDense) *mat.VecDense

var ColorFuncMap map[string]ColorFunc = map[string]ColorFunc{
	"color_func_1": ColorFunc1,
}

func ColorFunc1(ray *Ray, norm *mat.VecDense) *mat.VecDense {
	if math.Abs(math.Abs(norm.AtVec(0))-1) < utils.EPS {
		return mat.NewVecDense(3, []float64{1, 0.5, 0.5})
	} else if math.Abs(math.Abs(norm.AtVec(1))-1) < utils.EPS {
		return mat.NewVecDense(3, []float64{0.5, 1, 0.5})
	} else if math.Abs(math.Abs(norm.AtVec(2))-1) < utils.EPS {
		return mat.NewVecDense(3, []float64{0.5, 0.5, 1})
	} else if math.Abs(math.Abs(norm.AtVec(3))-1) < utils.EPS {
		return mat.NewVecDense(3, []float64{0.5, 1, 1})
	}
	return mat.NewVecDense(3, []float64{1, 1, 1})
}
