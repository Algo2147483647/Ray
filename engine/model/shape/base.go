package shape

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"gonum.org/v1/gonum/mat"
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

var (
	_ Shape = (*BoundedShape)(nil)
	_ Shape = (*Circle)(nil)
	_ Shape = (*Cuboid)(nil)
	_ Shape = (*FiniteCylinder)(nil)
	_ Shape = (*ImplicitEquation)(nil)
	_ Shape = (*KleinBottle4D)(nil)
	_ Shape = (*ParametricCurve)(nil)
	_ Shape = (*ParametricEquation)(nil)
	_ Shape = (*Polynomial)(nil)
	_ Shape = (*Sphere)(nil)
	_ Shape = (*Triangle)(nil)
)

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

func unsupportedGeodesicIntersection() (SurfaceInteraction, bool) {
	return SurfaceInteraction{}, false
}

func unboundedBoundingBox(dimension int) (pmin, pmax *mat.VecDense) {
	if dimension <= 0 {
		dimension = 3
	}
	pmin = mat.NewVecDense(dimension, nil)
	pmax = mat.NewVecDense(dimension, nil)
	for i := 0; i < dimension; i++ {
		pmin.SetVec(i, -math.MaxFloat64/2) // math.MaxFloat64 / 2 prevents overflow in later calculations.
		pmax.SetVec(i, +math.MaxFloat64/2)
	}
	return
}
