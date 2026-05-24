package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
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

func (p *QuadraticEquation) IntersectRange(
	raySt, rayDir *mat.VecDense,
	tMin, tMax float64,
) (SurfaceInteraction, bool) {
	if raySt == nil || rayDir == nil || p == nil || p.A == nil || p.B == nil {
		return SurfaceInteraction{}, false
	}

	n := raySt.Len()
	if rayDir.Len() != n || p.B.Len() != n {
		return SurfaceInteraction{}, false
	}

	ar, ac := p.A.Dims()
	if ar != n || ac != n {
		return SurfaceInteraction{}, false
	}

	// Ray:
	//
	//     x(t) = raySt + t * rayDir
	//
	// Surface:
	//
	//     x^T A x + B^T x + C = 0
	//
	// Substitute x(t):
	//
	//     a*t^2 + b*t + c = 0
	//
	// where:
	//
	//     a = d^T A d
	//     b = s^T A d + d^T A s + B^T d
	//     c = s^T A s + B^T s + C
	//
	// If A is guaranteed symmetric, then:
	//
	//     b = 2*s^T A d + B^T d
	//
	// But the general formula below is safer.

	aDir := mat.NewVecDense(n, nil)
	aSt := mat.NewVecDense(n, nil)

	aDir.MulVec(p.A, rayDir) // A * d
	aSt.MulVec(p.A, raySt)   // A * s

	a := mat.Dot(rayDir, aDir)
	b := mat.Dot(raySt, aDir) + mat.Dot(rayDir, aSt) + mat.Dot(p.B, rayDir)
	c := mat.Dot(raySt, aSt) + mat.Dot(p.B, raySt) + p.C

	roots, err := maths.SolveQuadraticEquationReal(a, b, c)
	if err != nil {
		return SurfaceInteraction{}, false
	}

	bestT := math.Inf(1)

	for _, root := range roots {
		if distanceInRange(root, tMin, tMax) && root < bestT {
			bestT = root
		}
	}

	if math.IsInf(bestT, 1) {
		return SurfaceInteraction{}, false
	}

	point := pointAt(raySt, rayDir, bestT)
	normal := p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))

	return newSurfaceInteractionAt(point, bestT, normal), true
}

func (p *QuadraticEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	// Compute the gradient: df(x) = 2A x + b.
	res.MulVec(p.A, intersect)
	maths.ScaleVec(res, 2, res)
	return maths.Normalize(maths.AddVec(res, res, p.B))
}
