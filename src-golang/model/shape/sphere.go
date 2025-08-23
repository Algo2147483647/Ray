package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
	"src-golang/utils"
)

type Sphere struct {
	BaseShape
	center *mat.VecDense `json:"center"`
	R      float64       `json:"r"`
}

// NewSphere 构造函数
func NewSphere(center *mat.VecDense, R float64) *Sphere {
	return &Sphere{center: center, R: R}
}

func (s *Sphere) Name() string {
	return "Sphere"
}

func (s *Sphere) Intersect(raySt, rayDir *mat.VecDense) float64 {
	distance := s.IntersectPure(raySt, rayDir)

	if s.EngravingFunc != nil && distance != math.MaxFloat64 {
		if s.EngravingFunc(map[string]interface{}{
			"ray_start": raySt,
			"ray_dir":   rayDir,
			"distance":  distance,
			"center":    s.center,
			"r":         s.R,
		}) {
			return math.MaxFloat64
		}
	}
	return distance
}

func (s *Sphere) IntersectPure(raySt, rayDir *mat.VecDense) float64 {
	t := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(t)
	}()

	// 计算系数
	t.SubVec(raySt, s.center)
	A := mat.Dot(rayDir, rayDir)
	B := 2 * mat.Dot(rayDir, t)
	Delta := B*B - 4*A*(mat.Dot(t, t)-s.R*s.R)
	if Delta < 0 {
		return math.MaxFloat64 // 无交点
	}

	Delta = math.Sqrt(Delta)
	root1 := (-B - Delta) / (2 * A)
	root2 := (-B + Delta) / (2 * A)

	switch {
	case root1 > math_lib.EPS && root2 > math_lib.EPS:
		return math.Min(root1, root2)
	case root1 > math_lib.EPS || root2 > math_lib.EPS:
		return math.Max(root1, root2)
	default:
		return math.MaxFloat64
	}
}

func (s *Sphere) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return math_lib.Normalize(math_lib.SubVec(res, intersect, s.center))
}

func (s *Sphere) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	offset := mat.NewVecDense(3, []float64{s.R, s.R, s.R})
	pmax = mat.NewVecDense(3, nil)
	pmin = mat.NewVecDense(3, nil)
	pmax.AddVec(s.center, offset)
	pmin.SubVec(s.center, offset)
	return
}
