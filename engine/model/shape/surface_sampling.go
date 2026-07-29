package shape

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"gonum.org/v1/gonum/mat"
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

func (s *Sphere) SurfaceArea() float64 {
	if s == nil || s.center == nil || s.center.Len() != 3 || s.R <= 0 {
		return 0
	}
	return 4 * math.Pi * s.R * s.R
}

func (s *Sphere) SampleSurface(u maths.Sample2D) (SurfaceSample, bool) {
	area := s.SurfaceArea()
	if area <= 0 {
		return SurfaceSample{}, false
	}
	z := 1 - 2*clampUnit(u.U)
	r := math.Sqrt(math.Max(0, 1-z*z))
	phi := 2 * math.Pi * clampUnit(u.V)
	normal := mat.NewVecDense(3, []float64{r * math.Cos(phi), r * math.Sin(phi), z})
	point := mat.NewVecDense(3, nil)
	point.AddScaledVec(s.center, s.R, normal)
	return SurfaceSample{Point: point, Normal: normal, UV: [2]float64{u.U, u.V}, PDFArea: 1 / area}, true
}

func (t *Triangle) SurfaceArea() float64 {
	if t == nil || t.P1 == nil || t.P1.Len() != 3 {
		return 0
	}
	cross := maths.Cross2(t.Mem.Edge1, t.Mem.Edge2)
	return 0.5 * mat.Norm(cross, 2)
}

func (t *Triangle) SampleSurface(u maths.Sample2D) (SurfaceSample, bool) {
	area := t.SurfaceArea()
	if area <= 0 {
		return SurfaceSample{}, false
	}
	su := math.Sqrt(clampUnit(u.U))
	b1 := 1 - su
	b2 := clampUnit(u.V) * su
	point := mat.NewVecDense(3, nil)
	point.AddScaledVec(point, b1, t.Mem.Edge1)
	point.AddScaledVec(point, b2, t.Mem.Edge2)
	point.AddVec(point, t.P1)
	normal := mat.VecDenseCopyOf(t.Mem.Normal)
	return SurfaceSample{Point: point, Normal: normal, UV: [2]float64{b1, b2}, PDFArea: 1 / area}, true
}

func (c *Circle) SurfaceArea() float64 {
	if c == nil || c.Center == nil || c.Center.Len() != 3 || c.R <= 0 {
		return 0
	}
	return math.Pi * c.R * c.R
}

func (c *Circle) SampleSurface(u maths.Sample2D) (SurfaceSample, bool) {
	area := c.SurfaceArea()
	if area <= 0 {
		return SurfaceSample{}, false
	}
	frame, ok := maths.NewFrameFromNormal(c.Normal)
	if !ok {
		return SurfaceSample{}, false
	}
	r := c.R * math.Sqrt(clampUnit(u.U))
	phi := 2 * math.Pi * clampUnit(u.V)
	local := maths.NewDirection(r*math.Cos(phi), r*math.Sin(phi), 0)
	point := frame.LocalToWorld(local)
	point.AddVec(point, c.Center)
	return SurfaceSample{
		Point: point, Normal: mat.VecDenseCopyOf(c.Normal),
		UV: [2]float64{u.U, u.V}, PDFArea: 1 / area,
	}, true
}

func (c *Cuboid) SurfaceArea() float64 {
	if c == nil || c.Pmin == nil || c.Pmin.Len() != 3 {
		return 0
	}
	dx := c.Pmax.AtVec(0) - c.Pmin.AtVec(0)
	dy := c.Pmax.AtVec(1) - c.Pmin.AtVec(1)
	dz := c.Pmax.AtVec(2) - c.Pmin.AtVec(2)
	if dx <= 0 || dy <= 0 || dz <= 0 {
		return 0
	}
	return 2 * (dx*dy + dx*dz + dy*dz)
}

func (c *Cuboid) SampleSurface(u maths.Sample2D) (SurfaceSample, bool) {
	area := c.SurfaceArea()
	if area <= 0 {
		return SurfaceSample{}, false
	}
	d := [3]float64{
		c.Pmax.AtVec(0) - c.Pmin.AtVec(0),
		c.Pmax.AtVec(1) - c.Pmin.AtVec(1),
		c.Pmax.AtVec(2) - c.Pmin.AtVec(2),
	}
	faceAreas := [3]float64{d[1] * d[2], d[0] * d[2], d[0] * d[1]}
	x := clampUnit(u.U) * area
	axis, side := 0, 0
	for a := 0; a < 3; a++ {
		for s := 0; s < 2; s++ {
			if x <= faceAreas[a] {
				axis, side = a, s
				goto selected
			}
			x -= faceAreas[a]
		}
	}
	axis, side = 2, 1
selected:
	// Recycle the residual within the selected face and decorrelate the second
	// coordinate. Both coordinates remain uniform on that face.
	a := x / faceAreas[axis]
	b := math.Mod(clampUnit(u.V)+0.6180339887498949*clampUnit(u.U), 1)
	point := mat.VecDenseCopyOf(c.Pmin)
	normal := mat.NewVecDense(3, nil)
	point.SetVec(axis, c.Pmin.AtVec(axis))
	if side == 1 {
		point.SetVec(axis, c.Pmax.AtVec(axis))
		normal.SetVec(axis, 1)
	} else {
		normal.SetVec(axis, -1)
	}
	j, k := (axis+1)%3, (axis+2)%3
	point.SetVec(j, c.Pmin.AtVec(j)+a*d[j])
	point.SetVec(k, c.Pmin.AtVec(k)+b*d[k])
	return SurfaceSample{Point: point, Normal: normal, UV: [2]float64{a, b}, PDFArea: 1 / area}, true
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

func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v >= 1 {
		return math.Nextafter(1, 0)
	}
	return v
}
