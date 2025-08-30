package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
)

type ParametricEquation struct {
	BaseShape
	Function func(u, v float64) *mat.VecDense // parametric function
	URange   [2]float64
	VRange   [2]float64
}

func NewParametricEquation(Function func(u, v float64) *mat.VecDense) *ParametricEquation { // 索引顺序: 1, x, y, z
	return &ParametricEquation{
		Function: Function,
	}
}

func (f *ParametricEquation) Name() string {
	return "Parametric Equation"
}

func (f *ParametricEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	return f.IntersectPure(raySt, rayDir, 0, 0, 10, 10)
}

func (f *ParametricEquation) IntersectPure(raySt, rayDir *mat.VecDense, u0, v0, tol float64, maxIter int) float64 {
	equations := func(x []float64) []float64 { // 定义方程组：ray(t) - surface(u, v) = 0		// 二维参数曲面
		t, u, v := x[0], x[1], x[2]
		pointOnRay := mat.NewVecDense(raySt.Len(), nil)
		pointOnRay.AddVec(raySt, math_lib.ScaleVec2(t, rayDir))
		pointOnSurface := f.Function(u, v)

		result := make([]float64, 3)
		for i := 0; i < 3; i++ {
			result[i] = pointOnRay.AtVec(i) - pointOnSurface.AtVec(i)
		}
		return result
	}

	x := []float64{0.0, u0, v0}                                             // 初始猜测
	solution, success := math_lib.NewtonRaphson(equations, x, tol, maxIter) // 使用牛顿迭代法求解
	if success {
		t, u, v := solution[0], solution[1], solution[2]
		if f.URange[0] <= u && u <= f.URange[1] && // 检查参数是否在有效范围内
			f.VRange[0] <= v && v <= f.VRange[1] {
			return t
		}
	}

	return math.MaxFloat64
}

func (f *ParametricEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return math_lib.Normalize(res)
}

func (f *ParametricEquation) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return f.Function(f.URange[0], f.VRange[0]), f.Function(f.URange[1], f.VRange[1])
}
