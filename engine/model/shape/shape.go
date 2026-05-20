package shape

import (
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type SurfaceInteraction struct {
	Distance        float64
	Point           *mat.VecDense
	GeometricNormal *mat.VecDense
	ShadingNormal   *mat.VecDense
	UV              [2]float64
	DPDU            *mat.VecDense
	DPDV            *mat.VecDense
	PrimitiveID     int
}

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

func distanceInRange(distance, tMin, tMax float64) bool {
	return distance >= tMin && distance <= tMax && !math.IsNaN(distance) && !math.IsInf(distance, 0)
}

func pointAt(rayStart, rayDir *mat.VecDense, distance float64) *mat.VecDense {
	point := mat.NewVecDense(rayStart.Len(), nil)
	point.AddScaledVec(rayStart, distance, rayDir)
	return point
}

func newSurfaceInteraction(rayStart, rayDir *mat.VecDense, distance float64, normal *mat.VecDense) SurfaceInteraction {
	point := pointAt(rayStart, rayDir, distance)
	geometricNormal := mat.VecDenseCopyOf(normal)
	shadingNormal := mat.VecDenseCopyOf(normal)
	return SurfaceInteraction{
		Distance:        distance,
		Point:           point,
		GeometricNormal: geometricNormal,
		ShadingNormal:   shadingNormal,
		PrimitiveID:     -1,
	}
}
