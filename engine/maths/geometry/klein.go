package geometry

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// klein implements H^3 via the Beltrami-Klein model: the open unit ball in
// R^3 with hyperbolic metric. Geodesics are Euclidean chords, so all BVH
// and Shape intersection code is reused; only distances and tangent-space
// inner products are translated through this type.
//
// Reference: Cannon, Floyd, Kenyon, Parry, "Hyperbolic Geometry",
// MSRI 1997, Chapter 7 (Klein and Hyperboloid models).
type klein struct{}

var kleinSingleton Geometry = klein{}

// Klein returns the Beltrami-Klein H^3 singleton.
func Klein() Geometry { return kleinSingleton }

func (klein) Name() string   { return "klein" }
func (klein) Dimension() int { return 3 }

func (klein) ProjectTangent(_, v, out *mat.VecDense) *mat.VecDense {
	if out != v {
		out.CopyVec(v)
	}
	return out
}

// InnerProduct under the Beltrami-Klein metric at p:
//
//	g_p(u, v) = ( u·v )/(1-|p|²) + ( p·u )( p·v )/(1-|p|²)²
//
// Falls back to Euclidean if |p|² >= 1 (caller misuse / boundary).
func (klein) InnerProduct(p, u, v *mat.VecDense) float64 {
	pp := mat.Dot(p, p)
	if pp >= 1 {
		return mat.Dot(u, v)
	}
	w := 1 - pp
	uv := mat.Dot(u, v)
	pu := mat.Dot(p, u)
	pv := mat.Dot(p, v)
	return uv/w + (pu*pv)/(w*w)
}

// hyperbolicDistance returns d_H(p, q) using the closed form
//
//	cosh d_H = (1 - p·q) / sqrt((1-|p|²)(1-|q|²))
//
// clamped to be safe at the boundary.
func hyperbolicDistance(p, q *mat.VecDense) float64 {
	pp := mat.Dot(p, p)
	qq := mat.Dot(q, q)
	if pp >= 1 || qq >= 1 {
		return maxArcClamp
	}
	num := 1 - mat.Dot(p, q)
	den := math.Sqrt((1 - pp) * (1 - qq))
	if den <= 0 {
		return maxArcClamp
	}
	c := num / den
	if c < 1 {
		c = 1
	}
	d := math.Acosh(c)
	if math.IsNaN(d) || math.IsInf(d, 0) {
		return maxArcClamp
	}
	if d > maxArcClamp {
		return maxArcClamp
	}
	return d
}

// maxArcClamp is the safe upper bound used in place of +Inf. Chosen so that
// exp(-sigma_a * maxArcClamp) underflows to zero for any plausible sigma_a.
const maxArcClamp = 1e6

func (klein) ArcLengthFromEmbedT(p, dir *mat.VecDense, tEuclid float64) float64 {
	if math.IsNaN(tEuclid) || tEuclid < 0 {
		return 0
	}
	if math.IsInf(tEuclid, 0) {
		return maxArcClamp
	}
	q := mat.NewVecDense(p.Len(), nil)
	q.CopyVec(p)
	q.AddScaledVec(q, tEuclid, dir)
	return hyperbolicDistance(p, q)
}

// Exp_p(v, s): walk along the Klein chord through p with direction v for
// hyperbolic arc length s. We find the embedded t on the chord by inverting
// hyperbolicDistance via a monotone bisection. v must point inside the ball
// (we use its Euclidean direction; metric scaling is implicit via the
// distance function).
func (klein) Exp(p, v *mat.VecDense, s float64, out *mat.VecDense) *mat.VecDense {
	if s == 0 {
		out.CopyVec(p)
		return out
	}
	dir := mat.NewVecDense(v.Len(), nil)
	dir.CopyVec(v)
	if n := mat.Norm(dir, 2); n > 0 {
		dir.ScaleVec(1/n, dir)
	}
	// Find t in [0, tBoundary] such that hyperbolicDistance(p, p+t*dir) = s.
	_, _, tMax := klein{}.EmbeddedRay(p, dir)
	if tMax <= 0 {
		out.CopyVec(p)
		return out
	}
	q := mat.NewVecDense(p.Len(), nil)
	dist := func(t float64) float64 {
		q.CopyVec(p)
		q.AddScaledVec(q, t, dir)
		return hyperbolicDistance(p, q)
	}
	lo, hi := 0.0, math.Min(tMax*(1-1e-12), 1e9)
	if dist(hi) < s {
		out.CopyVec(p)
		out.AddScaledVec(out, hi, dir)
		return out
	}
	for i := 0; i < 80; i++ {
		mid := 0.5 * (lo + hi)
		if dist(mid) < s {
			lo = mid
		} else {
			hi = mid
		}
		if hi-lo < 1e-12 {
			break
		}
	}
	t := 0.5 * (lo + hi)
	out.CopyVec(p)
	out.AddScaledVec(out, t, dir)
	return out
}

// EmbeddedRay: the chord (p, dir) leaves the unit ball at the larger root
// of |p + t*dir|² = 1.
func (klein) EmbeddedRay(p, dir *mat.VecDense) (*mat.VecDense, *mat.VecDense, float64) {
	a := mat.Dot(dir, dir)
	b := 2 * mat.Dot(p, dir)
	c := mat.Dot(p, p) - 1
	disc := b*b - 4*a*c
	if a == 0 || disc < 0 {
		return p, dir, 0
	}
	t := (-b + math.Sqrt(disc)) / (2 * a)
	if t < 0 {
		return p, dir, 0
	}
	return p, dir, t
}

func (klein) WrapBeyond(_, _ *mat.VecDense, _ float64) (*mat.VecDense, *mat.VecDense, bool) {
	return nil, nil, false
}
