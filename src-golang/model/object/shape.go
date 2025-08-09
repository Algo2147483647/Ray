package object

import "gonum.org/v1/gonum/mat"

// Shape 表示几何形状的接口
type Shape interface {
	Name() string
	Intersect(rayStart, rayDir *mat.VecDense) float64
	NormalVector(intersect *mat.VecDense) *mat.VecDense
	BoundingBox() (pmax, pmin *mat.VecDense)
	SetEngraving(fn func(*mat.VecDense) bool)
}

// BaseShape 提供形状的基础实现
type BaseShape struct {
	EngravingFunc func(*mat.VecDense) bool
}

func (bs *BaseShape) Name() string {
	return "base shape"
}

func (bs *BaseShape) Intersect(rayStart, rayDir *mat.VecDense) float64 {
	return 0
}

func (bs *BaseShape) NormalVector(intersect *mat.VecDense) *mat.VecDense {
	return &mat.VecDense{}
}

func (bs *BaseShape) BoundingBox() (pmax, pmin *mat.VecDense) {
	return &mat.VecDense{}, &mat.VecDense{}
}

func (bs *BaseShape) SetEngraving(fn func(*mat.VecDense) bool) {
	bs.EngravingFunc = fn
}
