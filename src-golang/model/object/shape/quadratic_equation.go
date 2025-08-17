package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
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
	A_rayDir := mat.NewVecDense(3, nil) // 计算二次项系数: rayDir^T A rayDir
	A_raySt := mat.NewVecDense(3, nil)  // 计算一次项系数: 2 * raySt^T A rayDir + b^T rayDir
	A_rayDir.MulVec(p.A, rayDir)
	A_raySt.MulVec(p.A, raySt)
	a := mat.Dot(rayDir, A_rayDir)
	b := 2*mat.Dot(raySt, A_rayDir) + mat.Dot(p.B, rayDir)
	c := mat.Dot(raySt, A_raySt) + mat.Dot(p.B, raySt) + p.C // 计算常数项: raySt^T A raySt + b^T raySt + c

	t1, t2, count := math_lib.SolveQuadraticEquationReal(a, b, c) // 解二次方程 a*t^2 + b*t + c = 0

	minT := 0.0 // 寻找最小正实数解
	found := false
	switch count {
	case 1:
		if t1 > math_lib.EPS {
			minT = t1
			found = true
		}
	case 2:
		if t1 > math_lib.EPS && t2 > math_lib.EPS {
			minT = math.Min(t1, t2)
			found = true
		} else if t1 > math_lib.EPS {
			minT = t1
			found = true
		} else if t2 > math_lib.EPS {
			minT = t2
			found = true
		}
	}

	if !found {
		return math.MaxFloat64
	}
	return minT
}

func (p *QuadraticEquation) GetNormalVector(intersect *mat.VecDense) *mat.VecDense {
	// 计算梯度: ∇f(x) = 2A x + b
	normal := mat.NewVecDense(3, nil)
	normal.MulVec(p.A, intersect)
	return math_lib.Normalize(math_lib.AddVec(normal, math_lib.ScaleVec(normal, 2, normal), p.B))
}
