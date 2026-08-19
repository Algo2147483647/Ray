package shape

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
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
	maths.Normalize(normalized)
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

func (c *FiniteCylinder) IntersectAffine(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if !options.valid() {
		return SurfaceInteraction{}, false
	}
	tMin, tMax := options.Range.Min, options.Range.Max
	best := c.intersectSide(raySt, rayDir, tMin, tMax)
	best = math.Min(best, c.intersectCap(raySt, rayDir, 0.5*c.Height, tMin, tMax))
	best = math.Min(best, c.intersectCap(raySt, rayDir, -0.5*c.Height, tMin, tMax))
	if best == math.MaxFloat64 {
		return SurfaceInteraction{}, false
	}

	point := affinePointAt(raySt, rayDir, best)
	normal := c.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteractionAt(point, best, normal), true
}

func (c *FiniteCylinder) intersectSide(raySt, rayDir *mat.VecDense, tMin, tMax float64) float64 {
	dim := raySt.Len()
	oc := mat.NewVecDense(dim, nil)
	dPerp := mat.NewVecDense(dim, nil)
	ocPerp := mat.NewVecDense(dim, nil)

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

	dim := raySt.Len()
	center := mat.NewVecDense(dim, nil)
	toCap := mat.NewVecDense(dim, nil)
	hit := mat.NewVecDense(dim, nil)
	offset := mat.NewVecDense(dim, nil)

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
	offset := mat.NewVecDense(intersect.Len(), nil)

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
	return maths.Normalize(res)
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

// SurfaceArea returns the area of the complete closed cylinder: its lateral
// surface plus both circular caps.
func (c *FiniteCylinder) SurfaceArea() float64 {
	if c == nil || c.Center == nil || c.Axis == nil || c.Center.Len() != 3 || c.Axis.Len() != 3 ||
		c.R <= 0 || c.Height <= 0 || math.IsNaN(c.R) || math.IsNaN(c.Height) ||
		math.IsInf(c.R, 0) || math.IsInf(c.Height, 0) {
		return 0
	}
	axisNorm := mat.Norm(c.Axis, 2)
	if axisNorm <= 0 || math.IsNaN(axisNorm) || math.IsInf(axisNorm, 0) {
		return 0
	}
	return 2 * math.Pi * c.R * (c.Height + c.R)
}

// SampleSurface samples the complete closed cylinder uniformly with respect to
// surface area. u.U first selects the lateral surface or one of the two caps in
// exact proportion to its area; the residual coordinate is then remapped to a
// uniform variate within that selected component.
func (c *FiniteCylinder) SampleSurface(u maths.Sample2D) (SurfaceSample, bool) {
	totalArea := c.SurfaceArea()
	if totalArea <= 0 {
		return SurfaceSample{}, false
	}
	frame, ok := maths.NewFrameFromNormal(c.Axis)
	if !ok || frame.Tangent == nil || frame.Bitangent == nil {
		return SurfaceSample{}, false
	}

	sideArea := 2 * math.Pi * c.R * c.Height
	capArea := math.Pi * c.R * c.R
	x := clampUnit(u.U) * totalArea
	v := clampUnit(u.V)

	if x < sideArea {
		heightU := x / sideArea
		phi := 2 * math.Pi * v
		radial := cylinderRadialDirection(frame, phi)
		point := mat.VecDenseCopyOf(c.Center)
		point.AddScaledVec(point, c.Height*(heightU-0.5), frame.Normal)
		point.AddScaledVec(point, c.R, radial)
		return SurfaceSample{
			Point: point, Normal: radial,
			UV: [2]float64{heightU, v}, PDFArea: 1 / totalArea,
		}, true
	}

	x -= sideArea
	capSign := 1.0
	if x >= capArea {
		x -= capArea
		capSign = -1
	}
	diskU := x / capArea
	phi := 2 * math.Pi * v
	radial := cylinderRadialDirection(frame, phi)
	point := mat.VecDenseCopyOf(c.Center)
	point.AddScaledVec(point, capSign*0.5*c.Height, frame.Normal)
	point.AddScaledVec(point, c.R*math.Sqrt(diskU), radial)
	normal := mat.VecDenseCopyOf(frame.Normal)
	normal.ScaleVec(capSign, normal)
	return SurfaceSample{
		Point: point, Normal: normal,
		UV: [2]float64{diskU, v}, PDFArea: 1 / totalArea,
	}, true
}

func cylinderRadialDirection(frame maths.Frame, phi float64) *mat.VecDense {
	radial := mat.NewVecDense(3, nil)
	radial.AddScaledVec(radial, math.Cos(phi), frame.Tangent)
	radial.AddScaledVec(radial, math.Sin(phi), frame.Bitangent)
	return radial
}
