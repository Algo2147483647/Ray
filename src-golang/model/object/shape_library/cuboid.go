package shape_library

import (
	"gonum.org/v1/gonum/spatial/r3"
	"math"
	"src-golang/model/object"
)

// Cuboid 表示轴对齐包围盒
type Cuboid struct {
	object.BaseShape

	Pmin      r3.Vec // 最小点
	Pmax      r3.Vec // 最大点
	Engraving func(r3.Vec) bool
}

func NewCuboid(Pmin, Pmax r3.Vec) *Cuboid {
	return &Cuboid{
		Pmin: Pmin,
		Pmax: Pmax,
	}
}

// Intersect 计算光线与包围盒的交点
func (c *Cuboid) Intersect(raySt, rayDir r3.Vec) float64 {
	tmin := (c.Pmin.X - raySt.X) / rayDir.X
	tmax := (c.Pmax.X - raySt.X) / rayDir.X
	if tmin > tmax {
		tmin, tmax = tmax, tmin
	}

	tymin := (c.Pmin.Y - raySt.Y) / rayDir.Y
	tymax := (c.Pmax.Y - raySt.Y) / rayDir.Y
	if tymin > tymax {
		tymin, tymax = tymax, tymin
	}

	if tmin > tymax || tymin > tmax {
		return math.MaxFloat64
	}
	if tymin > tmin {
		tmin = tymin
	}
	if tymax < tmax {
		tmax = tymax
	}

	tzmin := (c.Pmin.Z - raySt.Z) / rayDir.Z
	tzmax := (c.Pmax.Z - raySt.Z) / rayDir.Z
	if tzmin > tzmax {
		tzmin, tzmax = tzmax, tzmin
	}

	if tmin > tzmax || tzmin > tmax {
		return math.MaxFloat64
	}
	if tzmin > tmin {
		tmin = tzmin
	}
	if tzmax < tmax {
		tmax = tzmax
	}

	if tmin > 0 {
		return tmin
	}
	if tmax > 0 {
		return tmax
	}
	return math.MaxFloat64
}

func (c *Cuboid) Intersect2(raySt, rayDir r3.Vec) float64 {
	// 避免除以零
	invDir := r3.Vec{
		X: 1 / rayDir.X,
		Y: 1 / rayDir.Y,
		Z: 1 / rayDir.Z,
	}

	// 计算各轴的相交区间
	tmin := r3.Sub(c.Pmin, raySt).Mul(invDir)
	tmax := r3.Sub(c.Pmax, raySt).Mul(invDir)

	// 确保 tmin < tmax
	if invDir.X < 0 {
		tmin.X, tmax.X = tmax.X, tmin.X
	}
	if invDir.Y < 0 {
		tmin.Y, tmax.Y = tmax.Y, tmin.Y
	}
	if invDir.Z < 0 {
		tmin.Z, tmax.Z = tmax.Z, tmin.Z
	}

	// 计算实际相交区间
	t0 := max(tmin.X, max(tmin.Y, tmin.Z))
	t1 := min(tmax.X, min(tmax.Y, tmax.Z))

	// 检查是否相交
	if t0 > t1 || t1 < 0 {
		return math.MaxFloat64
	}
	if t0 >= 0 {
		return t0
	}
	return t1
}

func (c *Cuboid) GetNormalVector(intersect r3.Vec) r3.Vec {
	normal := r3.Vec{}
	if math.Abs(intersect.X-c.Pmin.X) < EPS {
		normal = r3.Vec{X: -1}
	} else if math.Abs(intersect.X-c.Pmax.X) < EPS {
		normal = r3.Vec{X: 1}
	} else if math.Abs(intersect.Y-c.Pmin.Y) < EPS {
		normal = r3.Vec{Y: -1}
	} else if math.Abs(intersect.Y-c.Pmax.Y) < EPS {
		normal = r3.Vec{Y: 1}
	} else if math.Abs(intersect.Z-c.Pmin.Z) < EPS {
		normal = r3.Vec{Z: -1}
	} else if math.Abs(intersect.Z-c.Pmax.Z) < EPS {
		normal = r3.Vec{Z: 1}
	}
	return normal
}

func (c *Cuboid) BuildBoundingBox() (r3.Vec, r3.Vec) {
	return c.Pmax, c.Pmin
}

// 辅助函数
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
