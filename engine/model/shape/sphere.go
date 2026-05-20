package shape

import (
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type Sphere struct {
	BaseShape
	center *mat.VecDense
	R      float64 `json:"r"`
}

// NewSphere is the constructor.
func NewSphere(center *mat.VecDense, R float64) *Sphere {
	return &Sphere{center: center, R: R}
}

func (s *Sphere) Name() string {
	return "Sphere"
}

func (s *Sphere) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := s.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (s *Sphere) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	t := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(t)
	}()

	// Compute coefficients.
	t.SubVec(raySt, s.center)
	A := mat.Dot(rayDir, rayDir)
	B := 2 * mat.Dot(rayDir, t)
	Delta := B*B - 4*A*(mat.Dot(t, t)-s.R*s.R)
	if Delta < 0 {
		return SurfaceInteraction{}, false
	}

	Delta = math.Sqrt(Delta)
	root1 := (-B - Delta) / (2 * A)
	root2 := (-B + Delta) / (2 * A)

	distance := math.MaxFloat64
	switch {
	case distanceInRange(root1, tMin, tMax) && distanceInRange(root2, tMin, tMax):
		distance = math.Min(root1, root2)
	case distanceInRange(root1, tMin, tMax):
		distance = root1
	case distanceInRange(root2, tMin, tMax):
		distance = root2
	}
	if distance == math.MaxFloat64 {
		return SurfaceInteraction{}, false
	}

	if s.EngravingFunc != nil {
		if s.EngravingFunc(map[string]interface{}{
			"ray_start": raySt,
			"ray_dir":   rayDir,
			"distance":  distance,
			"center":    s.center,
			"r":         s.R,
		}) {
			return SurfaceInteraction{}, false
		}
	}

	point := pointAt(raySt, rayDir, distance)
	normal := s.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteractionAt(point, distance, normal), true
}

func (s *Sphere) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return math_lib.Normalize(math_lib.SubVec(res, intersect, s.center))
}

func (s *Sphere) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	dim := s.center.Len()
	offsetData := make([]float64, dim)
	for i := range offsetData {
		offsetData[i] = s.R
	}
	offset := mat.NewVecDense(dim, offsetData)
	pmax = mat.NewVecDense(dim, nil)
	pmin = mat.NewVecDense(dim, nil)
	pmax.AddVec(s.center, offset)
	pmin.SubVec(s.center, offset)
	return
}
