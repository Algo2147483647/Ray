package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
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
	return newSurfaceInteractionAt(point, distance, normal), true
}

func (f *ParametricEquation) IntersectPure(
	raySt, rayDir *mat.VecDense,
	u0, v0, tol float64,
	maxIter int,
) float64 {
	if f == nil || raySt == nil || rayDir == nil {
		return math.Inf(1)
	}

	if raySt.Len() != rayDir.Len() || raySt.Len() < 3 {
		return math.Inf(1)
	}

	if len(f.URange) < 2 || len(f.VRange) < 2 {
		return math.Inf(1)
	}

	equations := func(x []float64) []float64 {
		t, u, v := x[0], x[1], x[2]

		pointOnSurface := f.Function(u, v)
		if pointOnSurface == nil || pointOnSurface.Len() < 3 {
			return []float64{
				math.Inf(1),
				math.Inf(1),
				math.Inf(1),
			}
		}

		result := make([]float64, 3)

		for i := 0; i < 3; i++ {
			pointOnRayI := raySt.AtVec(i) + t*rayDir.AtVec(i)
			result[i] = pointOnRayI - pointOnSurface.AtVec(i)
		}

		return result
	}

	options := &maths.NewtonOptions{
		Tol:         tol,
		MaxIter:     maxIter,
		JacobianEps: 1e-6,
		Damping:     true,
	}

	x0 := []float64{utils.EPS, u0, v0}

	solution, err := maths.NewtonRaphson(equations, x0, options)
	if err != nil {
		return math.Inf(1)
	}

	t, u, v := solution[0], solution[1], solution[2]

	if t <= utils.EPS {
		return math.Inf(1)
	}

	if u < f.URange[0] || u > f.URange[1] {
		return math.Inf(1)
	}

	if v < f.VRange[0] || v > f.VRange[1] {
		return math.Inf(1)
	}

	return t
}

func (f *ParametricEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return maths.Normalize(res)
}

func (f *ParametricEquation) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return f.Function(f.URange[0], f.VRange[0]), f.Function(f.URange[1], f.VRange[1])
}
