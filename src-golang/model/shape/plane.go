package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
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
	if math.Abs(t) < math_lib.EPS {
		return math.MaxFloat64
	}

	d := -(mat.Dot(p.A, raySt) + p.B) / t
	if d > math_lib.EPS {
		return d
	}
	return math.MaxFloat64
}

func (p *Plane) GetNormalVector(intersect *mat.VecDense) *mat.VecDense {
	return math_lib.Normalize(p.A)
}
