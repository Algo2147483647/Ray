package shape

import (
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

// Shape exposes the default Euclidean affine-ray intersection and a separate
// geometry-aware geodesic intersection. Callers choose the appropriate entry.
type Shape interface {
	Name() string
	IntersectAffine(rayStart, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool)
	IntersectGeodesic(
		rayStart, rayDir *mat.VecDense,
		g geometry.Geometry,
		options IntersectOptions,
	) (SurfaceInteraction, bool)
	GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense
	BuildBoundingBox() (pmin, pmax *mat.VecDense)
}

// BaseShape provides the basic shape implementation.
type BaseShape struct{}

func (bs *BaseShape) Name() string {
	return "Base Shape"
}

func (bs *BaseShape) IntersectGeodesic(
	_, _ *mat.VecDense,
	_ geometry.Geometry,
	_ IntersectOptions,
) (SurfaceInteraction, bool) {
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
