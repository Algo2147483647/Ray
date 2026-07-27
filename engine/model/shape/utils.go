package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
)

type SurfaceInteraction struct {
	Distance        float64
	ArcLength       float64
	Point           *mat.VecDense
	GeometricNormal *mat.VecDense
	ShadingNormal   *mat.VecDense
	UV              [2]float64
	DPDU            *mat.VecDense
	DPDV            *mat.VecDense
	PrimitiveID     int
}

// Interval is the closed parameter range accepted by an intersection query.
type Interval struct {
	Min float64
	Max float64
}

func (i Interval) Valid() bool {
	return !math.IsNaN(i.Min) && !math.IsNaN(i.Max) && i.Min <= i.Max
}

// IntersectionPath describes how rayStart and rayDir parameterize the path.
type IntersectionPath uint8

const (
	// PathAffine evaluates rayStart + t*rayDir.
	PathAffine IntersectionPath = iota
	// PathGreatCircle evaluates the unit-sphere geodesic starting at rayStart
	// with rayDir as its initial tangent.
	PathGreatCircle
)

// IntersectOptions contains all non-ray inputs shared by shape intersections.
// Its zero Path value is affine; callers should supply Range explicitly.
type IntersectOptions struct {
	Range Interval
	Path  IntersectionPath
}

func NewIntersectOptions(tMin, tMax float64) IntersectOptions {
	return IntersectOptions{Range: Interval{Min: tMin, Max: tMax}}
}

func NewGreatCircleIntersectOptions(sMin, sMax float64) IntersectOptions {
	return IntersectOptions{
		Range: Interval{Min: sMin, Max: sMax},
		Path:  PathGreatCircle,
	}
}

func (o IntersectOptions) validFor(path IntersectionPath) bool {
	return o.Path == path && o.Range.Valid()
}

func distanceInRange(distance, tMin, tMax float64) bool {
	return distance >= tMin && distance <= tMax && !math.IsNaN(distance) && !math.IsInf(distance, 0)
}

func closestDistance(root1, root2, tMin, tMax float64) float64 {
	switch {
	case distanceInRange(root1, tMin, tMax) && distanceInRange(root2, tMin, tMax):
		return math.Min(root1, root2)
	case distanceInRange(root1, tMin, tMax):
		return root1
	case distanceInRange(root2, tMin, tMax):
		return root2
	default:
		return math.MaxFloat64
	}
}

func pointAt(rayStart, rayDir *mat.VecDense, distance float64) *mat.VecDense {
	point := mat.NewVecDense(rayStart.Len(), nil)
	point.AddScaledVec(rayStart, distance, rayDir)
	return point
}

func vecDenseXYZ(v *mat.VecDense) [3]float64 {
	return [3]float64{v.AtVec(0), v.AtVec(1), v.AtVec(2)}
}

func newSurfaceInteraction(rayStart, rayDir *mat.VecDense, distance float64, normal *mat.VecDense) SurfaceInteraction {
	point := pointAt(rayStart, rayDir, distance)
	return newSurfaceInteractionAt(point, distance, normal)
}

func newSurfaceInteractionAt(point *mat.VecDense, distance float64, normal *mat.VecDense) SurfaceInteraction {
	return SurfaceInteraction{
		Distance:        distance,
		ArcLength:       0,
		Point:           point,
		GeometricNormal: normal,
		ShadingNormal:   normal,
		PrimitiveID:     -1,
	}
}
