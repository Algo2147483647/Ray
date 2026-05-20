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
	distance := f.IntersectPure(raySt, rayDir, 0, 0, 10, 10)
	if !distanceInRange(distance, tMin, tMax) {
		return SurfaceInteraction{}, false
	}

	point := pointAt(raySt, rayDir, distance)
	normal := f.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteraction(raySt, rayDir, distance, normal), true
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
