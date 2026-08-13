package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type Cuboid struct {
	BaseShape

	Pmin *mat.VecDense `json:"pmin"`
	Pmax *mat.VecDense `json:"pmax"`
}

func NewCuboid(Pmin, Pmax *mat.VecDense) *Cuboid {
	return &Cuboid{
		Pmin: Pmin,
		Pmax: Pmax,
	}
}

func (c *Cuboid) Name() string {
	return "Cuboid"
}

func (c *Cuboid) IntersectAffine(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if !options.valid() {
		return SurfaceInteraction{}, false
	}
	tMin, tMax := options.Range.Min, options.Range.Max
	t0, t1, ok := c.intersectionInterval(raySt, rayDir)
	if !ok || t1 < tMin || t0 > tMax {
		return SurfaceInteraction{}, false
	}

	distance := t1
	if t0 >= tMin {
		distance = t0
	}
	if !distanceInRange(distance, tMin, tMax) {
		return SurfaceInteraction{}, false
	}

	point := affinePointAt(raySt, rayDir, distance)
	normal := c.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteractionAt(point, distance, normal), true
}

func (c *Cuboid) IntersectGeodesic(rayStart, rayDir *mat.VecDense, g geometry.Geometry, options IntersectOptions) (SurfaceInteraction, bool) {
	if !supportsSphericalGeodesic(g, options) {
		return SurfaceInteraction{}, false
	}
	if c == nil || c.Pmin == nil || c.Pmax == nil || rayStart.Len() != rayDir.Len() {
		return SurfaceInteraction{}, false
	}
	sMin, sMax := options.Range.Min, options.Range.Max

	v, ok := sphericalUnitTangent(rayStart, rayDir)
	if !ok {
		return SurfaceInteraction{}, false
	}

	best := math.Inf(1)
	for axis := 0; axis < rayStart.Len(); axis++ {
		for _, bound := range []float64{c.Pmin.AtVec(axis), c.Pmax.AtVec(axis)} {
			for _, s := range solveSphericalLinearCoordinate(rayStart.AtVec(axis), v.AtVec(axis), bound, sMin, sMax) {
				if s >= best {
					continue
				}
				point := sphericalPointAtUnit(rayStart, v, s)
				if c.containsPoint(point, axis) {
					best = s
				}
			}
		}
	}

	if math.IsInf(best, 1) {
		return SurfaceInteraction{}, false
	}
	point := sphericalPointAtUnit(rayStart, v, best)
	normal := c.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return SurfaceInteraction{
		Distance:        best,
		ArcLength:       best,
		Point:           point,
		GeometricNormal: normal,
		ShadingNormal:   normal,
		PrimitiveID:     -1,
	}, true
}

// ClipAffine intersects an affine ray with the cuboid and returns the portion
// of options.Range that lies inside it.
func (c *Cuboid) ClipAffine(raySt, rayDir *mat.VecDense, options IntersectOptions) (Interval, bool) {
	if !options.valid() {
		return Interval{}, false
	}
	tMin, tMax := options.Range.Min, options.Range.Max
	t0, t1, ok := c.intersectionInterval(raySt, rayDir)
	if !ok || t1 < tMin || t0 > tMax {
		return Interval{}, false
	}
	if t0 < tMin {
		t0 = tMin
	}
	if t1 > tMax {
		t1 = tMax
	}
	return Interval{Min: t0, Max: t1}, true
}

func (c *Cuboid) intersectionInterval(raySt, rayDir *mat.VecDense) (float64, float64, bool) {
	if c.Pmin.Len() == 3 && c.Pmax.Len() == 3 && raySt.Len() == 3 && rayDir.Len() == 3 {
		return c.intersectionInterval3D(raySt, rayDir)
	}

	t0 := -math.MaxFloat64
	t1 := math.MaxFloat64

	for dim := 0; dim < raySt.Len(); dim++ {
		rayStDim := raySt.AtVec(dim)
		rayDirDim := rayDir.AtVec(dim)
		pminDim := c.Pmin.AtVec(dim)
		pmaxDim := c.Pmax.AtVec(dim)

		if math.Abs(rayDirDim) < utils.EPS {
			if rayStDim < pminDim || rayStDim > pmaxDim {
				return 0, 0, false
			}
			continue
		}

		t0t := (pminDim - rayStDim) / rayDirDim
		t1t := (pmaxDim - rayStDim) / rayDirDim
		if t0t > t1t {
			t0t, t1t = t1t, t0t
		}

		t0 = math.Max(t0, t0t)
		t1 = math.Min(t1, t1t)
		if t0 > t1 || t1 < utils.EPS {
			return 0, 0, false
		}
	}
	return t0, t1, true
}

func (c *Cuboid) intersectionInterval3D(raySt, rayDir *mat.VecDense) (float64, float64, bool) {
	ox, oy, oz := raySt.AtVec(0), raySt.AtVec(1), raySt.AtVec(2)
	dx, dy, dz := rayDir.AtVec(0), rayDir.AtVec(1), rayDir.AtVec(2)
	minX, minY, minZ := c.Pmin.AtVec(0), c.Pmin.AtVec(1), c.Pmin.AtVec(2)
	maxX, maxY, maxZ := c.Pmax.AtVec(0), c.Pmax.AtVec(1), c.Pmax.AtVec(2)

	t0 := -math.MaxFloat64
	t1 := math.MaxFloat64

	if !updateIntervalAxis(ox, dx, minX, maxX, &t0, &t1) {
		return 0, 0, false
	}
	if !updateIntervalAxis(oy, dy, minY, maxY, &t0, &t1) {
		return 0, 0, false
	}
	if !updateIntervalAxis(oz, dz, minZ, maxZ, &t0, &t1) {
		return 0, 0, false
	}
	return t0, t1, true
}

func updateIntervalAxis(origin, direction, pmin, pmax float64, t0, t1 *float64) bool {
	if math.Abs(direction) < utils.EPS {
		return origin >= pmin && origin <= pmax
	}

	near := (pmin - origin) / direction
	far := (pmax - origin) / direction
	if near > far {
		near, far = far, near
	}

	if near > *t0 {
		*t0 = near
	}
	if far < *t1 {
		*t1 = far
	}
	return *t0 <= *t1 && *t1 >= utils.EPS
}

// GetNormalVector computes the normal vector at the intersection point.
func (c *Cuboid) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	if res == nil || res.Len() != intersect.Len() {
		res = mat.NewVecDense(intersect.Len(), nil)
	} else {
		res.Zero()
	}

	a := mat.NewVecDense(intersect.Len(), nil)
	b := mat.NewVecDense(intersect.Len(), nil)
	a.SubVec(intersect, c.Pmin)
	b.SubVec(intersect, c.Pmax)

	for i := 0; i < intersect.Len(); i++ {
		if math.Abs(a.AtVec(i)) < utils.EPS {
			res.SetVec(i, -1)
			return res
		}
		if math.Abs(b.AtVec(i)) < utils.EPS {
			res.SetVec(i, 1)
			return res
		}
	}

	return res
}

// BuildBoundingBox returns the bounding box bounds.
func (c *Cuboid) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return c.Pmin, c.Pmax
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

func (c *Cuboid) containsPoint(point *mat.VecDense, hitAxis int) bool {
	for axis := 0; axis < point.Len(); axis++ {
		if axis == hitAxis {
			continue
		}
		x := point.AtVec(axis)
		if x < c.Pmin.AtVec(axis)-utils.EPS || x > c.Pmax.AtVec(axis)+utils.EPS {
			return false
		}
	}
	return true
}
