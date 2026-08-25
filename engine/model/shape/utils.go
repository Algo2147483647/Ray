package shape

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

const (
	defaultSphericalRootSteps = 2048
	sphericalRootTol          = 1e-8
	sphericalValueTol         = 1e-7
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

func newAffineSurfaceInteraction(rayStart, rayDir *mat.VecDense, distance float64, normal *mat.VecDense) SurfaceInteraction {
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

func intersectSphericalScalar(
	rayStart, rayDir *mat.VecDense,
	sMin, sMax float64,
	evaluate func(*mat.VecDense) float64,
	normalAt func(*mat.VecDense) *mat.VecDense,
) (SurfaceInteraction, bool) {
	v, ok := sphericalUnitTangent(rayStart, rayDir)
	if !ok || sMax < sMin {
		return SurfaceInteraction{}, false
	}

	best, ok := findFirstSphericalRoot(rayStart, v, sMin, sMax, evaluate)
	if !ok {
		return SurfaceInteraction{}, false
	}

	point := sphericalPointAtUnit(rayStart, v, best)
	normal := normalAt(point)
	return SurfaceInteraction{
		Distance:        best,
		ArcLength:       best,
		Point:           point,
		GeometricNormal: normal,
		ShadingNormal:   normal,
		PrimitiveID:     -1,
	}, true
}

func supportsSphericalGeodesic(g geometry.Geometry, options IntersectOptions) bool {
	return g != nil && g.Kind() == geometry.SphericalKind && options.valid()
}

func findFirstSphericalRoot(
	rayStart, unitTangent *mat.VecDense,
	sMin, sMax float64,
	evaluate func(*mat.VecDense) float64,
) (float64, bool) {
	steps := defaultSphericalRootSteps
	if sMax-sMin < 1e-6 {
		steps = 1
	}

	prevS := sMin
	prevValue := evaluate(sphericalPointAtUnit(rayStart, unitTangent, prevS))
	if maths.IsFinite(prevValue) && math.Abs(prevValue) <= sphericalValueTol {
		return prevS, true
	}

	for i := 1; i <= steps; i++ {
		currS := sMin + (sMax-sMin)*float64(i)/float64(steps)
		currValue := evaluate(sphericalPointAtUnit(rayStart, unitTangent, currS))
		if !maths.IsFinite(currValue) {
			prevS, prevValue = currS, currValue
			continue
		}
		if math.Abs(currValue) <= sphericalValueTol {
			return currS, true
		}
		if maths.IsFinite(prevValue) && maths.SignChanged(prevValue, currValue) {
			return refineSphericalRoot(rayStart, unitTangent, prevS, currS, prevValue, evaluate), true
		}
		prevS, prevValue = currS, currValue
	}

	return 0, false
}

func refineSphericalRoot(
	rayStart, unitTangent *mat.VecDense,
	left, right, fLeft float64,
	evaluate func(*mat.VecDense) float64,
) float64 {
	for i := 0; i < 80; i++ {
		mid := 0.5 * (left + right)
		fMid := evaluate(sphericalPointAtUnit(rayStart, unitTangent, mid))
		if !maths.IsFinite(fMid) || math.Abs(fMid) <= sphericalValueTol || math.Abs(right-left) <= sphericalRootTol {
			return mid
		}
		if maths.SignChanged(fLeft, fMid) {
			right = mid
		} else {
			left = mid
			fLeft = fMid
		}
	}
	return 0.5 * (left + right)
}

func sphericalUnitTangent(rayStart, rayDir *mat.VecDense) (*mat.VecDense, bool) {
	if rayStart == nil || rayDir == nil || rayStart.Len() != rayDir.Len() {
		return nil, false
	}
	v := mat.NewVecDense(rayDir.Len(), nil)
	v.CopyVec(rayDir)
	v.AddScaledVec(v, -mat.Dot(v, rayStart), rayStart)
	n := mat.Norm(v, 2)
	if n <= utils.EPS {
		return nil, false
	}
	v.ScaleVec(1/n, v)
	return v, true
}

func sphericalPointAtUnit(rayStart, unitTangent *mat.VecDense, s float64) *mat.VecDense {
	point := mat.NewVecDense(rayStart.Len(), nil)
	point.CopyVec(rayStart)
	point.ScaleVec(math.Cos(s), point)
	point.AddScaledVec(point, math.Sin(s), unitTangent)
	return point
}

func solveSphericalLinearCoordinate(a, b, c, sMin, sMax float64) []float64 {
	r := math.Hypot(a, b)
	if r <= utils.EPS || math.Abs(c) > r+utils.EPS {
		return nil
	}
	value := c / r
	if value > 1 {
		value = 1
	} else if value < -1 {
		value = -1
	}

	phase := math.Atan2(b, a)
	base := math.Acos(value)
	candidates := []float64{phase + base, phase - base}
	var result []float64
	for _, s := range candidates {
		for s < sMin {
			s += 2 * math.Pi
		}
		for s > sMax {
			s -= 2 * math.Pi
		}
		if distanceInRange(s, sMin, sMax) {
			result = append(result, s)
		}
	}
	return result
}

func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v >= 1 {
		return math.Nextafter(1, 0)
	}
	return v
}
