package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"gonum.org/v1/gonum/mat"
)

type BoundedShape struct {
	BaseShape
	Shape  Shape
	Bounds *Cuboid
}

func NewBoundedShape(inner Shape, bounds *Cuboid) *BoundedShape {
	return &BoundedShape{
		Shape:  inner,
		Bounds: bounds,
	}
}

func (b *BoundedShape) Name() string {
	return "Bounded " + b.Shape.Name()
}

func (b *BoundedShape) IntersectAffine(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if b == nil || b.Shape == nil || b.Bounds == nil || !options.Range.Valid() {
		return SurfaceInteraction{}, false
	}
	clipped, ok := b.Bounds.ClipAffine(raySt, rayDir, options)
	if !ok {
		return SurfaceInteraction{}, false
	}
	options.Range = clipped
	return b.Shape.IntersectAffine(raySt, rayDir, options)
}

func (b *BoundedShape) IntersectGeodesic(
	raySt, rayDir *mat.VecDense,
	g geometry.Geometry,
	options IntersectOptions,
) (SurfaceInteraction, bool) {
	if b == nil || b.Shape == nil || b.Bounds == nil || g == nil || !options.valid() {
		return SurfaceInteraction{}, false
	}
	interaction, ok := b.Shape.IntersectGeodesic(raySt, rayDir, g, options)
	if !ok || interaction.Point == nil || !b.Bounds.containsPoint(interaction.Point, -1) {
		return SurfaceInteraction{}, false
	}
	return interaction, true
}

func (b *BoundedShape) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return b.Shape.GetNormalVector(intersect, res)
}

func (b *BoundedShape) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return b.Bounds.BuildBoundingBox()
}

func (b *BoundedShape) SurfaceArea() float64 {
	if b == nil || b.Shape == nil || b.Bounds == nil || !boundsContainShape(b.Bounds, b.Shape) {
		return 0
	}
	if sampler, ok := b.Shape.(SurfaceSampler); ok {
		return sampler.SurfaceArea()
	}
	return 0
}

func (b *BoundedShape) SampleSurface(u maths.Sample2D) (SurfaceSample, bool) {
	if b == nil || b.SurfaceArea() <= 0 {
		return SurfaceSample{}, false
	}
	sampler, ok := b.Shape.(SurfaceSampler)
	if !ok {
		return SurfaceSample{}, false
	}
	return sampler.SampleSurface(u)
}

func boundsContainShape(bounds *Cuboid, inner Shape) bool {
	innerMin, innerMax := inner.BuildBoundingBox()
	if innerMin == nil || innerMax == nil || bounds.Pmin.Len() != innerMin.Len() {
		return false
	}
	for i := 0; i < innerMin.Len(); i++ {
		if innerMin.AtVec(i) < bounds.Pmin.AtVec(i) || innerMax.AtVec(i) > bounds.Pmax.AtVec(i) {
			return false
		}
	}
	return true
}
