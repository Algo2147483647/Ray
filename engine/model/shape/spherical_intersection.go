package shape

import (
	"math"

	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

const (
	defaultSphericalRootSteps = 2048
	sphericalRootTol          = 1e-8
	sphericalValueTol         = 1e-7
)

// SphericalSurfaceCandidateProvider intersects a shape with an S^3 geodesic.
// Distance and ArcLength are both the traveled spherical arc in radians.
type SphericalSurfaceCandidateProvider interface {
	IntersectSphericalCandidate(rayStart, rayDir *mat.VecDense, sMin, sMax float64) (SurfaceCandidate, bool)
}

func (b *BoundedShape) IntersectSphericalCandidate(rayStart, rayDir *mat.VecDense, sMin, sMax float64) (SurfaceCandidate, bool) {
	if b == nil || b.Shape == nil {
		return SurfaceCandidate{}, false
	}
	provider, ok := b.Shape.(SphericalSurfaceCandidateProvider)
	if !ok {
		return SurfaceCandidate{}, false
	}
	candidate, ok := provider.IntersectSphericalCandidate(rayStart, rayDir, sMin, sMax)
	if !ok || b.Bounds == nil || candidate.Point == nil {
		return candidate, ok
	}
	if !b.Bounds.containsPoint(candidate.Point, -1) {
		return SurfaceCandidate{}, false
	}
	return candidate, true
}

func (s *Sphere) IntersectSphericalCandidate(rayStart, rayDir *mat.VecDense, sMin, sMax float64) (SurfaceCandidate, bool) {
	return sphericalScalarCandidate(rayStart, rayDir, sMin, sMax, func(point *mat.VecDense) float64 {
		offset := mat.NewVecDense(point.Len(), nil)
		offset.SubVec(point, s.center)
		return mat.Dot(offset, offset) - s.R*s.R
	}, func(point *mat.VecDense) *mat.VecDense {
		return s.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	})
}

func (p *Plane) IntersectSphericalCandidate(rayStart, rayDir *mat.VecDense, sMin, sMax float64) (SurfaceCandidate, bool) {
	if p == nil || p.A == nil || rayStart.Len() != rayDir.Len() || p.A.Len() != rayStart.Len() {
		return SurfaceCandidate{}, false
	}
	return sphericalScalarCandidate(rayStart, rayDir, sMin, sMax, func(point *mat.VecDense) float64 {
		return mat.Dot(p.A, point) + p.B
	}, func(point *mat.VecDense) *mat.VecDense {
		return p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	})
}

func (c *Circle) IntersectSphericalCandidate(rayStart, rayDir *mat.VecDense, sMin, sMax float64) (SurfaceCandidate, bool) {
	if c == nil || c.Center == nil || c.Normal == nil || rayStart.Len() != rayDir.Len() {
		return SurfaceCandidate{}, false
	}
	v, ok := sphericalUnitTangent(rayStart, rayDir)
	if !ok {
		return SurfaceCandidate{}, false
	}

	a := mat.Dot(c.Normal, rayStart)
	b := mat.Dot(c.Normal, v)
	target := mat.Dot(c.Normal, c.Center)
	best := math.Inf(1)
	for _, s := range solveSphericalLinearCoordinate(a, b, target, sMin, sMax) {
		point := sphericalPointAtUnit(rayStart, v, s)
		offset := mat.NewVecDense(point.Len(), nil)
		offset.SubVec(point, c.Center)
		if mat.Dot(offset, offset) <= c.R*c.R+utils.EPS && s < best {
			best = s
		}
	}
	if math.IsInf(best, 1) {
		return SurfaceCandidate{}, false
	}

	point := sphericalPointAtUnit(rayStart, v, best)
	normal := c.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return SurfaceCandidate{
		Distance:        best,
		ArcLength:       best,
		Point:           point,
		GeometricNormal: normal,
		ShadingNormal:   normal,
		PrimitiveID:     -1,
	}, true
}

func (q *QuadraticEquation) IntersectSphericalCandidate(rayStart, rayDir *mat.VecDense, sMin, sMax float64) (SurfaceCandidate, bool) {
	n := rayStart.Len()
	if q == nil || q.A == nil || q.B == nil || rayDir.Len() != n || q.B.Len() != n {
		return SurfaceCandidate{}, false
	}
	ar, ac := q.A.Dims()
	if ar != n || ac != n {
		return SurfaceCandidate{}, false
	}
	return sphericalScalarCandidate(rayStart, rayDir, sMin, sMax, func(point *mat.VecDense) float64 {
		ap := mat.NewVecDense(n, nil)
		ap.MulVec(q.A, point)
		return mat.Dot(point, ap) + mat.Dot(q.B, point) + q.C
	}, func(point *mat.VecDense) *mat.VecDense {
		return q.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	})
}

func (p *PolynomialSurface) IntersectSphericalCandidate(rayStart, rayDir *mat.VecDense, sMin, sMax float64) (SurfaceCandidate, bool) {
	if p == nil || p.Coefficients == nil || rayStart.Len() != rayDir.Len() || p.InputDim > rayStart.Len() {
		return SurfaceCandidate{}, false
	}
	candidate, ok := sphericalScalarCandidate(rayStart, rayDir, sMin, sMax, func(point *mat.VecDense) float64 {
		local := p.localPoint(point)
		return p.Evaluate(local[:p.InputDim])
	}, func(point *mat.VecDense) *mat.VecDense {
		return p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	})
	return candidate, ok
}

func (f *ImplicitEquation) IntersectSphericalCandidate(rayStart, rayDir *mat.VecDense, sMin, sMax float64) (SurfaceCandidate, bool) {
	if f == nil || f.Function == nil || rayStart.Len() != rayDir.Len() {
		return SurfaceCandidate{}, false
	}
	candidate, ok := sphericalScalarCandidate(rayStart, rayDir, sMin, sMax, f.Function, func(point *mat.VecDense) *mat.VecDense {
		return f.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	})
	if !ok || !f.hasValidRange() || candidate.Point == nil {
		return candidate, ok
	}
	bounds := NewCuboid(f.Range[0], f.Range[1])
	if !bounds.containsPoint(candidate.Point, -1) {
		return SurfaceCandidate{}, false
	}
	return candidate, true
}

func (c *Cuboid) IntersectSphericalCandidate(rayStart, rayDir *mat.VecDense, sMin, sMax float64) (SurfaceCandidate, bool) {
	if c == nil || c.Pmin == nil || c.Pmax == nil || rayStart.Len() != rayDir.Len() {
		return SurfaceCandidate{}, false
	}

	v, ok := sphericalUnitTangent(rayStart, rayDir)
	if !ok {
		return SurfaceCandidate{}, false
	}

	best := math.Inf(1)
	for axis := 0; axis < rayStart.Len(); axis++ {
		for _, bound := range []float64{c.Pmin.AtVec(axis), c.Pmax.AtVec(axis)} {
			for _, s := range solveSphericalLinearCoordinate(rayStart.AtVec(axis), v.AtVec(axis), bound, sMin, sMax) {
				if s >= best {
					continue
				}
				point := sphericalPointAtUnit(rayStart, v, s)
				if c.containsPoint(point, axis) {
					best = s
				}
			}
		}
	}

	if math.IsInf(best, 1) {
		return SurfaceCandidate{}, false
	}
	point := sphericalPointAtUnit(rayStart, v, best)
	normal := c.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return SurfaceCandidate{
		Distance:        best,
		ArcLength:       best,
		Point:           point,
		GeometricNormal: normal,
		ShadingNormal:   normal,
		PrimitiveID:     -1,
	}, true
}

func sphericalScalarCandidate(
	rayStart, rayDir *mat.VecDense,
	sMin, sMax float64,
	evaluate func(*mat.VecDense) float64,
	normalAt func(*mat.VecDense) *mat.VecDense,
) (SurfaceCandidate, bool) {
	v, ok := sphericalUnitTangent(rayStart, rayDir)
	if !ok || sMax < sMin {
		return SurfaceCandidate{}, false
	}

	best, ok := findFirstSphericalRoot(rayStart, v, sMin, sMax, evaluate)
	if !ok {
		return SurfaceCandidate{}, false
	}

	point := sphericalPointAtUnit(rayStart, v, best)
	normal := normalAt(point)
	return SurfaceCandidate{
		Distance:        best,
		ArcLength:       best,
		Point:           point,
		GeometricNormal: normal,
		ShadingNormal:   normal,
		PrimitiveID:     -1,
	}, true
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
	if isFinite(prevValue) && math.Abs(prevValue) <= sphericalValueTol {
		return prevS, true
	}

	for i := 1; i <= steps; i++ {
		currS := sMin + (sMax-sMin)*float64(i)/float64(steps)
		currValue := evaluate(sphericalPointAtUnit(rayStart, unitTangent, currS))
		if !isFinite(currValue) {
			prevS, prevValue = currS, currValue
			continue
		}
		if math.Abs(currValue) <= sphericalValueTol {
			return currS, true
		}
		if isFinite(prevValue) && hasSignChange(prevValue, currValue) {
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
		if !isFinite(fMid) || math.Abs(fMid) <= sphericalValueTol || math.Abs(right-left) <= sphericalRootTol {
			return mid
		}
		if hasSignChange(fLeft, fMid) {
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

func (c *Cuboid) containsPoint(point *mat.VecDense, hitAxis int) bool {
	for axis := 0; axis < point.Len(); axis++ {
		if axis == hitAxis {
			continue
		}
		x := point.AtVec(axis)
		if x < c.Pmin.AtVec(axis)-utils.EPS || x > c.Pmax.AtVec(axis)+utils.EPS {
			return false
		}
	}
	return true
}
