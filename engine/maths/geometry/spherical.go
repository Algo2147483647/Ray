package geometry

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// spherical implements S^3, the unit 3-sphere embedded in R^4. Points are
// unit vectors in R^4; tangent vectors at p satisfy <v, p> = 0. Geodesics
// are great circles gamma(s)=cos(s)p+sin(s)v, not affine ambient rays.
// Surface intersection for S^3 must evaluate the great circle directly and
// keep hit points on the unit sphere.
//
// Reference: standard differential geometry; see Lee, "Riemannian Manifolds",
// chapter on space forms.
type spherical struct{}

var sphericalSingleton Geometry = spherical{}

// Spherical returns the S^3 singleton.
func Spherical() Geometry { return sphericalSingleton }

func (spherical) Name() string   { return "spherical" }
func (spherical) Dimension() int { return 4 }

// ProjectTangent computes v - <v, p> p.
func (spherical) ProjectTangent(p, v, out *mat.VecDense) *mat.VecDense {
	dot := mat.Dot(v, p)
	if out != v {
		out.CopyVec(v)
	}
	out.AddScaledVec(out, -dot, p)
	return out
}

// InnerProduct is the ambient Euclidean dot product inherited by S^n.
func (spherical) InnerProduct(_, u, v *mat.VecDense) float64 {
	return mat.Dot(u, v)
}

// ArcLengthFromEmbedT is retained for compatibility with callers that still
// pass a tangent-line parameter. It projects p+t*d back to S^3 before
// measuring the angle, so it is only valid in that local projected chart and
// is not used for S^3 surface hits.
func (spherical) ArcLengthFromEmbedT(p, dir *mat.VecDense, tEuclid float64) float64 {
	if math.IsNaN(tEuclid) || tEuclid < 0 {
		return 0
	}
	if math.IsInf(tEuclid, 0) {
		return math.Pi
	}
	q := mat.NewVecDense(p.Len(), nil)
	q.CopyVec(p)
	q.AddScaledVec(q, tEuclid, dir)
	n := mat.Norm(q, 2)
	if n == 0 {
		return 0
	}
	q.ScaleVec(1/n, q)
	c := mat.Dot(p, q)
	if c > 1 {
		c = 1
	} else if c < -1 {
		c = -1
	}
	return math.Acos(c)
}

// Exp_p(v, s) = cos(s) p + sin(s) vhat where vhat is the unit tangent in the
// embedded inner product. v is assumed already in T_p.
func (spherical) Exp(p, v *mat.VecDense, s float64, out *mat.VecDense) *mat.VecDense {
	vn := mat.Norm(v, 2)
	if vn == 0 || s == 0 {
		out.CopyVec(p)
		return out
	}
	cs, sn := math.Cos(s), math.Sin(s)
	out.CopyVec(p)
	out.ScaleVec(cs, out)
	out.AddScaledVec(out, sn/vn, v)
	return out
}

// EmbeddedRay returns no affine ray because a great circle on S^3 is not a
// Euclidean line in R^4. Spherical tracing uses a dedicated geodesic surface
// query bounded to the half-circle before the antipode.
func (spherical) EmbeddedRay(p, dir *mat.VecDense) (*mat.VecDense, *mat.VecDense, float64) {
	return p, dir, 0
}

// WrapBeyond advances the great circle by arcAdvance and parallel-transports
// the direction along that same great circle.
func (spherical) WrapBeyond(p, dir *mat.VecDense, arcAdvance float64) (*mat.VecDense, *mat.VecDense, bool) {
	v := mat.NewVecDense(p.Len(), nil)
	spherical{}.ProjectTangent(p, dir, v)
	vn := mat.Norm(v, 2)
	if vn == 0 {
		return nil, nil, false
	}
	cs, sn := math.Cos(arcAdvance), math.Sin(arcAdvance)
	newP := mat.NewVecDense(p.Len(), nil)
	newP.CopyVec(p)
	newP.ScaleVec(cs, newP)
	newP.AddScaledVec(newP, sn/vn, v)

	newD := mat.NewVecDense(p.Len(), nil)
	newD.CopyVec(p)
	newD.ScaleVec(-sn*vn, newD)
	newD.AddScaledVec(newD, cs, v)
	return newP, newD, true
}
