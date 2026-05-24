package shape

import (
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
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

func (b *BoundedShape) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := b.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (b *BoundedShape) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	t0, t1, ok := b.Bounds.OverlapRange(raySt, rayDir, tMin, tMax)
	if !ok {
		return SurfaceInteraction{}, false
	}
	return b.Shape.IntersectRange(raySt, rayDir, t0, t1)
}

func (b *BoundedShape) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return b.Shape.GetNormalVector(intersect, res)
}

func (b *BoundedShape) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return b.Bounds.BuildBoundingBox()
}
