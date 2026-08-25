package geometry

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

type euclidean struct {
	dimension int
}

// Euclidean returns a K=0 geometry with the requested embedding dimension.
func Euclidean(dimension int) Geometry {
	if dimension <= 0 {
		panic("Euclidean dimension must be positive")
	}
	return euclidean{dimension: dimension}
}

func (euclidean) Name() string     { return "euclidean" }
func (euclidean) Kind() Kind       { return EuclideanKind }
func (e euclidean) Dimension() int { return e.dimension }

func (euclidean) ProjectTangent(_, v, out *mat.VecDense) *mat.VecDense {
	if out != v {
		out.CopyVec(v)
	}
	return out
}

func (euclidean) InnerProduct(_, u, v *mat.VecDense) float64 {
	return mat.Dot(u, v)
}

func (euclidean) IntrinsicNormal(_, ambientGradient, out *mat.VecDense) *mat.VecDense {
	if out != ambientGradient {
		out.CopyVec(ambientGradient)
	}
	return out
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

func (euclidean) GeodesicDirection(_, v *mat.VecDense, _ float64, out *mat.VecDense) *mat.VecDense {
	out.CopyVec(v)
	return out
}

func (euclidean) EmbeddedRay(p, dir *mat.VecDense) (*mat.VecDense, *mat.VecDense, float64) {
	return p, dir, math.Inf(+1)
}

func (euclidean) WrapBeyond(_, _ *mat.VecDense, _ float64) (*mat.VecDense, *mat.VecDense, bool) {
	return nil, nil, false
}
