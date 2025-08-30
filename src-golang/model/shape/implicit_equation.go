package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
)

type ImplicitEquation struct {
	BaseShape
	Function func(*mat.VecDense) float64 // parametric function
	Range    [2]*mat.VecDense
}

func NewImplicitEquation(Function func(*mat.VecDense) float64, Range [2]*mat.VecDense) *ImplicitEquation { // 索引顺序: 1, x, y, z
	return &ImplicitEquation{
		Function: Function,
		Range:    Range,
	}
}

func (f *ImplicitEquation) Name() string {
	return "Implicit Equation"
}

func (f *ImplicitEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	return f.IntersectPure(raySt, rayDir, 0, 0, 10, 10)
}

func (f *ImplicitEquation) IntersectPure(raySt, rayDir *mat.VecDense, u0, v0, tol float64, maxIter int) float64 {
	return math.MaxFloat64
}

func (f *ImplicitEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return math_lib.Normalize(res)
}

func (f *ImplicitEquation) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return f.Range[0], f.Range[1]
}
