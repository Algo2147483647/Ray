package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
	"src-golang/utils"
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
	t := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(t)
	}()

	t.MulVec(p.A, rayDir)
	a := mat.Dot(rayDir, t)                                                         // 计算二次项系数: rayDir^T A rayDir
	b := 2*mat.Dot(raySt, t) + mat.Dot(p.B, rayDir)                                 // 计算一次项系数: 2 * raySt^T A rayDir + b^T rayDir
	c := mat.Dot(raySt, math_lib.MulVec(t, p.A, raySt)) + mat.Dot(p.B, raySt) + p.C // 计算常数项: raySt^T A raySt + b^T raySt + c
	t1, t2, count := math_lib.SolveQuadraticEquationReal(a, b, c)                   // 解二次方程 a*t^2 + b*t + c = 0

	switch count {
	case 1:
		if t1 > math_lib.EPS {
			return t1
		}
	case 2:
		minValidRoots := math.MaxFloat64 // 过滤出所有正根，选择其中最小的
		if t1 != math.MaxFloat64 && t1 > math_lib.EPS {
			minValidRoots = math.Min(minValidRoots, t1)
		}
		if t2 != math.MaxFloat64 && t2 > math_lib.EPS {
			minValidRoots = math.Min(minValidRoots, t2)
		}
		return minValidRoots
	}

	return math.MaxFloat64
}

func (p *QuadraticEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	// 计算梯度: ∇f(x) = 2A x + b
	res.MulVec(p.A, intersect)
	math_lib.ScaleVec(res, 2, res)
	return math_lib.Normalize(math_lib.AddVec(res, res, p.B))
}
