package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
)

type Plane struct {
	BaseShape
	A, B, C, D float64
}

func (p *Plane) GetName() string {
	return "Plane"
}

func (p *Plane) Intersect(raySt, rayDir *mat.VecDense) float64 {
	t := p.A*rayDir.AtVec(0) + p.B*rayDir.AtVec(1) + p.C*rayDir.AtVec(2)
	if t < 0 {
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

func (p *Plane) BuildBoundingBox() (*mat.VecDense, *mat.VecDense) {
	maxVal := math.MaxFloat64 / 2 // 避免后续计算溢出
	minVec := mat.NewVecDense(3, []float64{-maxVal, -maxVal, -maxVal})
	maxVec := mat.NewVecDense(3, []float64{maxVal, maxVal, maxVal})
	return minVec, maxVec
}
