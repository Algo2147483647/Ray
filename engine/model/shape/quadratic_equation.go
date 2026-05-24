package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

// QuadraticEquation f(x) = x^T A x + b^T x + c
type QuadraticEquation struct {
	BaseShape
	A *mat.Dense    `json:"a"`
	B *mat.VecDense `json:"b"`
	C float64       `json:"c"`
}

func NewQuadraticEquation(A *mat.Dense, B *mat.VecDense, C float64) *QuadraticEquation {
	return &QuadraticEquation{
		A: A,
		B: B,
		C: C,
	}
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
	n := raySt.Len()
	if rayDir.Len() != n || p.B.Len() != n {
		return SurfaceInteraction{}, false
	}

	ar, ac := p.A.Dims()
	if ar != n || ac != n {
		return SurfaceInteraction{}, false
	}

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
