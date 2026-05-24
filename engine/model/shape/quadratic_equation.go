package shape

import (
	"github.com/Algo2147483647/golang_toolkit/math/basic_algebra"
	"github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type QuadraticEquation struct { // f(x) = x^T A x + b^T x + c
	BaseShape
	A      *mat.Dense    `json:"a"`
	B      *mat.VecDense `json:"b"`
	C      float64       `json:"c"`
	Center [3]float64
	Scale  [3]float64
}

func NewQuadraticEquation(A *mat.Dense, B *mat.VecDense, C float64, centerScale ...[]float64) *QuadraticEquation {
	center, scale := normalizePolynomialCenterScale(centerScale...)
	worldA, worldB, worldC := bakeQuadraticCoefficients(A, B, C, center, scale)
	return &QuadraticEquation{
		A:      worldA,
		B:      worldB,
		C:      worldC,
		Center: center,
		Scale:  scale,
	}
}

func bakeQuadraticCoefficients(a *mat.Dense, b *mat.VecDense, c float64, center, scale [3]float64) (*mat.Dense, *mat.VecDense, float64) {
	if polynomialPlacementIsIdentity(center, scale) {
		return mat.DenseCopyOf(a), mat.VecDenseCopyOf(b), c
	}

	d := mat.NewDense(3, 3, []float64{
		1 / scale[0], 0, 0,
		0, 1 / scale[1], 0,
		0, 0, 1 / scale[2],
	})
	e := mat.NewVecDense(3, []float64{
		-center[0] / scale[0],
		-center[1] / scale[1],
		-center[2] / scale[2],
	})

	var aD mat.Dense
	aD.Mul(a, d)
	var worldA mat.Dense
	worldA.Mul(d.T(), &aD)

	var aPlusAT mat.Dense
	aPlusAT.Add(a, a.T())
	tmp := mat.NewVecDense(3, nil)
	tmp.MulVec(&aPlusAT, e)
	worldB := mat.NewVecDense(3, nil)
	worldB.MulVec(d.T(), tmp)
	worldB.AddVec(worldB, scaledByDiagonal(b, d))

	aE := mat.NewVecDense(3, nil)
	aE.MulVec(a, e)
	worldC := mat.Dot(e, aE) + mat.Dot(b, e) + c

	return &worldA, worldB, worldC
}

func scaledByDiagonal(v *mat.VecDense, d *mat.Dense) *mat.VecDense {
	result := mat.NewVecDense(3, nil)
	for i := 0; i < 3; i++ {
		result.SetVec(i, v.AtVec(i)*d.At(i, i))
	}
	return result
}

func (p *QuadraticEquation) Name() string {
	return "Quadratic Equation"
}

func (p *QuadraticEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := p.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (p *QuadraticEquation) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	t := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(t)
	}()

	t.MulVec(p.A, rayDir)
	a := mat.Dot(rayDir, t)                                                               // Compute the quadratic coefficient: rayDir^T A rayDir
	b := 2*mat.Dot(raySt, t) + mat.Dot(p.B, rayDir)                                       // Compute the linear coefficient: 2 * raySt^T A rayDir + b^T rayDir
	c := mat.Dot(raySt, linear_algebra.MulVec(t, p.A, raySt)) + mat.Dot(p.B, raySt) + p.C // Compute the constant term: raySt^T A raySt + b^T raySt + c
	t1, t2, count := basic_algebra.SolveQuadraticEquationReal(a, b, c)                    // Solve the quadratic equation a*t^2 + b*t + c = 0.

	switch count {
	case 1:
		if distanceInRange(t1, tMin, tMax) {
			point := pointAt(raySt, rayDir, t1)
			normal := p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
			return newSurfaceInteractionAt(point, t1, normal), true
		}
	case 2:
		minValidRoots := math.MaxFloat64 // Filter positive roots and choose the smallest one.
		if distanceInRange(t1, tMin, tMax) {
			minValidRoots = math.Min(minValidRoots, t1)
		}
		if distanceInRange(t2, tMin, tMax) {
			minValidRoots = math.Min(minValidRoots, t2)
		}
		if minValidRoots != math.MaxFloat64 {
			point := pointAt(raySt, rayDir, minValidRoots)
			normal := p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
			return newSurfaceInteractionAt(point, minValidRoots, normal), true
		}
	}

	return SurfaceInteraction{}, false
}

func (p *QuadraticEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	// Compute the gradient: df(x) = 2A x + b.
	res.MulVec(p.A, intersect)
	linear_algebra.ScaleVec(res, 2, res)
	return linear_algebra.Normalize(linear_algebra.AddVec(res, res, p.B))
}
