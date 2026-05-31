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

type SurfaceCandidate struct {
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

type SurfaceCandidateProvider interface {
	IntersectCandidate(rayStart, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceCandidate, bool)
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

func SurfaceInteractionFromCandidate(rayStart, rayDir *mat.VecDense, candidate SurfaceCandidate) SurfaceInteraction {
	point := candidate.Point
	if point == nil {
		point = pointAt(rayStart, rayDir, candidate.Distance)
	}
	return SurfaceInteraction{
		Distance:        candidate.Distance,
		ArcLength:       candidate.ArcLength,
		Point:           point,
		GeometricNormal: candidate.GeometricNormal,
		ShadingNormal:   candidate.ShadingNormal,
		UV:              candidate.UV,
		DPDU:            candidate.DPDU,
		DPDV:            candidate.DPDV,
		PrimitiveID:     candidate.PrimitiveID,
	}
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
