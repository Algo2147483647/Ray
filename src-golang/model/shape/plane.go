package shape

import (
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/utils"
)

type Plane struct {
	BaseShape
	A *mat.VecDense `json:"A"` // f(x) = a^T * x + b
	B float64       `json:"b"`
}

func (p *Plane) Name() string {
	return "Plane"
}

func (p *Plane) Intersect(raySt, rayDir *mat.VecDense) float64 {
	t := mat.Dot(p.A, rayDir)
	if math.Abs(t) < utils.EPS {
		return math.MaxFloat64
	}

	d := -(mat.Dot(p.A, raySt) + p.B) / t
	if d > utils.EPS {
		return d
	}
	return math.MaxFloat64
}

func (p *Plane) GetNormalVector(_, res *mat.VecDense) *mat.VecDense {
	res.CloneFromVec(p.A)
	return math_lib.Normalize(res)
}
