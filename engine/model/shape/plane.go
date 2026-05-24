package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
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

func (p *Plane) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := p.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (p *Plane) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	t := mat.Dot(p.A, rayDir)
	if math.Abs(t) < utils.EPS {
		return SurfaceInteraction{}, false
	}

	d := -(mat.Dot(p.A, raySt) + p.B) / t
	if !distanceInRange(d, tMin, tMax) {
		return SurfaceInteraction{}, false
	}

	normal := p.GetNormalVector(nil, mat.NewVecDense(p.A.Len(), nil))
	return newSurfaceInteraction(raySt, rayDir, d, normal), true
}

func (p *Plane) GetNormalVector(_, res *mat.VecDense) *mat.VecDense {
	res.CloneFromVec(p.A)
	return maths.Normalize(res)
}
