package shape

import (
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type Sphere struct {
	BaseShape
	center    *mat.VecDense
	centerXYZ [3]float64
	has3D     bool
	R         float64 `json:"r"`
}

// NewSphere is the constructor.
func NewSphere(center *mat.VecDense, R float64) *Sphere {
	s := &Sphere{center: center, R: R}
	if center.Len() == 3 {
		s.centerXYZ = vecDenseXYZ(center)
		s.has3D = true
	}
	return s
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
	candidate, ok := s.IntersectCandidate(raySt, rayDir, tMin, tMax)
	if !ok {
		return SurfaceInteraction{}, false
	}
	interaction := SurfaceInteractionFromCandidate(raySt, rayDir, candidate)
	interaction.GeometricNormal = s.GetNormalVector(interaction.Point, mat.NewVecDense(interaction.Point.Len(), nil))
	interaction.ShadingNormal = interaction.GeometricNormal
	return interaction, true
}

func (s *Sphere) IntersectCandidate(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceCandidate, bool) {
	if s.has3D && raySt.Len() == 3 && rayDir.Len() == 3 {
		return s.intersectCandidate3D(raySt, rayDir, tMin, tMax)
	}

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
		return SurfaceCandidate{}, false
	}

	Delta = math.Sqrt(Delta)
	root1 := (-B - Delta) / (2 * A)
	root2 := (-B + Delta) / (2 * A)

	distance := closestDistance(root1, root2, tMin, tMax)
	if distance == math.MaxFloat64 {
		return SurfaceCandidate{}, false
	}

	if s.EngravingFunc != nil {
		if s.EngravingFunc(map[string]interface{}{
			"ray_start": raySt,
			"ray_dir":   rayDir,
			"distance":  distance,
			"center":    s.center,
			"r":         s.R,
		}) {
			return SurfaceCandidate{}, false
		}
	}

	return SurfaceCandidate{Distance: distance, PrimitiveID: -1}, true
}

func (s *Sphere) intersectCandidate3D(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceCandidate, bool) {
	ox := raySt.AtVec(0) - s.centerXYZ[0]
	oy := raySt.AtVec(1) - s.centerXYZ[1]
	oz := raySt.AtVec(2) - s.centerXYZ[2]
	dx, dy, dz := rayDir.AtVec(0), rayDir.AtVec(1), rayDir.AtVec(2)

	a := dx*dx + dy*dy + dz*dz
	b := 2 * (dx*ox + dy*oy + dz*oz)
	c := ox*ox + oy*oy + oz*oz - s.R*s.R
	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return SurfaceCandidate{}, false
	}

	sqrtDiscriminant := math.Sqrt(discriminant)
	root1 := (-b - sqrtDiscriminant) / (2 * a)
	root2 := (-b + sqrtDiscriminant) / (2 * a)
	distance := closestDistance(root1, root2, tMin, tMax)
	if distance == math.MaxFloat64 {
		return SurfaceCandidate{}, false
	}

	if s.EngravingFunc != nil {
		if s.EngravingFunc(map[string]interface{}{
			"ray_start": raySt,
			"ray_dir":   rayDir,
			"distance":  distance,
			"center":    s.center,
			"r":         s.R,
		}) {
			return SurfaceCandidate{}, false
		}
	}

	return SurfaceCandidate{Distance: distance, PrimitiveID: -1}, true
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
