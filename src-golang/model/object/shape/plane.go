package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
)

type Plane struct {
	BaseShape
	A float64 `json:"a"`
	B float64 `json:"b"`
	C float64 `json:"c"`
	D float64 `json:"d"`
}

func (p *Plane) Name() string {
	return "Plane"
}

func (p *Plane) Intersect(raySt, rayDir *mat.VecDense) float64 {
	t := p.A*rayDir.AtVec(0) + p.B*rayDir.AtVec(1) + p.C*rayDir.AtVec(2)
	if math.Abs(t) < math_lib.EPS {
		return math.MaxFloat64
	}

	d := -(p.A*raySt.AtVec(0) + p.B*raySt.AtVec(1) + p.C*raySt.AtVec(2) + p.D) / t
	if d > 0 {
		return d
	}
	return math.MaxFloat64
}

func (p *Plane) GetNormalVector(intersect *mat.VecDense) *mat.VecDense {
	res := mat.NewVecDense(3, []float64{p.A, p.B, p.C})
	return math_lib.Normalize(res)
}
