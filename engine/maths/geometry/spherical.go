package geometry

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// spherical implements S^3, the unit 3-sphere embedded in R^4. Points are
// unit vectors in R^4; tangent vectors at p satisfy <v, p> = 0. Geodesics
// are great circles. As in the Klein model, a ray (p, d) hits an embedded
// object exactly where the Euclidean line p + t*d enters the object —
// because BVH and Shape intersection are computed in R^4 directly. We only
// translate the parameter t into arc length acos.
//
// Reference: standard differential geometry; see Lee, "Riemannian Manifolds",
// chapter on space forms.
type spherical struct{}

var sphericalSingleton Geometry = spherical{}

// Spherical returns the S^3 singleton.
func Spherical() Geometry { return sphericalSingleton }

func (spherical) Name() string   { return "spherical" }
func (spherical) Dimension() int { return 4 }

// ProjectTangent: v - <v, p> p.
func (spherical) ProjectTangent(p, v, out *mat.VecDense) *mat.VecDense {
	dot := mat.Dot(v, p)
	if out != v {
		out.CopyVec(v)
	}
	out.AddScaledVec(out, -dot, p)
	return out
}

// InnerProduct: ambient Euclidean dot — S^n inherits the round metric from R^{n+1}.
func (spherical) InnerProduct(_, u, v *mat.VecDense) float64 {
	return mat.Dot(u, v)
}

// ArcLengthFromEmbedT: q = p + t*d (necessarily on S^3 if d is a chord direction
// of the great circle through p), arc = acos(<p, q>) with q renormalized.
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

// Exp_p(v, s) = cos(s) p + sin(s) v̂ where v̂ is the unit tangent in the
// embedded inner product. v is assumed already in T_p (caller must Project).
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

// EmbeddedRay: the BVH sees the chord (p, dir). The natural extent is the
// chord across the half-great-circle (arc = π), corresponding to embedded
// t = 2 <p, -dir>/<dir, dir> if dir is a unit tangent... but for generality
// we pass a comfortably large t and let ArcLengthFromEmbedT clamp at π.
// To keep callers simple, we return tMaxEmbed = 2 (since the chord from p
// to its antipode through any tangent direction has Euclidean length 2 when
// dir is unit; longer dir scales linearly).
func (spherical) EmbeddedRay(p, dir *mat.VecDense) (*mat.VecDense, *mat.VecDense, float64) {
	n := mat.Norm(dir, 2)
	if n == 0 {
		return p, dir, 0
	}
	return p, dir, 2 / n
}

// WrapBeyond: advance the great circle by arcAdvance. New position is the
// standard Exp; new direction is the parallel transport of dir along the
// great circle from p to newP, which in S^n has the closed form:
//
//	newD = -sin(s)/|v| p + cos(s) v̂      (where v = dir, projected & unit)
//
// (See Boumal, "An Introduction to Optimization on Smooth Manifolds",
// Example 7.5 — parallel transport on the sphere.)
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
	// Parallel transport: tangent at newP pointing along the same great circle.
	newD := mat.NewVecDense(p.Len(), nil)
	newD.CopyVec(p)
	newD.ScaleVec(-sn*vn, newD)
	newD.AddScaledVec(newD, cs, v)
	return newP, newD, true
}
