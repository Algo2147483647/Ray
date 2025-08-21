package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
)

// Shape 表示几何形状的接口
type Shape interface {
	Name() string
	Intersect(rayStart, rayDir *mat.VecDense) float64
	GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense
	BuildBoundingBox() (pmin, pmax *mat.VecDense)
	SetEngraving(fn func(*mat.VecDense) bool)
}

// BaseShape 提供形状的基础实现
type BaseShape struct {
	EngravingFunc func(*mat.VecDense) bool
}

func (bs *BaseShape) Name() string {
	return "Base Shape"
}

func (bs *BaseShape) Intersect(rayStart, rayDir *mat.VecDense) float64 {
	return 0
}

func (bs *BaseShape) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return res
}

func (bs *BaseShape) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	maxVal := math.MaxFloat64 / 2 // 避免后续计算溢出
	pmin = mat.NewVecDense(3, []float64{-maxVal, -maxVal, -maxVal})
	pmax = mat.NewVecDense(3, []float64{+maxVal, +maxVal, +maxVal})
	return
}

func (bs *BaseShape) SetEngraving(fn func(*mat.VecDense) bool) {
	bs.EngravingFunc = fn
}
