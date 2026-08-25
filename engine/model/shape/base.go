package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"gonum.org/v1/gonum/mat"
	"math"
)

// Shape exposes distinct affine-ray and geometry-aware geodesic intersection
// contracts. Callers choose the representation appropriate to the geometry.
type Shape interface {
	Name() string
	IntersectAffine(rayStart, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool)
	IntersectGeodesic(rayStart, rayDir *mat.VecDense, g geometry.Geometry, options IntersectOptions) (SurfaceInteraction, bool)
	GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense
	BuildBoundingBox() (pmin, pmax *mat.VecDense)
}

// SurfaceSample is a point sampled with respect to surface area.
type SurfaceSample struct {
	Point   *mat.VecDense
	Normal  *mat.VecDense
	UV      [2]float64
	PDFArea float64
}

// SurfaceSampler is implemented by finite shapes that can be used as area
// lights. Infinite and implicit shapes intentionally do not satisfy it.
type SurfaceSampler interface {
	SampleSurface(u maths.Sample2D) (SurfaceSample, bool)
	SurfaceArea() float64
}

// BaseShape provides the basic shape implementation.
type BaseShape struct {
	Dimension int
}

func (bs *BaseShape) dimension() int {
	if bs != nil && bs.Dimension > 0 {
		return bs.Dimension
	}
	return 3
}

func (bs *BaseShape) Name() string {
	return "Base Shape"
}

func (bs *BaseShape) IntersectAffine(
	_, _ *mat.VecDense,
	_ IntersectOptions,
) (SurfaceInteraction, bool) {
	return SurfaceInteraction{}, false
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
	dimension := bs.dimension()
	pmin = mat.NewVecDense(dimension, nil)
	pmax = mat.NewVecDense(dimension, nil)
	for i := 0; i < dimension; i++ {
		pmin.SetVec(i, -math.MaxFloat64/2) // math.MaxFloat64 / 2 prevents overflow in later calculations.
		pmax.SetVec(i, +math.MaxFloat64/2)
	}
	return
}
