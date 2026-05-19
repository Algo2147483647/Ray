package shape

import (
	"math"

	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
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
	math_lib.Normalize(normalized)
	return &Circle{
		Center: center,
		Normal: normalized,
		R:      r,
	}
}

func (c *Circle) Name() string {
	return "Circle"
}

func (c *Circle) Intersect(raySt, rayDir *mat.VecDense) float64 {
	distance := c.IntersectPure(raySt, rayDir)

	if c.EngravingFunc != nil && distance != math.MaxFloat64 {
		if c.EngravingFunc(map[string]interface{}{
			"ray_start": raySt,
			"ray_dir":   rayDir,
			"distance":  distance,
			"self":      c,
		}) {
			return math.MaxFloat64
		}
	}
	return distance
}

func (c *Circle) IntersectPure(raySt, rayDir *mat.VecDense) float64 {
	denominator := mat.Dot(c.Normal, rayDir)
	if math.Abs(denominator) < utils.EPS {
		return math.MaxFloat64
	}

	toCenter := utils.VectorPool.Get().(*mat.VecDense)
	hit := utils.VectorPool.Get().(*mat.VecDense)
	offset := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(toCenter)
		utils.VectorPool.Put(hit)
		utils.VectorPool.Put(offset)
	}()

	toCenter.SubVec(c.Center, raySt)
	distance := mat.Dot(c.Normal, toCenter) / denominator
	if distance <= utils.EPS {
		return math.MaxFloat64
	}

	hit.AddScaledVec(raySt, distance, rayDir)
	offset.SubVec(hit, c.Center)
	if mat.Dot(offset, offset) > c.R*c.R+utils.EPS {
		return math.MaxFloat64
	}

	return distance
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
