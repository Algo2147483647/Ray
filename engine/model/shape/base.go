package shape

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

// Shape represents the interface for geometric shapes.
type Shape interface {
	Name() string
	Intersect(rayStart, rayDir *mat.VecDense) float64
	IntersectRange(rayStart, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool)
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
	return math.MaxFloat64
}

func (bs *BaseShape) IntersectRange(rayStart, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	return SurfaceInteraction{}, false
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

func normalizePolynomialCenterScale(args ...[]float64) ([3]float64, [3]float64) {
	center := [3]float64{}
	scale := [3]float64{1, 1, 1}

	if len(args) > 0 && args[0] != nil {
		if len(args[0]) != utils.Dimension {
			panic(fmt.Sprintf("center must contain %d values, got %d", utils.Dimension, len(args[0])))
		}
		copy(center[:], args[0])
	}
	if len(args) > 1 && args[1] != nil {
		if len(args[1]) != utils.Dimension {
			panic(fmt.Sprintf("scale must contain %d values, got %d", utils.Dimension, len(args[1])))
		}
		for i, value := range args[1] {
			if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				panic(fmt.Sprintf("scale index %d must be a finite positive number", i))
			}
			scale[i] = value
		}
	}
	return center, scale
}

func polynomialPlacementIsIdentity(center, scale [3]float64) bool {
	return center == [3]float64{} && scale == [3]float64{1, 1, 1}
}

func polynomialHomogeneousMatrix3D(center, scale [3]float64) [4][4]float64 {
	return [4][4]float64{
		{1, 0, 0, 0},
		{-center[0] / scale[0], 1 / scale[0], 0, 0},
		{-center[1] / scale[1], 0, 1 / scale[1], 0},
		{-center[2] / scale[2], 0, 0, 1 / scale[2]},
	}
}
