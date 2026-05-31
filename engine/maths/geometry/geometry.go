// Package geometry defines the abstract space in which rays propagate.
//
// Rays in this engine always carry embedded coordinates: Klein in the unit
// ball of R^3, Spherical as unit vectors in R^4, Euclidean as plain R^3.
// Klein geodesics are Euclidean chords in the embedded ball, so ordinary
// Shape intersection can be reused with a model-boundary tMax. Spherical
// geodesics are great circles, so surface intersection must evaluate the
// geodesic itself instead of an affine ambient ray.
package geometry

import (
	"gonum.org/v1/gonum/mat"
)

// Geometry describes the metric model rays propagate in.
type Geometry interface {
	// Name is a stable identifier ("euclidean", "klein", "spherical").
	Name() string

	// Dimension is the embedding dimension: Euclidean=3, Klein=3, Spherical=4.
	Dimension() int

	// ProjectTangent projects v into the tangent space T_p M, writing into out.
	// For Euclidean/Klein this is identity; for Spherical it subtracts the
	// radial component. out may alias v.
	ProjectTangent(p, v, out *mat.VecDense) *mat.VecDense

	// InnerProduct returns <u, v>_p under the metric of M at p.
	InnerProduct(p, u, v *mat.VecDense) float64

	// ArcLengthFromEmbedT translates the Euclidean ray parameter t (as
	// returned by Shape.Intersect on the embedded ray (p, dir)) into the
	// geodesic arc length traveled in M. Implementations must clamp pathological
	// inputs (NaN, Inf, negative) to a safe finite value.
	ArcLengthFromEmbedT(p, dir *mat.VecDense, tEuclid float64) float64

	// Exp evaluates gamma(t) = Exp_p(t*v), writing into out. out may alias p.
	Exp(p, v *mat.VecDense, t float64, out *mat.VecDense) *mat.VecDense

	// EmbeddedRay returns the (origin, direction) to hand to BVH/Shape
	// intersection, plus the natural maximum embedded t after which the ray
	// leaves the model. For Euclidean tMaxEmbed is +Inf; for Klein it is the
	// unit-ball boundary. Spherical has no affine embedded-ray representation
	// for great-circle tracing and may return tMaxEmbed=0.
	EmbeddedRay(p, dir *mat.VecDense) (eo, ed *mat.VecDense, tMaxEmbed float64)

	// WrapBeyond is used by S^3: advance the ray by arcAdvance along its
	// geodesic and parallel-transport the direction. Returns ok=false for
	// other geometries.
	WrapBeyond(p, dir *mat.VecDense, arcAdvance float64) (newP, newD *mat.VecDense, ok bool)
}

// Get returns the geometry, falling back to Euclidean if g is nil.
// This lets call sites be nil-safe without sprinkling checks.
func Get(g Geometry) Geometry {
	if g == nil {
		return Euclidean()
	}
	return g
}
