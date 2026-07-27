package shape

import "gonum.org/v1/gonum/mat"

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

func (b *BoundedShape) Intersect(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if b == nil || b.Shape == nil || b.Bounds == nil || !options.Range.Valid() {
		return SurfaceInteraction{}, false
	}
	if options.Path == PathGreatCircle {
		interaction, ok := b.Shape.Intersect(raySt, rayDir, options)
		if !ok || interaction.Point == nil || !b.Bounds.containsPoint(interaction.Point, -1) {
			return SurfaceInteraction{}, false
		}
		return interaction, true
	}
	clipped, ok := b.Bounds.Clip(raySt, rayDir, options)
	if !ok {
		return SurfaceInteraction{}, false
	}
	options.Range = clipped
	return b.Shape.Intersect(raySt, rayDir, options)
}

func (b *BoundedShape) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return b.Shape.GetNormalVector(intersect, res)
}

func (b *BoundedShape) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return b.Bounds.BuildBoundingBox()
}
