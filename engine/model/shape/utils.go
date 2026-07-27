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

// IntersectOptions contains parameter-domain constraints shared by Euclidean
// and geodesic intersection queries. Geometry selection is deliberately kept
// out of this value and handled by the caller.
type IntersectOptions struct {
	Range Interval
}

func NewIntersectOptions(tMin, tMax float64) IntersectOptions {
	return IntersectOptions{Range: Interval{Min: tMin, Max: tMax}}
}

func (o IntersectOptions) valid() bool {
	return o.Range.Valid()
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

func affinePointAt(rayStart, rayDir *mat.VecDense, distance float64) *mat.VecDense {
	point := mat.NewVecDense(rayStart.Len(), nil)
	point.AddScaledVec(rayStart, distance, rayDir)
	return point
}

func vecDenseXYZ(v *mat.VecDense) [3]float64 {
	return [3]float64{v.AtVec(0), v.AtVec(1), v.AtVec(2)}
}

func newSurfaceInteraction(rayStart, rayDir *mat.VecDense, distance float64, normal *mat.VecDense) SurfaceInteraction {
	point := affinePointAt(rayStart, rayDir, distance)
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
