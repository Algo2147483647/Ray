package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
)

type Cuboid struct {
	BaseShape

	Pmin *mat.VecDense // 最小点
	Pmax *mat.VecDense // 最大点
}

func NewCuboid(Pmin, Pmax *mat.VecDense) *Cuboid {
	return &Cuboid{
		Pmin: Pmin,
		Pmax: Pmax,
	}
}

func (c *Cuboid) Intersect(raySt, rayDir *mat.VecDense) float64 {
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
		if t0 > t1 || t1 < 0 {
			return math.MaxFloat64
		}
	}

	if t0 >= 0 {
		return t0
	}
	return t1
}

// GetNormalVector 计算交点的法向量
func (c *Cuboid) GetNormalVector(intersect *mat.VecDense) *mat.VecDense {
	dists := make([]float64, 3)
	for i := 0; i < 3; i++ {
		d1 := math.Abs(intersect.AtVec(i) - c.Pmin.AtVec(i))
		d2 := math.Abs(intersect.AtVec(i) - c.Pmax.AtVec(i))
		dists[i] = math.Min(d1, d2)
	}

	// 确定最近面
	normal := mat.NewVecDense(3, nil)
	switch {
	case dists[0] < dists[1] && dists[0] < dists[2]:
		if intersect.AtVec(0) < (c.Pmin.AtVec(0)+c.Pmax.AtVec(0))/2 {
			normal.SetVec(0, -1)
		} else {
			normal.SetVec(0, 1)
		}
	case dists[1] < dists[2]:
		if intersect.AtVec(1) < (c.Pmin.AtVec(1)+c.Pmax.AtVec(1))/2 {
			normal.SetVec(1, -1)
		} else {
			normal.SetVec(1, 1)
		}
	default:
		if intersect.AtVec(2) < (c.Pmin.AtVec(2)+c.Pmax.AtVec(2))/2 {
			normal.SetVec(2, -1)
		} else {
			normal.SetVec(2, 1)
		}
	}
	return normal
}

// BuildBoundingBox 返回包围盒边界
func (c *Cuboid) BuildBoundingBox() (*mat.VecDense, *mat.VecDense) {
	return c.Pmin, c.Pmax
}
