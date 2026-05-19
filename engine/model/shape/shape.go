package shape

import (
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

// Shape represents the interface for geometric shapes.
type Shape interface {
	Name() string
	Intersect(rayStart, rayDir *mat.VecDense) float64
	GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense
	BuildBoundingBox() (pmin, pmax *mat.VecDense)
}

// BaseShape provides the basic shape implementation.
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
		pmin.SetVec(i, -math.MaxFloat64/2) // math.MaxFloat64 / 2 prevents overflow in later calculations.
		pmax.SetVec(i, +math.MaxFloat64/2)
	}
	return
}
