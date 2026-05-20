package shape

import (
	"math"

	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type FiniteCylinder struct {
	BaseShape
	Center *mat.VecDense `json:"center"`
	Axis   *mat.VecDense `json:"axis"`
	R      float64       `json:"r"`
	Height float64       `json:"height"`
}

func NewFiniteCylinder(center, axis *mat.VecDense, r, height float64) *FiniteCylinder {
	normalized := mat.VecDenseCopyOf(axis)
	math_lib.Normalize(normalized)
	return &FiniteCylinder{
		Center: center,
		Axis:   normalized,
		R:      r,
		Height: height,
	}
}

func (c *FiniteCylinder) Name() string {
	return "Finite Cylinder"
}

func (c *FiniteCylinder) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := c.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (c *FiniteCylinder) IntersectPure(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := c.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (c *FiniteCylinder) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	best := c.intersectSide(raySt, rayDir, tMin, tMax)
	best = math.Min(best, c.intersectCap(raySt, rayDir, 0.5*c.Height, tMin, tMax))
	best = math.Min(best, c.intersectCap(raySt, rayDir, -0.5*c.Height, tMin, tMax))
	if best == math.MaxFloat64 {
		return SurfaceInteraction{}, false
	}

	if c.EngravingFunc != nil {
		if c.EngravingFunc(map[string]interface{}{
			"ray_start": raySt,
			"ray_dir":   rayDir,
			"distance":  best,
			"self":      c,
		}) {
			return SurfaceInteraction{}, false
		}
	}

	point := pointAt(raySt, rayDir, best)
	normal := c.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteraction(raySt, rayDir, best, normal), true
}

func (c *FiniteCylinder) intersectSide(raySt, rayDir *mat.VecDense, tMin, tMax float64) float64 {
	oc := utils.VectorPool.Get().(*mat.VecDense)
	dPerp := utils.VectorPool.Get().(*mat.VecDense)
	ocPerp := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(oc)
		utils.VectorPool.Put(dPerp)
		utils.VectorPool.Put(ocPerp)
	}()

	oc.SubVec(raySt, c.Center)
	dParallel := mat.Dot(rayDir, c.Axis)
	ocParallel := mat.Dot(oc, c.Axis)

	dPerp.AddScaledVec(rayDir, -dParallel, c.Axis)
	ocPerp.AddScaledVec(oc, -ocParallel, c.Axis)

	a := mat.Dot(dPerp, dPerp)
	if a < utils.EPS {
		return math.MaxFloat64
	}
	b := 2 * mat.Dot(ocPerp, dPerp)
	cc := mat.Dot(ocPerp, ocPerp) - c.R*c.R
	discriminant := b*b - 4*a*cc
	if discriminant < 0 {
		return math.MaxFloat64
	}

	sqrtDiscriminant := math.Sqrt(discriminant)
	root1 := (-b - sqrtDiscriminant) / (2 * a)
	root2 := (-b + sqrtDiscriminant) / (2 * a)

	best := math.MaxFloat64
	for _, distance := range []float64{root1, root2} {
		if !distanceInRange(distance, tMin, tMax) {
			continue
		}
		axisDistance := ocParallel + distance*dParallel
		if math.Abs(axisDistance) <= 0.5*c.Height+utils.EPS {
			best = math.Min(best, distance)
		}
	}
	return best
}

func (c *FiniteCylinder) intersectCap(raySt, rayDir *mat.VecDense, axisDistance, tMin, tMax float64) float64 {
	denominator := mat.Dot(c.Axis, rayDir)
	if math.Abs(denominator) < utils.EPS {
		return math.MaxFloat64
	}

	center := utils.VectorPool.Get().(*mat.VecDense)
	toCap := utils.VectorPool.Get().(*mat.VecDense)
	hit := utils.VectorPool.Get().(*mat.VecDense)
	offset := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(center)
		utils.VectorPool.Put(toCap)
		utils.VectorPool.Put(hit)
		utils.VectorPool.Put(offset)
	}()

	center.AddScaledVec(c.Center, axisDistance, c.Axis)
	toCap.SubVec(center, raySt)
	distance := mat.Dot(c.Axis, toCap) / denominator
	if !distanceInRange(distance, tMin, tMax) {
		return math.MaxFloat64
	}

	hit.AddScaledVec(raySt, distance, rayDir)
	offset.SubVec(hit, center)
	if mat.Dot(offset, offset) > c.R*c.R+utils.EPS {
		return math.MaxFloat64
	}
	return distance
}

func (c *FiniteCylinder) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	offset := utils.VectorPool.Get().(*mat.VecDense)
	defer utils.VectorPool.Put(offset)

	offset.SubVec(intersect, c.Center)
	axisDistance := mat.Dot(offset, c.Axis)
	if math.Abs(axisDistance-0.5*c.Height) < utils.EPS {
		res.CloneFromVec(c.Axis)
		return res
	}
	if math.Abs(axisDistance+0.5*c.Height) < utils.EPS {
		res.ScaleVec(-1, c.Axis)
		return res
	}

	res.AddScaledVec(offset, -axisDistance, c.Axis)
	return math_lib.Normalize(res)
}

func (c *FiniteCylinder) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	dim := c.Center.Len()
	pmin = mat.NewVecDense(dim, nil)
	pmax = mat.NewVecDense(dim, nil)

	for i := 0; i < dim; i++ {
		axisComponent := c.Axis.AtVec(i)
		extent := 0.5*c.Height*math.Abs(axisComponent) + c.R*math.Sqrt(math.Max(0, 1-axisComponent*axisComponent))
		pmin.SetVec(i, c.Center.AtVec(i)-extent)
		pmax.SetVec(i, c.Center.AtVec(i)+extent)
	}

	return pmin, pmax
}
