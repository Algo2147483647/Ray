package shape

import (
	"github.com/Algo2147483647/golang_toolkit/math/basic_algebra"
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type ParametricEquation struct {
	BaseShape
	Function func(u, v float64) *mat.VecDense // parametric function
	URange   [2]float64
	VRange   [2]float64
}

func NewParametricEquation(Function func(u, v float64) *mat.VecDense) *ParametricEquation { // Index order: 1, x, y, z
	return &ParametricEquation{
		Function: Function,
	}
}

func (f *ParametricEquation) Name() string {
	return "Parametric Equation"
}

func (f *ParametricEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := f.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (f *ParametricEquation) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	distance := f.IntersectPure(raySt, rayDir, 0, 0, 10, 10)
	if !distanceInRange(distance, tMin, tMax) {
		return SurfaceInteraction{}, false
	}

	point := pointAt(raySt, rayDir, distance)
	normal := f.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteraction(raySt, rayDir, distance, normal), true
}

func (f *ParametricEquation) IntersectPure(raySt, rayDir *mat.VecDense, u0, v0, tol float64, maxIter int) float64 {
	equations := func(x []float64) []float64 { // Define the equation system: ray(t) - surface(u, v) = 0. // 2D parametric surface.
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

	x := []float64{0.0, u0, v0}                                                  // Initial guess
	solution, success := basic_algebra.NewtonRaphson(equations, x, tol, maxIter) // Solve with the Newton-Raphson method.
	if success {
		t, u, v := solution[0], solution[1], solution[2]
		if f.URange[0] <= u && u <= f.URange[1] && // Check whether parameters are in valid range.
			f.VRange[0] <= v && v <= f.VRange[1] {
			if t > utils.EPS {
				return t
			}
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
