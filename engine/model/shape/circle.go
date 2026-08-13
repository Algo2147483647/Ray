package shape

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type Circle struct {
	BaseShape
	Center *mat.VecDense `json:"center"`
	Normal *mat.VecDense `json:"normal"`
	R      float64       `json:"r"`
}

func NewCircle(center, normal *mat.VecDense, r float64) *Circle {
	normalized := mat.VecDenseCopyOf(normal)
	maths.Normalize(normalized)
	return &Circle{
		Center: center,
		Normal: normalized,
		R:      r,
	}
}

func (c *Circle) Name() string {
	return "Circle"
}

func (c *Circle) IntersectAffine(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if !options.valid() {
		return SurfaceInteraction{}, false
	}
	denominator := mat.Dot(c.Normal, rayDir)
	if math.Abs(denominator) < utils.EPS {
		return SurfaceInteraction{}, false
	}

	toCenter := mat.NewVecDense(raySt.Len(), nil)
	toCenter.SubVec(c.Center, raySt)
	distance := mat.Dot(c.Normal, toCenter) / denominator
	if !distanceInRange(distance, options.Range.Min, options.Range.Max) {
		return SurfaceInteraction{}, false
	}

	hit := mat.NewVecDense(raySt.Len(), nil)
	offset := mat.NewVecDense(raySt.Len(), nil)
	hit.AddScaledVec(raySt, distance, rayDir)
	offset.SubVec(hit, c.Center)
	if mat.Dot(offset, offset) > c.R*c.R+utils.EPS {
		return SurfaceInteraction{}, false
	}

	return newAffineSurfaceInteraction(raySt, rayDir, distance, c.Normal), true
}

func (c *Circle) IntersectGeodesic(rayStart, rayDir *mat.VecDense, g geometry.Geometry, options IntersectOptions) (SurfaceInteraction, bool) {
	if !supportsSphericalGeodesic(g, options) {
		return SurfaceInteraction{}, false
	}
	if c == nil || c.Center == nil || c.Normal == nil || rayStart.Len() != rayDir.Len() {
		return SurfaceInteraction{}, false
	}
	sMin, sMax := options.Range.Min, options.Range.Max
	v, ok := sphericalUnitTangent(rayStart, rayDir)
	if !ok {
		return SurfaceInteraction{}, false
	}

	a := mat.Dot(c.Normal, rayStart)
	b := mat.Dot(c.Normal, v)
	target := mat.Dot(c.Normal, c.Center)
	best := math.Inf(1)
	for _, s := range solveSphericalLinearCoordinate(a, b, target, sMin, sMax) {
		point := sphericalPointAtUnit(rayStart, v, s)
		offset := mat.NewVecDense(point.Len(), nil)
		offset.SubVec(point, c.Center)
		if mat.Dot(offset, offset) <= c.R*c.R+utils.EPS && s < best {
			best = s
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

func (c *Circle) GetNormalVector(_, res *mat.VecDense) *mat.VecDense {
	res.CloneFromVec(c.Normal)
	return res
}

func (c *Circle) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	dim := c.Center.Len()
	pmin = mat.NewVecDense(dim, nil)
	pmax = mat.NewVecDense(dim, nil)

	for i := 0; i < dim; i++ {
		axisProjectionRadius := c.R * math.Sqrt(math.Max(0, 1-c.Normal.AtVec(i)*c.Normal.AtVec(i)))
		pmin.SetVec(i, c.Center.AtVec(i)-axisProjectionRadius)
		pmax.SetVec(i, c.Center.AtVec(i)+axisProjectionRadius)
	}

	return pmin, pmax
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
