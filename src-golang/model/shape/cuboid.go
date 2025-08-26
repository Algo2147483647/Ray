package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
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

func (c *Cuboid) IntersectPure(raySt, rayDir *mat.VecDense) float64 {
	t0 := -math.MaxFloat64
	t1 := math.MaxFloat64

	for dim := 0; dim < raySt.Len(); dim++ {
		rayStDim := raySt.AtVec(dim)
		rayDirDim := rayDir.AtVec(dim)
		pminDim := c.Pmin.AtVec(dim)
		pmaxDim := c.Pmax.AtVec(dim)

		if math.Abs(rayDirDim) < math_lib.EPS && (rayStDim < pminDim || rayStDim > pmaxDim) {
			return math.MaxFloat64
		}

		t0t := (pminDim - rayStDim) / rayDirDim
		t1t := (pmaxDim - rayStDim) / rayDirDim
		if t0t > t1t {
			t0t, t1t = t1t, t0t
		}

		t0 = math.Max(t0, t0t)
		t1 = math.Min(t1, t1t)
		if t0 > t1 || t1 < math_lib.EPS {
			return math.MaxFloat64
		}
	}

	if t0 > math_lib.EPS {
		return t0
	}
	return t1
}

// GetNormalVector 计算交点的法向量
func (c *Cuboid) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	a := mat.NewVecDense(intersect.Len(), nil)
	b := mat.NewVecDense(intersect.Len(), nil)
	res = mat.NewVecDense(intersect.Len(), nil)
	a.SubVec(intersect, c.Pmin)
	b.SubVec(intersect, c.Pmax)

	for i := 0; i < intersect.Len(); i++ {
		if math.Abs(a.AtVec(i)) < math_lib.EPS || math.Abs(b.AtVec(i)) < math_lib.EPS {
			res.SetVec(i, 1)
			return res
		}
	}

	return res
}

// BuildBoundingBox 返回包围盒边界
func (c *Cuboid) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return c.Pmin, c.Pmax
}
