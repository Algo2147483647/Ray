package shape_library

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
	"src-golang/model/object"
)

type Cuboid struct {
	object.BaseShape
	
	Pmin *mat.VecDense // 最小点
	Pmax *mat.VecDense // 最大点
}

func NewCuboid(Pmin, Pmax *mat.VecDense) *Cuboid {
	return &Cuboid{
		Pmin: Pmin,
		Pmax: Pmax,
	}
}

// Intersect 计算光线与包围盒的交点
func (c *Cuboid) Intersect(raySt, rayDir *mat.VecDense) float64 {
	// 初始化相交区间
	tmin := math.Inf(-1)
	tmax := math.Inf(1)

	// 辅助函数：计算单个轴的相交区间
	updateInterval := func(axis int) (float64, float64) {
		origin := raySt.AtVec(axis)
		direction := rayDir.AtVec(axis)
		min := c.Pmin.AtVec(axis)
		max := c.Pmax.AtVec(axis)

		if math.Abs(direction) < math_lib.EPS {
			// 光线平行于轴
			if origin < min || origin > max {
				return math.Inf(1), math.Inf(-1) // 无交点
			}
			return tmin, tmax // 保持原区间
		}

		invDir := 1.0 / direction
		t1 := (min - origin) * invDir
		t2 := (max - origin) * invDir

		if t1 > t2 {
			return t2, t1
		}
		return t1, t2
	}

	// 处理X轴
	tx1, tx2 := updateInterval(0)
	tmin = math.Max(tmin, tx1)
	tmax = math.Min(tmax, tx2)
	if tmin > tmax {
		return math.MaxFloat64
	}

	// 处理Y轴
	ty1, ty2 := updateInterval(1)
	tmin = math.Max(tmin, ty1)
	tmax = math.Min(tmax, ty2)
	if tmin > tmax {
		return math.MaxFloat64
	}

	// 处理Z轴
	tz1, tz2 := updateInterval(2)
	tmin = math.Max(tmin, tz1)
	tmax = math.Min(tmax, tz2)
	if tmin > tmax {
		return math.MaxFloat64
	}

	// 检查有效交点
	if tmax < 0 {
		return math.MaxFloat64 // 交点在光线后方
	}

	if tmin > 0 {
		return tmin // 最小正交点
	}
	return tmax // 光线起点在盒内
}

// GetNormalVector 计算交点的法向量
func (c *Cuboid) GetNormalVector(intersect *mat.VecDense) *mat.VecDense {
	// 计算各面距离
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
