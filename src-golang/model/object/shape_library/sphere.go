package shape_library

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
	"src-golang/model/object"
)

type Sphere struct {
	object.BaseShape

	center    *mat.VecDense
	R         float64
	engraving func(*mat.VecDense) bool
}

// NewSphere 构造函数
func NewSphere(center *mat.VecDense, R float64) *Sphere {
	return &Sphere{center: center, R: R}
}

func (s *Sphere) Name() string {
	return "Sphere"
}

func (s *Sphere) Intersect(raySt, ray *mat.VecDense) float64 {
	// 计算 raySt - center
	rayStCenter := mat.NewVecDense(3, nil)
	rayStCenter.SubVec(raySt, s.center)

	// 计算系数
	A := mat.Dot(ray, ray)
	B := 2 * mat.Dot(ray, rayStCenter)
	C := mat.Dot(rayStCenter, rayStCenter) - s.R*s.R

	// 计算判别式
	Delta := B*B - 4*A*C
	if Delta < 0 {
		return math.MaxFloat64 // 无交点
	}

	Delta = math.Sqrt(Delta)
	root1 := (-B - Delta) / (2 * A)
	root2 := (-B + Delta) / (2 * A)

	// 处理雕刻函数
	if s.engraving != nil {
		intersection := mat.NewVecDense(3, nil)

		if root1 > 0 {
			// 计算交点并归一化
			intersection.AddScaledVec(raySt, root1, ray)
			intersection.SubVec(intersection, s.center)
			math_lib.normalize(intersection)

			if s.engraving(intersection) {
				return root1
			}
		}

		if root2 > 0 {
			// 计算交点并归一化
			intersection.AddScaledVec(raySt, root2, ray)
			intersection.SubVec(intersection, s.center)
			math_lib.normalize(intersection)

			if s.engraving(intersection) {
				return root2
			}
		}

		return math.MaxFloat64
	}

	// 无雕刻函数的情况
	switch {
	case root1 > 0 && root2 > 0:
		return math.Min(root1, root2)
	case root1 > 0:
		return root1
	case root2 > 0:
		return root2
	default:
		return math.MaxFloat64
	}
}

func (s *Sphere) GetNormalVector(intersect *mat.VecDense) *mat.VecDense {
	res := mat.NewVecDense(3, nil)
	res.SubVec(intersect, s.center)
	math_lib.normalize(res)
	return res
}

func (s *Sphere) BuildBoundingBox() (pmax, pmin *mat.VecDense) {
	offset := []float64{s.R, s.R, s.R}
	pmax = mat.NewVecDense(3, nil)
	pmin = mat.NewVecDense(3, nil)

	pmax.AddVec(s.center, mat.NewVecDense(3, offset))
	pmin.SubVec(s.center, mat.NewVecDense(3, offset))
	return pmax, pmin
}
