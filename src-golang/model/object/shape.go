package object

import "gonum.org/v1/gonum/spatial/r3"

// Shape 表示几何形状的接口
type Shape interface {
	Name() string
	Intersect(rayStart, rayDir r3.Vec) float64
	NormalVector(intersect r3.Vec) r3.Vec
	BoundingBox() (pmax, pmin r3.Vec)
	SetEngraving(fn func(r3.Vec) bool)
}

// BaseShape 提供形状的基础实现
type BaseShape struct {
	EngravingFunc func(r3.Vec) bool
}

func (bs *BaseShape) Name() string {
	return "base shape"
}

func (bs *BaseShape) Intersect(rayStart, rayDir r3.Vec) float64 {
	return 0
}

func (bs *BaseShape) NormalVector(intersect r3.Vec) r3.Vec {
	return r3.Vec{}
}

func (bs *BaseShape) BoundingBox() (pmax, pmin r3.Vec) {
	return r3.Vec{}, r3.Vec{}
}

func (bs *BaseShape) SetEngraving(fn func(r3.Vec) bool) {
	bs.EngravingFunc = fn
}
