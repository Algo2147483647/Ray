package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/utils"
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

// SurfacePDF evaluates the sampler's ordinary surface-area density. Uniform
// area samplers need no method of their own; their density is derived from
// SurfaceArea. A non-uniform sampler may provide a specialized SurfacePDF
// method.
func SurfacePDF(sampler SurfaceSampler, point *mat.VecDense) float64 {
	if sampler == nil {
		return 0
	}
	if specialized, ok := sampler.(interface {
		SurfacePDF(*mat.VecDense) float64
	}); ok {
		return specialized.SurfacePDF(point)
	}
	area := sampler.SurfaceArea()
	if area <= 0 || math.IsNaN(area) || math.IsInf(area, 0) {
		return 0
	}
	return 1 / area
}

// SampleSurfaceFrom samples a surface with respect to a world-space reference
// point when the sampler provides a specialized implementation. Otherwise it
// falls back to the sampler's ordinary surface-area distribution.
func SampleSurfaceFrom(
	sampler SurfaceSampler,
	reference *mat.VecDense,
	u maths.Sample2D,
) (SurfaceSample, bool) {
	if sampler == nil {
		return SurfaceSample{}, false
	}
	if conditioned, ok := sampler.(interface {
		SampleSurfaceFrom(*mat.VecDense, maths.Sample2D) (SurfaceSample, bool)
		SurfacePDFFrom(*mat.VecDense, *mat.VecDense) float64
	}); ok {
		return conditioned.SampleSurfaceFrom(reference, u)
	}
	return sampler.SampleSurface(u)
}

// SurfacePDFFrom evaluates the density used by SampleSurfaceFrom. Every PDF
// remains expressed with respect to surface area.
func SurfacePDFFrom(sampler SurfaceSampler, reference, point *mat.VecDense) float64 {
	if sampler == nil {
		return 0
	}
	if conditioned, ok := sampler.(interface {
		SampleSurfaceFrom(*mat.VecDense, maths.Sample2D) (SurfaceSample, bool)
		SurfacePDFFrom(*mat.VecDense, *mat.VecDense) float64
	}); ok {
		return conditioned.SurfacePDFFrom(reference, point)
	}
	return SurfacePDF(sampler, point)
}

// BaseShape provides the basic shape implementation.
type BaseShape struct{}

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
	pmin = mat.NewVecDense(utils.Dimension, nil)
	pmax = mat.NewVecDense(utils.Dimension, nil)
	for i := 0; i < utils.Dimension; i++ {
		pmin.SetVec(i, -math.MaxFloat64/2) // math.MaxFloat64 / 2 prevents overflow in later calculations.
		pmax.SetVec(i, +math.MaxFloat64/2)
	}
	return
}
