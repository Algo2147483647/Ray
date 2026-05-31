package geometry

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

type euclidean struct{}

var euclideanSingleton Geometry = euclidean{}

// Euclidean returns the K=0 geometry singleton.
func Euclidean() Geometry { return euclideanSingleton }

func (euclidean) Name() string   { return "euclidean" }
func (euclidean) Dimension() int { return 3 }

func (euclidean) ProjectTangent(_, v, out *mat.VecDense) *mat.VecDense {
	if out != v {
		out.CopyVec(v)
	}
	return out
}

func (euclidean) InnerProduct(_, u, v *mat.VecDense) float64 {
	return mat.Dot(u, v)
}

func (euclidean) ArcLengthFromEmbedT(_, dir *mat.VecDense, tEuclid float64) float64 {
	if math.IsNaN(tEuclid) || math.IsInf(tEuclid, 0) || tEuclid < 0 {
		return 0
	}
	return tEuclid * mat.Norm(dir, 2)
}

func (euclidean) Exp(p, v *mat.VecDense, t float64, out *mat.VecDense) *mat.VecDense {
	out.CopyVec(p)
	out.AddScaledVec(out, t, v)
	return out
}

func (euclidean) EmbeddedRay(p, dir *mat.VecDense) (*mat.VecDense, *mat.VecDense, float64) {
	return p, dir, math.Inf(+1)
}

func (euclidean) WrapBeyond(_, _ *mat.VecDense, _ float64) (*mat.VecDense, *mat.VecDense, bool) {
	return nil, nil, false
}
