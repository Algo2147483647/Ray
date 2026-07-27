package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
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
	s := &Sphere{center: center, R: R}
	return s
}

func (s *Sphere) Name() string {
	return "Sphere"
}

func (s *Sphere) Intersect(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if options.Path == PathGreatCircle {
		return s.intersectGreatCircle(raySt, rayDir, options)
	}
	if !options.validFor(PathAffine) {
		return SurfaceInteraction{}, false
	}
	distance, ok := s.intersectAffine(raySt, rayDir, options.Range)
	if !ok {
		return SurfaceInteraction{}, false
	}
	interaction := newSurfaceInteraction(raySt, rayDir, distance, nil)
	interaction.GeometricNormal = s.GetNormalVector(interaction.Point, mat.NewVecDense(interaction.Point.Len(), nil))
	interaction.ShadingNormal = interaction.GeometricNormal
	return interaction, true
}

func (s *Sphere) intersectAffine(raySt, rayDir *mat.VecDense, interval Interval) (float64, bool) {
	if raySt.Len() == 3 && rayDir.Len() == 3 {
		return s.intersectAffine3D(raySt, rayDir, interval)
	}

	// Compute coefficients.
	t := mat.NewVecDense(raySt.Len(), nil)
	t.SubVec(raySt, s.center)
	A := mat.Dot(rayDir, rayDir)
	B := 2 * mat.Dot(rayDir, t)
	Delta := B*B - 4*A*(mat.Dot(t, t)-s.R*s.R)
	if Delta < 0 {
		return 0, false
	}

	Delta = math.Sqrt(Delta)
	root1 := (-B - Delta) / (2 * A)
	root2 := (-B + Delta) / (2 * A)

	distance := closestDistance(root1, root2, interval.Min, interval.Max)
	if distance == math.MaxFloat64 {
		return 0, false
	}

	return distance, true
}

func (s *Sphere) intersectAffine3D(raySt, rayDir *mat.VecDense, interval Interval) (float64, bool) {
	ox := raySt.AtVec(0) - s.center.AtVec(0)
	oy := raySt.AtVec(1) - s.center.AtVec(1)
	oz := raySt.AtVec(2) - s.center.AtVec(2)
	dx, dy, dz := rayDir.AtVec(0), rayDir.AtVec(1), rayDir.AtVec(2)

	a := dx*dx + dy*dy + dz*dz
	b := 2 * (dx*ox + dy*oy + dz*oz)
	c := ox*ox + oy*oy + oz*oz - s.R*s.R
	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return 0, false
	}

	sqrtDiscriminant := math.Sqrt(discriminant)
	root1 := (-b - sqrtDiscriminant) / (2 * a)
	root2 := (-b + sqrtDiscriminant) / (2 * a)
	distance := closestDistance(root1, root2, interval.Min, interval.Max)
	if distance == math.MaxFloat64 {
		return 0, false
	}

	return distance, true
}

func (s *Sphere) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	return maths.Normalize(maths.SubVec(res, intersect, s.center))
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
