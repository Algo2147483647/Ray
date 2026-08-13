package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type Plane struct {
	BaseShape
	A *mat.VecDense `json:"A"` // f(x) = a^T * x + b
	B float64       `json:"b"`
}

func (p *Plane) Name() string {
	return "Plane"
}

func (p *Plane) IntersectAffine(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if !options.valid() {
		return SurfaceInteraction{}, false
	}
	t := mat.Dot(p.A, rayDir)
	if math.Abs(t) < utils.EPS {
		return SurfaceInteraction{}, false
	}

	d := -(mat.Dot(p.A, raySt) + p.B) / t
	if !distanceInRange(d, options.Range.Min, options.Range.Max) {
		return SurfaceInteraction{}, false
	}

	normal := p.GetNormalVector(nil, mat.NewVecDense(p.A.Len(), nil))
	return newAffineSurfaceInteraction(raySt, rayDir, d, normal), true
}

func (p *Plane) IntersectGeodesic(rayStart, rayDir *mat.VecDense, g geometry.Geometry, options IntersectOptions) (SurfaceInteraction, bool) {
	if !supportsSphericalGeodesic(g, options) {
		return SurfaceInteraction{}, false
	}
	if p == nil || p.A == nil || rayStart.Len() != rayDir.Len() || p.A.Len() != rayStart.Len() {
		return SurfaceInteraction{}, false
	}
	sMin, sMax := options.Range.Min, options.Range.Max
	return intersectSphericalScalar(rayStart, rayDir, sMin, sMax, func(point *mat.VecDense) float64 {
		return mat.Dot(p.A, point) + p.B
	}, func(point *mat.VecDense) *mat.VecDense {
		return p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	})
}

func (p *Plane) GetNormalVector(_, res *mat.VecDense) *mat.VecDense {
	res.CloneFromVec(p.A)
	return maths.Normalize(res)
}
