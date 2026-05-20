package shape

import (
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type ImplicitEquation struct {
	BaseShape
	Function func(*mat.VecDense) float64 // parametric function
	Range    [2]*mat.VecDense
}

func NewImplicitEquation(Function func(*mat.VecDense) float64, Range [2]*mat.VecDense) *ImplicitEquation { // Index order: 1, x, y, z
	return &ImplicitEquation{
		Function: Function,
		Range:    Range,
	}
}

func (f *ImplicitEquation) Name() string {
	return "Implicit Equation"
}

func (f *ImplicitEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := f.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (f *ImplicitEquation) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	return SurfaceInteraction{}, false
}

func (f *ImplicitEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return math_lib.Normalize(res)
}

func (f *ImplicitEquation) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return f.Range[0], f.Range[1]
}
