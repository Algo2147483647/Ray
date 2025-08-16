package shape

import (
	"gonum.org/v1/gonum/mat"
	"src-golang/math_lib"
)

type QuadraticEquation struct { // f(x) = x^T A x + b^T x + c
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
	return 0
}

func (p *QuadraticEquation) GetNormalVector(intersect *mat.VecDense) *mat.VecDense {
	return math_lib.Normalize(mat.NewVecDense(3, nil))
}
