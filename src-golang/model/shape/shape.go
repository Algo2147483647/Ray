package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/utils"
)

// Shape 表示几何形状的接口
type Shape interface {
	Name() string
	Intersect(rayStart, rayDir *mat.VecDense) float64
	GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense
	BuildBoundingBox() (pmin, pmax *mat.VecDense)
}

// BaseShape 提供形状的基础实现
type BaseShape struct {
	EngravingFunc func(data map[string]interface{}) bool
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
	pmin = mat.NewVecDense(utils.Dimension, nil)
	pmax = mat.NewVecDense(utils.Dimension, nil)
	for i := 0; i < utils.Dimension; i++ {
		pmin.SetVec(i, -math.MaxFloat64/2) // math.MaxFloat64 / 2  避免后续计算溢出
		pmax.SetVec(i, +math.MaxFloat64/2)
	}
	return
}
