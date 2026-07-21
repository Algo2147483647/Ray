package shape

import (
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

func (c *Cuboid) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := c.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (c *Cuboid) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
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

	point := pointAt(raySt, rayDir, distance)
	normal := c.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteractionAt(point, distance, normal), true
}

func (c *Cuboid) OverlapsRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) bool {
	_, ok := c.OverlapRangeNear(raySt, rayDir, tMin, tMax)
	return ok
}

func (c *Cuboid) OverlapRangeNear(raySt, rayDir *mat.VecDense, tMin, tMax float64) (float64, bool) {
	t0, _, ok := c.OverlapRange(raySt, rayDir, tMin, tMax)
	return t0, ok
}

func (c *Cuboid) OverlapRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (float64, float64, bool) {
	t0, t1, ok := c.intersectionInterval(raySt, rayDir)
	if !ok || t1 < tMin || t0 > tMax {
		return 0, 0, false
	}
	if t0 < tMin {
		t0 = tMin
	}
	if t1 > tMax {
		t1 = tMax
	}
	return t0, t1, true
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
