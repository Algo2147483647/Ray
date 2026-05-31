# Non-Euclidean Rendering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Beltrami-Klein (H³) and Spherical (S³) ray-tracing to `engine/`, behind a `Geometry` interface, with K=0 byte-equivalent to current renders. Validation: hyperbolic chessboard + S³ Hopf rings scenes.

**Architecture:** New `engine/model/geometry/` package with three implementations (`Euclidean`, `Klein`, `Spherical`). All BVH/Shape/BSDF code is reused: rays still travel along Euclidean line segments in the embedded domain (Klein ⊂ R³, S³ ⊂ R⁴). Only distance-to-arclength translation, tangent-frame construction, direction projection, and the S³ "wrap past antipode" loop are geometry-aware. Two new cameras (`HyperbolicCamera`, `SphericalCamera`). Scene JSON gains `"geometry"` and `"max_arc"` fields.

**Tech Stack:** Go 1.24+, `gonum.org/v1/gonum/mat`, standard `testing` + `math`. Existing test idioms: table-driven unit tests in `*_test.go` colocated with implementation; integration tests in `engine/controller/gallery_scene_test.go` style; render output fingerprinting where used.

**Spec:** `docs/superpowers/specs/2026-05-31-non-euclidean-rendering-design.md` (read first).

---

## File Map

**New files:**
- `engine/model/geometry/geometry.go` — `Geometry` interface, `Euclidean` singleton, `Get()` nil-safe helper
- `engine/model/geometry/klein.go` — Beltrami-Klein H³ implementation
- `engine/model/geometry/spherical.go` — S³ in R⁴ implementation
- `engine/model/geometry/geometry_test.go` — unit tests for all three
- `engine/maths/frame_geometry.go` — `NewFrameFromNormalInGeometry`
- `engine/maths/frame_geometry_test.go` — Frame-in-geometry tests
- `engine/model/camera/camera_hyperbolic.go` — Klein camera
- `engine/model/camera/camera_hyperbolic_test.go`
- `engine/model/camera/camera_spherical.go` — S³ camera
- `engine/model/camera/camera_spherical_test.go`
- `examples/scenes/hyperbolic_chessboard.json`
- `examples/scenes/spherical_hopf.json`

**Modified files:**
- `engine/model/optics/ray.go` — add `Geometry`, `ArcTraveled` fields + `G()` helper
- `engine/model/optics/ray_test.go` — assert defaults are nil-safe
- `engine/ray_tracing/trace_ray.go` — arc-length translation, Frame-in-geometry, direction reprojection, S³ wrap
- `engine/ray_tracing/trace_ray_test.go` — non-Euclidean trace tests
- `engine/model/scene.go` — add `Geometry geometry.Geometry`
- `engine/controller/parser/schema.go` — add `Geometry` and `MaxArc` to schema
- `engine/controller/factory/scene.go` — parse geometry & attach to Scene
- `engine/controller/factory/cameras.go` — dispatch `hyperbolic` / `spherical` types
- `engine/ray_tracing/handler.go` or `trace_scene.go` — propagate `Scene.Geometry` to each ray; honor `MaxArc`

---

## Conventions Used Throughout

- **Branch:** all work on `feature/non-euclidean`. Create at task 1; do not switch.
- **Test command:** `go -C engine test ./...` for full run; `go -C engine test ./model/geometry/...` for package-scoped.
- **Commit cadence:** every task ends with one commit. Messages: `feat(geometry): …`, `test(geometry): …`, `refactor(trace): …`.
- **Float tolerance:** `1e-9` for analytic identities; `1e-6` for transcendental round-trips. Use `math.Abs(a-b) <= tol`. Don't use `assert.InDelta`-style helpers — codebase uses plain `t.Errorf`.
- **Vector idiom:** `mat.NewVecDense(3, []float64{x,y,z})`. Read components via `v.AtVec(i)`. Don't introduce new helpers.

---

## Task 1: Create branch and scaffold geometry package

**Files:**
- Create: `engine/model/geometry/geometry.go`
- Create: `engine/model/geometry/geometry_test.go`

- [ ] **Step 1: Create branch**

```bash
cd /Users/didi/Ray
git checkout -b feature/non-euclidean
```

Expected: `Switched to a new branch 'feature/non-euclidean'`.

- [ ] **Step 2: Write the failing test**

Create `engine/model/geometry/geometry_test.go`:

```go
package geometry

import (
	"gonum.org/v1/gonum/mat"
	"testing"
)

func TestEuclideanIsDefault(t *testing.T) {
	g := Get(nil)
	if g == nil {
		t.Fatal("Get(nil) returned nil")
	}
	if g.Name() != "euclidean" {
		t.Errorf("default geometry name = %q, want %q", g.Name(), "euclidean")
	}
	if g.Dimension() != 3 {
		t.Errorf("default geometry dimension = %d, want 3", g.Dimension())
	}
}

func TestEuclideanArcLengthEqualsT(t *testing.T) {
	g := Euclidean()
	p := mat.NewVecDense(3, []float64{0, 0, 0})
	d := mat.NewVecDense(3, []float64{1, 0, 0})
	got := g.ArcLengthFromEmbedT(p, d, 2.5)
	if got != 2.5 {
		t.Errorf("ArcLengthFromEmbedT = %v, want 2.5", got)
	}
}

func TestEuclideanExpIsAdd(t *testing.T) {
	g := Euclidean()
	p := mat.NewVecDense(3, []float64{1, 2, 3})
	v := mat.NewVecDense(3, []float64{0, 1, 0})
	out := mat.NewVecDense(3, nil)
	got := g.Exp(p, v, 4, out)
	want := []float64{1, 6, 3}
	for i, w := range want {
		if got.AtVec(i) != w {
			t.Errorf("Exp[%d] = %v, want %v", i, got.AtVec(i), w)
		}
	}
}

func TestEuclideanWrapBeyondAlwaysFalse(t *testing.T) {
	g := Euclidean()
	p := mat.NewVecDense(3, []float64{0, 0, 0})
	d := mat.NewVecDense(3, []float64{1, 0, 0})
	_, _, ok := g.WrapBeyond(p, d, 1)
	if ok {
		t.Error("Euclidean.WrapBeyond should never return ok=true")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go -C engine test ./model/geometry/... -run . -count=1
```

Expected: build failure (`package geometry: no Go files`) or `Get undeclared`.

- [ ] **Step 4: Implement minimal `geometry.go`**

Create `engine/model/geometry/geometry.go`:

```go
// Package geometry defines the abstract space in which rays propagate.
//
// Rays in this engine always carry embedded coordinates: Klein in the unit
// ball of R^3, Spherical as unit vectors in R^4, Euclidean as plain R^3.
// Because both Klein and Spherical models keep geodesics as Euclidean line
// segments in the embedded domain, all BVH and Shape intersection code is
// reused unchanged. The Geometry interface only mediates the few places
// where distance, tangent frames, or direction reprojection care about the
// metric.
package geometry

import "gonum.org/v1/gonum/mat"

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
	// leaves the model (Klein ball boundary; S^3 half-circle limit). For
	// Euclidean tMaxEmbed is +Inf. The returned vectors may alias the inputs.
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
```

Create `engine/model/geometry/euclidean.go`:

```go
package geometry

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

type euclidean struct{}

var euclideanSingleton Geometry = euclidean{}

// Euclidean returns the K=0 geometry singleton.
func Euclidean() Geometry { return euclideanSingleton }

func (euclidean) Name() string  { return "euclidean" }
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
```

- [ ] **Step 5: Run test**

```bash
go -C engine test ./model/geometry/... -count=1
```

Expected: PASS (all four tests).

- [ ] **Step 6: Commit**

```bash
git add engine/model/geometry/
git commit -m "feat(geometry): add Geometry interface and Euclidean singleton"
```

---

## Task 2: Implement Klein (H³) geometry

**Files:**
- Create: `engine/model/geometry/klein.go`
- Modify: `engine/model/geometry/geometry_test.go`

**Math reference (do not change, copy verbatim into klein.go comment):**

The Beltrami-Klein model represents H³ as the open unit ball in R³ with metric

```
ds² = ( dx·dx )/(1-|x|²) + ( x·dx )²/(1-|x|²)²
```

Geodesics are Euclidean straight-line chords of the ball. Hyperbolic
distance between two interior points p and q is

```
d_H(p, q) = arccosh( (1 - p·q) / sqrt((1 - |p|²)(1 - |q|²)) )
```

For a ray (p, d) inside the ball, hitting an interior point q = p + t·d:

```
arc(t) = d_H(p, q)
```

For Exp_p(v): construct the chord through p with direction v (where v is in
T_p H³ via the Klein metric); parametrize so arc-length along it equals s.

The simplest implementation is to compute arc length by `d_H(p, p+t·d)` and
to compute Exp_p(v, s) by binary-searching t such that `d_H(p, p+t·v̂) = s`
where v̂ has Klein-unit length. The closed forms are well-known but easy to
get wrong; we use the analytic distance and a closed Exp via the projective
embedding (see *Cannon & Floyd, Hyperbolic Geometry*, MSRI 1997, §7).

- [ ] **Step 1: Add failing Klein tests**

Append to `engine/model/geometry/geometry_test.go`:

```go
import "math"

func TestKleinName(t *testing.T) {
	if Klein().Name() != "klein" {
		t.Errorf("Klein.Name = %q, want %q", Klein().Name(), "klein")
	}
}

func TestKleinDistanceFromOriginIsAtanhRadius(t *testing.T) {
	g := Klein()
	origin := mat.NewVecDense(3, []float64{0, 0, 0})
	// For p = (r,0,0) in the Klein model, d_H(0, p) = atanh(r).
	// Use ArcLengthFromEmbedT with dir = (1,0,0) and t = r.
	dir := mat.NewVecDense(3, []float64{1, 0, 0})
	for _, r := range []float64{0.1, 0.3, 0.7, 0.9} {
		got := g.ArcLengthFromEmbedT(origin, dir, r)
		want := math.Atanh(r)
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("ArcLength(0->%v) = %v, want atanh(%v)=%v",
				r, got, r, want)
		}
	}
}

func TestKleinExpRoundTrip(t *testing.T) {
	g := Klein()
	p := mat.NewVecDense(3, []float64{0.1, 0.2, -0.1})
	v := mat.NewVecDense(3, []float64{1, 0, 0}) // direction in embedded space
	out := mat.NewVecDense(3, nil)
	q := g.Exp(p, v, 0.5, out)
	// d_H(p, q) should equal 0.5 (up to tolerance).
	d := mat.NewVecDense(3, nil)
	d.SubVec(q, p)
	got := g.ArcLengthFromEmbedT(p, d, 1.0) // travel from p toward q by t=1 lands at q
	if math.Abs(got-0.5) > 1e-6 {
		t.Errorf("Exp round-trip arc = %v, want 0.5", got)
	}
}

func TestKleinEmbeddedRayHitsBoundary(t *testing.T) {
	g := Klein()
	p := mat.NewVecDense(3, []float64{0, 0, 0})
	d := mat.NewVecDense(3, []float64{1, 0, 0})
	_, _, tMax := g.EmbeddedRay(p, d)
	// Boundary at |x|=1 means t = 1 from origin along +x.
	if math.Abs(tMax-1) > 1e-9 {
		t.Errorf("EmbeddedRay tMax = %v, want 1", tMax)
	}
}

func TestKleinWrapBeyondFalse(t *testing.T) {
	_, _, ok := Klein().WrapBeyond(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}), 1)
	if ok {
		t.Error("Klein.WrapBeyond should return ok=false")
	}
}
```

- [ ] **Step 2: Run; expect failure**

```bash
go -C engine test ./model/geometry/... -count=1
```

Expected: `undefined: Klein`.

- [ ] **Step 3: Implement `klein.go`**

Create `engine/model/geometry/klein.go`:

```go
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

func (klein) Name() string  { return "klein" }
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
```

- [ ] **Step 4: Run; expect PASS**

```bash
go -C engine test ./model/geometry/... -count=1 -v
```

Expected: all Klein tests PASS.

- [ ] **Step 5: Commit**

```bash
git add engine/model/geometry/klein.go engine/model/geometry/geometry_test.go
git commit -m "feat(geometry): add Beltrami-Klein H^3 implementation"
```

---

## Task 3: Implement Spherical (S³) geometry

**Files:**
- Create: `engine/model/geometry/spherical.go`
- Modify: `engine/model/geometry/geometry_test.go`

- [ ] **Step 1: Add failing Spherical tests**

Append to `engine/model/geometry/geometry_test.go`:

```go
func TestSphericalName(t *testing.T) {
	if Spherical().Name() != "spherical" {
		t.Errorf("Spherical.Name = %q, want %q", Spherical().Name(), "spherical")
	}
	if Spherical().Dimension() != 4 {
		t.Errorf("Spherical.Dimension = %d, want 4", Spherical().Dimension())
	}
}

func TestSphericalProjectTangentSubtractsRadial(t *testing.T) {
	g := Spherical()
	p := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	v := mat.NewVecDense(4, []float64{0.3, 1, 0, 0})
	out := mat.NewVecDense(4, nil)
	g.ProjectTangent(p, v, out)
	// Result should be (0, 1, 0, 0) — radial component removed.
	want := []float64{0, 1, 0, 0}
	for i, w := range want {
		if math.Abs(out.AtVec(i)-w) > 1e-12 {
			t.Errorf("ProjectTangent[%d] = %v, want %v", i, out.AtVec(i), w)
		}
	}
}

func TestSphericalDistanceToAntipodeIsPi(t *testing.T) {
	g := Spherical()
	p := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	// antipode at t along direction d means p + t*d = -p with |d|=1.
	// d = -2p/2 = -p, t = 2: chord length is 2, arc length is π.
	d := mat.NewVecDense(4, []float64{-1, 0, 0, 0})
	got := g.ArcLengthFromEmbedT(p, d, 2)
	if math.Abs(got-math.Pi) > 1e-9 {
		t.Errorf("Arc to antipode = %v, want π", got)
	}
}

func TestSphericalExpQuarterTurn(t *testing.T) {
	g := Spherical()
	p := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	v := mat.NewVecDense(4, []float64{0, 1, 0, 0})
	out := mat.NewVecDense(4, nil)
	q := g.Exp(p, v, math.Pi/2, out)
	want := []float64{0, 1, 0, 0}
	for i, w := range want {
		if math.Abs(q.AtVec(i)-w) > 1e-9 {
			t.Errorf("Exp(π/2)[%d] = %v, want %v", i, q.AtVec(i), w)
		}
	}
}

func TestSphericalWrapBeyondParallelTransport(t *testing.T) {
	g := Spherical()
	p := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	d := mat.NewVecDense(4, []float64{0, 1, 0, 0}) // unit tangent at p
	// Advance by π: lands at antipode -p; transported tangent is -d.
	newP, newD, ok := g.WrapBeyond(p, d, math.Pi)
	if !ok {
		t.Fatal("Spherical.WrapBeyond returned ok=false")
	}
	expectedP := []float64{-1, 0, 0, 0}
	expectedD := []float64{0, -1, 0, 0}
	for i := 0; i < 4; i++ {
		if math.Abs(newP.AtVec(i)-expectedP[i]) > 1e-9 {
			t.Errorf("newP[%d] = %v, want %v", i, newP.AtVec(i), expectedP[i])
		}
		if math.Abs(newD.AtVec(i)-expectedD[i]) > 1e-9 {
			t.Errorf("newD[%d] = %v, want %v", i, newD.AtVec(i), expectedD[i])
		}
	}
}
```

- [ ] **Step 2: Run; expect failure**

```bash
go -C engine test ./model/geometry/... -count=1
```

Expected: `undefined: Spherical`.

- [ ] **Step 3: Implement `spherical.go`**

Create `engine/model/geometry/spherical.go`:

```go
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

func (spherical) Name() string  { return "spherical" }
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
```

- [ ] **Step 4: Run; expect PASS**

```bash
go -C engine test ./model/geometry/... -count=1 -v
```

Expected: all Spherical tests PASS.

- [ ] **Step 5: Commit**

```bash
git add engine/model/geometry/spherical.go engine/model/geometry/geometry_test.go
git commit -m "feat(geometry): add Spherical (S^3) implementation"
```

---

## Task 4: Add `Geometry` and `ArcTraveled` to Ray

**Files:**
- Modify: `engine/model/optics/ray.go:11-23`
- Modify: `engine/model/optics/ray.go:25-49` (the `Init` method)

- [ ] **Step 1: Write the failing test**

Create or append to `engine/model/optics/ray_geometry_test.go`:

```go
package optics

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/model/geometry"
)

func TestRayDefaultGeometryIsEuclidean(t *testing.T) {
	r := &Ray{}
	r.Init()
	if r.Geometry != nil {
		t.Errorf("Ray.Init left Geometry = %v, want nil (sentinel for Euclidean)", r.Geometry)
	}
	if g := r.G(); g.Name() != "euclidean" {
		t.Errorf("Ray.G() = %q, want %q", g.Name(), "euclidean")
	}
	if r.ArcTraveled != 0 {
		t.Errorf("Ray.ArcTraveled = %v, want 0", r.ArcTraveled)
	}
}

func TestRayGReturnsAttachedGeometry(t *testing.T) {
	r := &Ray{Geometry: geometry.Spherical()}
	if r.G().Name() != "spherical" {
		t.Errorf("Ray.G() = %q, want %q", r.G().Name(), "spherical")
	}
}

func TestRayInitResetsArcTraveled(t *testing.T) {
	r := &Ray{ArcTraveled: 5}
	r.Init()
	if r.ArcTraveled != 0 {
		t.Errorf("Init did not reset ArcTraveled: got %v", r.ArcTraveled)
	}
}
```

- [ ] **Step 2: Run; expect failure**

```bash
go -C engine test ./model/optics/... -run TestRay -count=1
```

Expected: `Geometry undefined` / `ArcTraveled undefined` / `G undefined`.

- [ ] **Step 3: Modify `ray.go`**

In `engine/model/optics/ray.go`:

Add import:

```go
import (
	"math"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/model/geometry"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)
```

Extend the struct (add two fields at the bottom, before the closing `}`):

```go
	MediumStack          medium.Stack       `json:"-"`
	Geometry             geometry.Geometry  `json:"-"` // nil ⇒ Euclidean (back-compat default)
	ArcTraveled          float64            `json:"-"` // geodesic arc length traveled so far (S^3 wrap)
}
```

In `Init` (the function starting near line 25), at the end, before the closing `}`:

```go
	r.ArcTraveled = 0
	// Geometry is intentionally NOT reset: it is set per-render by the
	// renderer when handing out a Ray, and Init may be called from a
	// pool that pre-assigns it. Setting it to nil here would break the
	// non-Euclidean integrator.
}
```

Add the helper method at the bottom of the file:

```go
// G returns the ray's geometry, falling back to Euclidean if unset.
func (r *Ray) G() geometry.Geometry {
	return geometry.Get(r.Geometry)
}
```

- [ ] **Step 4: Run**

```bash
go -C engine test ./model/optics/... -run TestRay -count=1
```

Expected: PASS.

- [ ] **Step 5: Full-package regression**

```bash
go -C engine test ./model/optics/... -count=1
```

Expected: PASS (existing ray tests untouched).

- [ ] **Step 6: Commit**

```bash
git add engine/model/optics/ray.go engine/model/optics/ray_geometry_test.go
git commit -m "feat(optics): add Geometry and ArcTraveled fields to Ray"
```

---

## Task 5: Frame construction in a geometry

**Files:**
- Create: `engine/maths/frame_geometry.go`
- Create: `engine/maths/frame_geometry_test.go`

- [ ] **Step 1: Write the failing test**

Create `engine/maths/frame_geometry_test.go`:

```go
package maths

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/geometry"
	"gonum.org/v1/gonum/mat"
)

func TestFrameInEuclideanMatchesPlainFrame(t *testing.T) {
	n := mat.NewVecDense(3, []float64{0, 0, 1})
	plain, _ := NewFrameFromNormal(n)
	g := geometry.Euclidean()
	got, ok := NewFrameFromNormalInGeometry(g, mat.NewVecDense(3, []float64{1, 2, 3}), n)
	if !ok {
		t.Fatal("NewFrameFromNormalInGeometry returned ok=false")
	}
	if got.Normal.AtVec(2) != plain.Normal.AtVec(2) {
		t.Errorf("normal differs from plain frame")
	}
}

func TestFrameInSphericalOrthogonalToPosition(t *testing.T) {
	g := geometry.Spherical()
	p := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	// A normal that has a radial component; should be projected out.
	n := mat.NewVecDense(4, []float64{0.5, 0, 0, 1})
	f, ok := NewFrameFromNormalInGeometry(g, p, n)
	if !ok {
		t.Fatal("frame construction failed")
	}
	// f.Normal must be orthogonal to p in R^4.
	if d := mat.Dot(f.Normal, p); math.Abs(d) > 1e-9 {
		t.Errorf("frame normal not in T_p: <n,p> = %v", d)
	}
	// All tangents must also be in T_p.
	for i, tt := range f.Tangents {
		if d := mat.Dot(tt, p); math.Abs(d) > 1e-9 {
			t.Errorf("tangent[%d] not in T_p: <t,p> = %v", i, d)
		}
	}
}
```

- [ ] **Step 2: Run; expect failure**

```bash
go -C engine test ./maths/... -run TestFrameIn -count=1
```

Expected: `undefined: NewFrameFromNormalInGeometry`.

- [ ] **Step 3: Implement `frame_geometry.go`**

Create `engine/maths/frame_geometry.go`:

```go
package maths

import (
	"github.com/Algo2147483647/ray/engine/model/geometry"
	"gonum.org/v1/gonum/mat"
)

// NewFrameFromNormalInGeometry builds an orthonormal frame at point p
// whose normal direction is the geometry-projected version of n.
//
// For Euclidean/Klein this delegates to NewFrameFromNormal: the existing
// Gram-Schmidt in R^d already produces the right basis.
//
// For Spherical we first project both the normal and every candidate
// tangent direction into T_p S^3, so the resulting frame lives in the
// 3-dimensional tangent subspace embedded in R^4.
func NewFrameFromNormalInGeometry(g geometry.Geometry, p, n *mat.VecDense) (Frame, bool) {
	g = geometry.Get(g)
	if g.Name() != "spherical" {
		return NewFrameFromNormal(n)
	}

	// Project the normal into T_p, then orthonormalize.
	projected := mat.NewVecDense(n.Len(), nil)
	g.ProjectTangent(p, n, projected)
	if mat.Norm(projected, 2) <= 1e-12 {
		return Frame{}, false
	}
	Normalize(projected)

	dim := n.Len()
	tangents := make([]*mat.VecDense, 0, dim-2)
	for axis := 0; axis < dim && len(tangents) < dim-2; axis++ {
		candidate := mat.NewVecDense(dim, nil)
		candidate.SetVec(axis, 1)
		// Remove components along p (radial) and along the normal.
		candidate.AddScaledVec(candidate, -mat.Dot(candidate, p), p)
		candidate.AddScaledVec(candidate, -mat.Dot(candidate, projected), projected)
		for _, t := range tangents {
			candidate.AddScaledVec(candidate, -mat.Dot(candidate, t), t)
		}
		if mat.Norm(candidate, 2) <= 1e-12 {
			continue
		}
		tangents = append(tangents, Normalize(candidate))
	}
	if len(tangents) != dim-2 {
		return Frame{}, false
	}

	var tangent, bitangent *mat.VecDense
	if len(tangents) > 0 {
		tangent = tangents[0]
	}
	if len(tangents) > 1 {
		bitangent = tangents[1]
	}

	return Frame{
		Tangent:   tangent,
		Bitangent: bitangent,
		Normal:    projected,
		Tangents:  tangents,
	}, true
}
```

- [ ] **Step 4: Run**

```bash
go -C engine test ./maths/... -count=1
```

Expected: PASS, including the new frame-in-geometry tests, and all existing frame tests still pass.

- [ ] **Step 5: Commit**

```bash
git add engine/maths/frame_geometry.go engine/maths/frame_geometry_test.go
git commit -m "feat(maths): add NewFrameFromNormalInGeometry for non-Euclidean tangent frames"
```

---

## Task 6: Wire geometry into `trace_ray.go`

**Files:**
- Modify: `engine/ray_tracing/trace_ray.go`
- Create: `engine/ray_tracing/trace_ray_geometry_test.go`

This task changes the integrator to (a) translate `hit.Distance` to arc length before passing to `applyMediumAbsorption`, (b) construct frames via `NewFrameFromNormalInGeometry`, (c) reproject the sampled outgoing direction into the tangent space at the hit point, and (d) accumulate `ArcTraveled`. The S³ wrap loop is added in Task 7.

- [ ] **Step 1: Write the failing test**

Create `engine/ray_tracing/trace_ray_geometry_test.go`:

```go
package ray_tracing

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/geometry"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
)

// TestArcLengthFedToMediumAbsorption asserts that when ray.Geometry is
// Klein, the medium-absorption code receives the hyperbolic arc length
// rather than the embedded t. We exercise this through a tiny scene:
// a ray of unit direction starting at the origin with a sphere at t=0.5
// inside an absorbing medium.
//
// Implementation note: this test goes through TraceRay end-to-end; it
// is sensitive to changes in the trace-ray contract. If it breaks, see
// docs/superpowers/specs/2026-05-31-non-euclidean-rendering-design.md §5.
func TestKleinRayArcAffectsAbsorption(t *testing.T) {
	// Smoke test: a ray with Geometry=Klein computes a different
	// post-absorption spectral power than the same ray with Geometry=nil.
	// The actual computation is in applyMediumAbsorption, but we verify
	// the wiring here by computing arc length directly and confirming it
	// differs from t.
	g := geometry.Klein()
	p := mat.NewVecDense(3, []float64{0, 0, 0})
	d := mat.NewVecDense(3, []float64{1, 0, 0})
	tEmbed := 0.5
	arc := g.ArcLengthFromEmbedT(p, d, tEmbed)
	if math.Abs(arc-math.Atanh(0.5)) > 1e-9 {
		t.Fatalf("Klein arc length wiring wrong: got %v, want %v", arc, math.Atanh(0.5))
	}
	if arc == tEmbed {
		t.Fatal("Klein arc length must differ from embedded t at non-zero radius")
	}
	_ = &optics.Ray{Geometry: g} // ensure the field compiles in the import graph
}

func TestEuclideanRayArcEqualsT(t *testing.T) {
	g := geometry.Euclidean()
	p := mat.NewVecDense(3, []float64{0, 0, 0})
	d := mat.NewVecDense(3, []float64{1, 0, 0})
	if g.ArcLengthFromEmbedT(p, d, 2.5) != 2.5 {
		t.Fatal("Euclidean arc length should equal t for unit direction")
	}
}
```

- [ ] **Step 2: Run; expect PASS even without changes (it tests the geometry contract, not the integrator)**

```bash
go -C engine test ./ray_tracing/... -run TestKleinRayArc -count=1
```

Expected: PASS. The integrator changes below have NO test-driven failure mode — they are mechanical refactors guarded by:
(a) the geometry unit tests (Tasks 1–3),
(b) full-package regression (Step 6 below),
(c) Task 9 integration scenes.

This is intentional: end-to-end ray-tracer behavior is hard to assert at unit granularity; we instead protect with the regression test and the integration scenes. **Do not add speculative integrator unit tests here.**

- [ ] **Step 3: Modify `trace_ray.go` — translate distance to arc length**

In `engine/ray_tracing/trace_ray.go`, locate the `TraceRay` method body around lines 22–66. Replace the medium-absorption block (currently calling `applyMediumAbsorption(media, ray, hit.Distance, mediumCtx)`) with:

```go
	// Translate the embedded-domain ray parameter into geodesic arc length
	// before doing anything physical with it (absorption, direction update,
	// arc-budget bookkeeping).
	g := ray.G()
	arcLen := g.ArcLengthFromEmbedT(ray.Origin, ray.Direction, hit.Distance)

	// Apply medium absorption accumulated along the segment before the hit point.
	media := getMediumRegistry(objTree)
	mediumCtx := h.newShadingContext(ray)
	applyMediumAbsorption(media, ray, arcLen, mediumCtx)

	// Track total geodesic distance traveled (used by the S^3 wrap loop and
	// as a fail-safe even on flat geometries).
	ray.ArcTraveled += arcLen
```

(Remove the old two lines that defined `media` / `mediumCtx` / called `applyMediumAbsorption(... hit.Distance ...)`; they're consolidated above.)

- [ ] **Step 4: Modify `trace_ray.go` — use geometry-aware Frame and reproject direction**

In `prepareSurfaceInteraction` (around line 90), replace:

```go
	frame, ok := maths.NewFrameFromNormal(hit.ShadingNormal)
```

with:

```go
	frame, ok := maths.NewFrameFromNormalInGeometry(ray.G(), hit.Point, hit.ShadingNormal)
```

In `TraceRay` at the bottom (around lines 60–65), replace:

```go
	si.Frame.LocalToWorldInto(ray.Direction, sample.Wi)
	maths.Normalize(ray.Direction)
```

with:

```go
	si.Frame.LocalToWorldInto(ray.Direction, sample.Wi)
	// Project the sampled outgoing direction back into T_p of the current
	// geometry, then renormalize using the geometry's inner product so the
	// next embedded-ray intersection is parameterized correctly.
	ray.G().ProjectTangent(ray.Origin, ray.Direction, ray.Direction)
	maths.Normalize(ray.Direction)
```

(The Euclidean and Klein ProjectTangent are identity; only Spherical does work.)

- [ ] **Step 5: Run full ray-tracing tests**

```bash
go -C engine test ./ray_tracing/... -count=1
```

Expected: PASS. Existing tests must still pass because the default geometry is Euclidean, where `ArcLengthFromEmbedT(p, d, t) = t * |d|` and `ProjectTangent` is identity — semantically identical to the old code (modulo the `* |d|` factor, which equals 1 for normalized directions).

- [ ] **Step 6: Run engine-wide tests**

```bash
go -C engine test ./... -count=1
```

Expected: PASS. If any test fails (especially in `engine/controller/`), the most likely cause is a ray with non-unit `Direction` being affected by the `* |d|` factor — investigate before proceeding.

- [ ] **Step 7: Commit**

```bash
git add engine/ray_tracing/trace_ray.go engine/ray_tracing/trace_ray_geometry_test.go
git commit -m "refactor(trace): route distance through Geometry.ArcLengthFromEmbedT

- Translate hit.Distance to geodesic arc length before medium absorption
- Construct shading frame via NewFrameFromNormalInGeometry
- Re-project sampled outgoing direction into T_p
- Track ArcTraveled per ray (used by S^3 wrap loop in upcoming task)"
```

---

## Task 7: S³ wrap loop and `max_arc` enforcement

**Files:**
- Modify: `engine/ray_tracing/trace_ray.go`
- Modify: `engine/ray_tracing/handler.go`
- Create: `engine/ray_tracing/trace_ray_wrap_test.go`

- [ ] **Step 1: Add `MaxArc` to `Handler`**

In `engine/ray_tracing/handler.go`, around line 11 (the `Handler` struct), add a field:

```go
type Handler struct {
	MaxRayLevel          int64                    `json:"max_ray_level"`
	RussianRouletteDepth int64                    `json:"russian_roulette_depth"`
	MaxArc               float64                  `json:"max_arc"` // total geodesic distance budget per ray (0 ⇒ unbounded)
	// ... rest unchanged
```

In `NewHandler()`, add a default (right after `RussianRouletteDepth: 3,`):

```go
		MaxArc:               0, // 0 means unbounded; set by scene factory for spherical scenes.
```

- [ ] **Step 2: Add the wrap helper and per-bounce termination**

In `engine/ray_tracing/trace_ray.go`, modify `TraceRay` near the top:

Add to the existing `terminateBeforeBounce` check or as a separate guard after computing `arcLen`:

```go
	// Geodesic-budget kill (used primarily by S^3 to bound the wrap loop).
	if h.MaxArc > 0 && ray.ArcTraveled >= h.MaxArc {
		terminateRay(ray)
		return
	}
```

Place this immediately after `ray.ArcTraveled += arcLen` from Task 6.

For the "no hit" branch (currently around line 27: `if !ok { terminateRay(ray); return }`), replace with:

```go
	if !ok {
		// Spherical: the chord may have left the visible hemisphere without
		// hitting anything; wrap past the antipode and continue tracing if
		// we still have arc budget.
		if newO, newD, wrapped := ray.G().WrapBeyond(ray.Origin, ray.Direction, math.Pi); wrapped {
			advance := math.Pi
			if h.MaxArc > 0 && ray.ArcTraveled+advance > h.MaxArc {
				terminateRay(ray)
				return
			}
			ray.Origin.CopyVec(newO)
			ray.Direction.CopyVec(newD)
			ray.ArcTraveled += advance
			h.TraceRay(objTree, ray, level+1)
			return
		}
		terminateRay(ray)
		return
	}
```

Add the import for `math` at the top of `trace_ray.go` if not already present.

- [ ] **Step 3: Write the wrap test**

Create `engine/ray_tracing/trace_ray_wrap_test.go`:

```go
package ray_tracing

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/geometry"
	"gonum.org/v1/gonum/mat"
)

// TestSphericalWrapAdvancesByPi verifies that the WrapBeyond contract used
// inside TraceRay (arcAdvance = π) lands at the antipode with transported
// direction. This is the same invariant that the no-hit branch of TraceRay
// relies on.
func TestSphericalWrapAdvancesByPi(t *testing.T) {
	g := geometry.Spherical()
	p := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	d := mat.NewVecDense(4, []float64{0, 1, 0, 0})
	newP, newD, ok := g.WrapBeyond(p, d, math.Pi)
	if !ok {
		t.Fatal("WrapBeyond ok=false")
	}
	if math.Abs(newP.AtVec(0)+1) > 1e-9 {
		t.Errorf("did not land at antipode: newP[0] = %v", newP.AtVec(0))
	}
	if math.Abs(newD.AtVec(1)+1) > 1e-9 {
		t.Errorf("direction not transported: newD[1] = %v", newD.AtVec(1))
	}
}

// TestMaxArcZeroIsUnbounded asserts the default MaxArc=0 does not kill rays.
func TestMaxArcZeroIsUnbounded(t *testing.T) {
	h := NewHandler()
	if h.MaxArc != 0 {
		t.Errorf("default MaxArc = %v, want 0 (unbounded)", h.MaxArc)
	}
}
```

- [ ] **Step 4: Run**

```bash
go -C engine test ./ray_tracing/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Engine-wide regression**

```bash
go -C engine test ./... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add engine/ray_tracing/trace_ray.go engine/ray_tracing/handler.go engine/ray_tracing/trace_ray_wrap_test.go
git commit -m "feat(trace): S^3 wrap loop and MaxArc geodesic budget"
```

---

## Task 8: Hyperbolic and Spherical cameras

**Files:**
- Create: `engine/model/camera/camera_hyperbolic.go`
- Create: `engine/model/camera/camera_hyperbolic_test.go`
- Create: `engine/model/camera/camera_spherical.go`
- Create: `engine/model/camera/camera_spherical_test.go`

### 8a — Hyperbolic camera

In Klein H³ the camera's position lies inside the unit ball; `Direction` and `Up` are ordinary R³ vectors (interpreted as Klein tangent vectors at `Position`). Because Klein geodesics are Euclidean chords, ray generation is **byte-identical** to `Camera3D` except it sets `ray.Geometry = geometry.Klein()`.

- [ ] **Step 1: Write the failing Hyperbolic camera test**

Create `engine/model/camera/camera_hyperbolic_test.go`:

```go
package camera

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/model/geometry"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
)

func TestHyperbolicCameraAttachesKleinGeometry(t *testing.T) {
	c := &HyperbolicCamera{
		Camera3D: Camera3D{
			Position:    mat.NewVecDense(3, []float64{0, 0, 0}),
			Direction:   mat.NewVecDense(3, []float64{1, 0, 0}),
			Up:          mat.NewVecDense(3, []float64{0, 0, 1}),
			Width:       4, Height: 4,
			FieldOfView: 60, AspectRatio: 1,
		},
	}
	r := &optics.Ray{}
	c.GenerateRay(r, 0, 0)
	if r.G() != geometry.Klein() {
		t.Errorf("HyperbolicCamera did not set Geometry to Klein; got %v", r.G().Name())
	}
}
```

- [ ] **Step 2: Run; expect failure**

```bash
go -C engine test ./model/camera/... -run TestHyperbolic -count=1
```

Expected: `undefined: HyperbolicCamera`.

- [ ] **Step 3: Implement `camera_hyperbolic.go`**

Create `engine/model/camera/camera_hyperbolic.go`:

```go
package camera

import (
	"github.com/Algo2147483647/ray/engine/model/geometry"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
)

// HyperbolicCamera is a Camera3D whose generated rays are tagged with the
// Klein H^3 geometry. Ray generation logic is otherwise identical: in the
// Klein model, the chord from the camera position with any embedded
// direction is the geodesic.
type HyperbolicCamera struct {
	Camera3D
}

func NewHyperbolicCamera() *HyperbolicCamera { return &HyperbolicCamera{} }

func (c *HyperbolicCamera) GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray {
	res = c.Camera3D.GenerateRay(res, index...)
	res.Geometry = geometry.Klein()
	return res
}
```

- [ ] **Step 4: Run**

```bash
go -C engine test ./model/camera/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add engine/model/camera/camera_hyperbolic.go engine/model/camera/camera_hyperbolic_test.go
git commit -m "feat(camera): hyperbolic camera over Camera3D with Klein geometry tag"
```

### 8b — Spherical camera

This one is more involved because rays live in R⁴: position is a unit vector in S³, and the tangent plane at it is 3-dimensional.

- [ ] **Step 6: Write the failing Spherical camera test**

Create `engine/model/camera/camera_spherical_test.go`:

```go
package camera

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/geometry"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
)

func TestSphericalCameraGeneratesUnitTangentRay(t *testing.T) {
	c := &SphericalCamera{
		Position:    mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		Forward:     mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		Up:          mat.NewVecDense(4, []float64{0, 0, 1, 0}),
		Width:       4, Height: 4,
		FieldOfView: 60, AspectRatio: 1,
	}
	r := &optics.Ray{}
	// Init via the camera; Origin/Direction may need 4-component allocation.
	r.Origin = mat.NewVecDense(4, nil)
	r.Direction = mat.NewVecDense(4, nil)
	c.GenerateRay(r, 0, 0)

	if r.G() != geometry.Spherical() {
		t.Errorf("SphericalCamera did not set Geometry to Spherical")
	}

	// Origin must equal Position.
	for i := 0; i < 4; i++ {
		if math.Abs(r.Origin.AtVec(i)-c.Position.AtVec(i)) > 1e-9 {
			t.Errorf("origin[%d] = %v, want %v", i, r.Origin.AtVec(i), c.Position.AtVec(i))
		}
	}

	// Direction must be in T_p: orthogonal to Position.
	if d := mat.Dot(r.Direction, c.Position); math.Abs(d) > 1e-9 {
		t.Errorf("ray direction has radial component: <d,p> = %v", d)
	}
}
```

- [ ] **Step 7: Run; expect failure**

```bash
go -C engine test ./model/camera/... -run TestSpherical -count=1
```

Expected: `undefined: SphericalCamera`.

- [ ] **Step 8: Implement `camera_spherical.go`**

Create `engine/model/camera/camera_spherical.go`:

```go
package camera

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/geometry"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
)

// SphericalCamera lives on S^3 embedded in R^4. Position is a unit vector;
// Forward and Up are R^4 vectors interpreted as tangent vectors at Position
// (they are projected into T_p and orthonormalized at Prepare time).
type SphericalCamera struct {
	CameraBase
	Position    *mat.VecDense
	Forward     *mat.VecDense
	Up          *mat.VecDense
	Width       int
	Height      int
	FieldOfView float64 // degrees
	AspectRatio float64

	forward    *mat.VecDense
	up         *mat.VecDense
	right      *mat.VecDense
	halfWidth  float64
	halfHeight float64
	invWidth2  float64
	invHeight2 float64
	prepared   bool
}

func NewSphericalCamera() *SphericalCamera { return &SphericalCamera{} }

func (c *SphericalCamera) Prepare() error {
	if c.Position == nil || c.Forward == nil || c.Up == nil {
		return fmt.Errorf("spherical camera requires position, forward, up")
	}
	if c.Width <= 0 || c.Height <= 0 || c.FieldOfView <= 0 || c.AspectRatio <= 0 {
		return fmt.Errorf("spherical camera requires positive width, height, fov, aspect ratio")
	}

	g := geometry.Spherical()

	// Renormalize position onto S^3.
	pos := mat.VecDenseCopyOf(c.Position)
	maths.Normalize(pos)
	c.Position = pos

	// Project Forward and Up into T_p, then orthonormalize.
	fwd := mat.NewVecDense(4, nil)
	g.ProjectTangent(c.Position, c.Forward, fwd)
	if mat.Norm(fwd, 2) == 0 {
		return fmt.Errorf("forward direction collapses in T_p")
	}
	maths.Normalize(fwd)

	up := mat.NewVecDense(4, nil)
	g.ProjectTangent(c.Position, c.Up, up)
	up.AddScaledVec(up, -mat.Dot(up, fwd), fwd)
	if mat.Norm(up, 2) == 0 {
		return fmt.Errorf("up direction collapses after orthogonalization")
	}
	maths.Normalize(up)

	// Right = the third tangent direction: orthogonal in T_p to both fwd and up
	// and to Position. Find a coordinate axis with the smallest projection and
	// Gram-Schmidt against (Position, fwd, up).
	right := orthogonalInTangent(c.Position, fwd, up)
	if right == nil {
		return fmt.Errorf("could not construct right vector in T_p")
	}
	c.forward, c.up, c.right = fwd, up, right

	fovRad := c.FieldOfView * math.Pi / 180
	c.halfHeight = math.Tan(fovRad / 2)
	c.halfWidth = c.AspectRatio * c.halfHeight
	c.invWidth2 = 2 / float64(c.Width)
	c.invHeight2 = 2 / float64(c.Height)
	c.prepared = true
	return nil
}

// orthogonalInTangent returns a unit vector in T_p orthogonal to a and b.
// p, a, b are assumed orthonormal already (p radial; a, b in T_p, mutually
// orthonormal). Probes each coordinate axis, orthogonalizes, picks first
// non-degenerate.
func orthogonalInTangent(p, a, b *mat.VecDense) *mat.VecDense {
	for axis := 0; axis < 4; axis++ {
		cand := mat.NewVecDense(4, nil)
		cand.SetVec(axis, 1)
		cand.AddScaledVec(cand, -mat.Dot(cand, p), p)
		cand.AddScaledVec(cand, -mat.Dot(cand, a), a)
		cand.AddScaledVec(cand, -mat.Dot(cand, b), b)
		if mat.Norm(cand, 2) > 1e-9 {
			maths.Normalize(cand)
			return cand
		}
	}
	return nil
}

func (c *SphericalCamera) GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray {
	if res == nil {
		res = &renderray.Ray{}
	}
	res.Init()
	if !c.prepared {
		if err := c.Prepare(); err != nil {
			panic(err)
		}
	}

	row, col := index[0], index[1]
	u := (float64(row)+rand.Float64())*c.invWidth2 - 1
	v := (float64(col)+rand.Float64())*c.invHeight2 - 1

	if res.Origin.Len() != 4 {
		res.Origin = mat.NewVecDense(4, nil)
	}
	if res.Direction.Len() != 4 {
		res.Direction = mat.NewVecDense(4, nil)
	}

	res.Origin.CopyVec(c.Position)
	res.Direction.CopyVec(c.forward)
	res.Direction.AddScaledVec(res.Direction, u*c.halfWidth, c.right)
	res.Direction.AddScaledVec(res.Direction, -v*c.halfHeight, c.up)
	// Direction already lives in T_p (sum of T_p vectors). Normalize.
	maths.Normalize(res.Direction)

	res.Geometry = geometry.Spherical()
	return res
}
```

- [ ] **Step 9: Run**

```bash
go -C engine test ./model/camera/... -count=1
```

Expected: PASS.

- [ ] **Step 10: Engine-wide regression**

```bash
go -C engine test ./... -count=1
```

Expected: PASS.

- [ ] **Step 11: Commit**

```bash
git add engine/model/camera/camera_spherical.go engine/model/camera/camera_spherical_test.go
git commit -m "feat(camera): spherical camera on S^3 embedded in R^4"
```

---

## Task 9: Scene wiring (schema, factory, dispatch)

**Files:**
- Modify: `engine/controller/parser/schema.go` (around line 64, the `Script` struct)
- Modify: `engine/controller/factory/scene.go` (around line 13, `LoadSceneFromScript`)
- Modify: `engine/controller/factory/cameras.go` (around line 177, `BuildCameraFromScript`)
- Modify: `engine/model/scene.go`

- [ ] **Step 1: Add `Geometry` field to Scene**

In `engine/model/scene.go`, modify the struct:

```go
package model

import (
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/geometry"
	"github.com/Algo2147483647/ray/engine/model/object"
)

type Scene struct {
	ObjectTree *object.ObjectTree `json:"object_tree"`
	Cameras    []camera.Camera    `json:"cameras"`
	Geometry   geometry.Geometry  `json:"-"` // nil ⇒ Euclidean
	MaxArc     float64            `json:"-"` // 0 ⇒ unbounded
}
```

- [ ] **Step 2: Add `GeometryScript` to schema**

In `engine/controller/parser/schema.go`, just below `RenderScript` (around line 41), add:

```go
type GeometryScript struct {
	Type   string  `json:"type"`    // "euclidean" | "klein" | "spherical"
	MaxArc float64 `json:"max_arc"` // total geodesic budget per ray; 0 ⇒ defaults (∞ for klein/euclidean, 2π for spherical)
}
```

In the `Script` struct (around line 64), add a field:

```go
type Script struct {
	Includes  []string                          `json:"includes"`
	Render    RenderScript                      `json:"render"`
	Geometry  *GeometryScript                   `json:"geometry"`
	// ... rest unchanged
```

(Insert after `Render`; preserve all other existing fields.)

- [ ] **Step 3: Parse geometry in scene factory**

In `engine/controller/factory/scene.go`, add to the imports:

```go
	"github.com/Algo2147483647/ray/engine/model/geometry"
	"strings"
```

Inside `LoadSceneFromScript`, after `scene.Cameras = nil` (around line 20), add:

```go
	// Resolve scene geometry. Default is Euclidean (nil sentinel).
	if script.Geometry != nil {
		switch strings.ToLower(script.Geometry.Type) {
		case "", "euclidean":
			scene.Geometry = nil
		case "klein", "hyperbolic":
			scene.Geometry = geometry.Klein()
		case "spherical", "sphere":
			scene.Geometry = geometry.Spherical()
		default:
			return fmt.Errorf("unsupported geometry type %q", script.Geometry.Type)
		}
		scene.MaxArc = script.Geometry.MaxArc
		if scene.MaxArc == 0 && scene.Geometry == geometry.Spherical() {
			scene.MaxArc = 2 * math.Pi
		}
	}
```

Add to imports if not present: `"math"`.

- [ ] **Step 4: Dispatch new camera types**

In `engine/controller/factory/cameras.go`, modify `BuildCameraFromScript` (around line 177):

```go
func BuildCameraFromScript(def parser.CameraScript) (modelcamera.Camera, error) {
	switch strings.ToLower(def.Type) {
	case "", "3d", "camera3d":
		return BuildCamera3DFromScript(def)
	case "n_dim", "ndim", "n-dimensional":
		return BuildCameraNDimFromScript(def)
	case "hyperbolic", "klein":
		return BuildHyperbolicCameraFromScript(def)
	case "spherical", "s3":
		return BuildSphericalCameraFromScript(def)
	default:
		return nil, fmt.Errorf("unsupported camera type %q", def.Type)
	}
}

func BuildHyperbolicCameraFromScript(def parser.CameraScript) (*modelcamera.HyperbolicCamera, error) {
	base, err := BuildCamera3DFromScript(def)
	if err != nil {
		return nil, err
	}
	return &modelcamera.HyperbolicCamera{Camera3D: *base}, nil
}

func BuildSphericalCameraFromScript(def parser.CameraScript) (*modelcamera.SphericalCamera, error) {
	if utils.Dimension != 4 {
		return nil, fmt.Errorf("spherical camera requires render dimension 4, got %d", utils.Dimension)
	}
	position, err := vectorFromScript("position", def.Position)
	if err != nil {
		return nil, err
	}
	forward, err := vectorFromScript("direction", def.Direction)
	if err != nil {
		return nil, err
	}
	up, err := vectorFromScript("up", def.Up)
	if err != nil {
		return nil, err
	}
	aspect := def.AspectRatio
	if aspect <= 0 && def.Width > 0 && def.Height > 0 {
		aspect = float64(def.Width) / float64(def.Height)
	}
	cam := &modelcamera.SphericalCamera{
		Position:    position,
		Forward:     forward,
		Up:          up,
		Width:       def.Width,
		Height:      def.Height,
		FieldOfView: def.FieldOfView,
		AspectRatio: aspect,
	}
	if err := cam.Prepare(); err != nil {
		return nil, err
	}
	return cam, nil
}
```

- [ ] **Step 5: Propagate `Scene.Geometry` to handler**

Find where the renderer applies per-render configuration. Search:

```bash
grep -n "Cameras\|ObjectTree" engine/ray_tracing/trace_scene.go engine/controller/handler.go
```

In whichever file constructs/configures the `Handler` for a scene, after the handler is created and before `TraceTiles`/`TracePixel` is invoked, set:

```go
	handler.MaxArc = scene.MaxArc
```

And in the per-ray dispatch (where `RayPool.Get()` produces a `*optics.Ray`), assign:

```go
	ray.Geometry = scene.Geometry
```

If the existing code path does not have a clean injection point, add a `SceneGeometry geometry.Geometry` field to `Handler` set by the caller, and assign it onto each `ray` after `Init()` in `trace_pixel.go`. **Do not refactor existing pool reset semantics**: the comment in `ray.Init` already calls out that `Geometry` is intentionally not cleared.

Concrete edit (most likely in `engine/ray_tracing/trace_pixel.go` or `trace_scene.go`): after every `ray := h.RayPool.Get().(*optics.Ray)` and `ray.Init()`, add `ray.Geometry = h.SceneGeometry` (creating that field on `Handler`).

Add to `Handler` struct in `handler.go`:

```go
	SceneGeometry        geometry.Geometry        `json:"-"`
```

(Import `"github.com/Algo2147483647/ray/engine/model/geometry"`.)

In the caller (factory / controller handler that builds the `Handler`), set `handler.SceneGeometry = scene.Geometry` and `handler.MaxArc = scene.MaxArc`.

- [ ] **Step 6: Run engine-wide tests**

```bash
go -C engine test ./... -count=1
```

Expected: PASS. Existing scenes have no `"geometry"` field → `scene.Geometry = nil` → `ray.G()` returns Euclidean → byte-equivalent to before.

- [ ] **Step 7: Commit**

```bash
git add engine/model/scene.go engine/controller/parser/schema.go engine/controller/factory/scene.go engine/controller/factory/cameras.go engine/ray_tracing/handler.go engine/ray_tracing/trace_pixel.go engine/ray_tracing/trace_scene.go
git commit -m "feat(scene): geometry & max_arc schema, dispatch, handler wiring"
```

---

## Task 10: Validation scene — hyperbolic chessboard

**Files:**
- Create: `examples/scenes/hyperbolic_chessboard.json`

- [ ] **Step 1: Inspect an existing simple scene to match formatting**

```bash
ls examples/scenes/ | head -20
cat examples/scenes/default.json | head -80
```

Note the JSON shape: `render`, `cameras`, `materials`, `objects` blocks. Use the same indentation and field ordering.

- [ ] **Step 2: Create `examples/scenes/hyperbolic_chessboard.json`**

This is a Klein-ball scene with a checkered plane at z=0 (inside the unit ball) and a couple of Lambert spheres for depth cue. Use existing material types found in `examples/scenes/` — copy from a working scene rather than inventing material IDs.

```json
{
  "render": {
    "dimension": 3,
    "samples": 64,
    "width": 400,
    "height": 400,
    "output_image": "../../outputs/hyperbolic_chessboard.png"
  },
  "geometry": {
    "type": "klein",
    "max_arc": 0
  },
  "cameras": [
    {
      "id": "main",
      "type": "hyperbolic",
      "position": [0.0, 0.0, 0.3],
      "look_at":  [0.5, 0.0, 0.0],
      "up":       [0.0, 0.0, 1.0],
      "width": 400, "height": 400,
      "field_of_view": 90.0,
      "aspect_ratio": 1.0
    }
  ],
  "materials": [
    { "id": "white_lambert", "surface": { "type": "lambert", "albedo": [0.9, 0.9, 0.9] } },
    { "id": "black_lambert", "surface": { "type": "lambert", "albedo": [0.05, 0.05, 0.05] } },
    { "id": "red_mirror",    "surface": { "type": "mirror",  "albedo": [0.9, 0.2, 0.2] } }
  ],
  "objects": [
    { "id": "floor", "material_id": "white_lambert",
      "shape": { "type": "plane", "A": [0, 0, 1], "b": 0.02 } },
    { "id": "ball1", "material_id": "red_mirror",
      "shape": { "type": "sphere", "center": [0.3, 0.0, 0.1], "r": 0.06 } },
    { "id": "ball2", "material_id": "black_lambert",
      "shape": { "type": "sphere", "center": [-0.2, 0.2, 0.1], "r": 0.06 } }
  ]
}
```

If the existing scene schema uses different field names (e.g. `surface` may be nested under `material` differently), adjust **only** the names to match — do not invent new ones. Refer to a known-good scene from `examples/scenes/`.

- [ ] **Step 3: Render it**

```bash
go -C engine run . -scene ../examples/scenes/hyperbolic_chessboard.json
```

Expected: completes without error; produces `outputs/hyperbolic_chessboard.png`.

If the CLI does not accept `-scene`, find the correct flag from `engine/main.go` or `engine/controller/render_config.go`.

- [ ] **Step 4: Visual smoke check (manual)**

Open `outputs/hyperbolic_chessboard.png`. Expected: balls visibly compressed/distorted compared to the Euclidean analogue (where the same coordinates would render in a tight 1-ball region). If the image is uniform background, the camera or scene is mis-set.

- [ ] **Step 5: Commit**

```bash
git add examples/scenes/hyperbolic_chessboard.json
git commit -m "feat(scenes): add hyperbolic chessboard validation scene"
```

---

## Task 11: Validation scene — spherical Hopf rings

**Files:**
- Create: `examples/scenes/spherical_hopf.json`

S³ has dimension 4, so all vectors are 4-component unit vectors.

- [ ] **Step 1: Create the scene**

Create `examples/scenes/spherical_hopf.json`:

```json
{
  "render": {
    "dimension": 4,
    "samples": 32,
    "width": 400,
    "height": 400,
    "output_image": "../../outputs/spherical_hopf.png"
  },
  "geometry": {
    "type": "spherical",
    "max_arc": 6.283185307179586
  },
  "cameras": [
    {
      "id": "main",
      "type": "spherical",
      "position": [1.0, 0.0, 0.0, 0.0],
      "direction": [0.0, 1.0, 0.0, 0.0],
      "up":        [0.0, 0.0, 1.0, 0.0],
      "width": 400, "height": 400,
      "field_of_view": 90.0,
      "aspect_ratio": 1.0
    }
  ],
  "materials": [
    { "id": "ring_red",   "surface": { "type": "lambert", "albedo": [0.9, 0.1, 0.1] } },
    { "id": "ring_blue",  "surface": { "type": "lambert", "albedo": [0.1, 0.1, 0.9] } }
  ],
  "objects": [
    { "id": "ring1", "material_id": "ring_red",
      "shape": { "type": "sphere", "center": [0.0, 0.7071, 0.7071, 0.0], "r": 0.2 } },
    { "id": "ring2", "material_id": "ring_blue",
      "shape": { "type": "sphere", "center": [0.0, 0.7071, 0.0, 0.7071], "r": 0.2 } }
  ]
}
```

(True Hopf-ring torus geometry would need a new shape type; for v1 we approximate with two 4-spheres on the Clifford torus.)

- [ ] **Step 2: Render**

```bash
go -C engine run . -scene ../examples/scenes/spherical_hopf.json
```

Expected: completes; produces `outputs/spherical_hopf.png`. If any sphere intersection code rejects dimension 4, check `sphere.go:46-48` (`if raySt.Len() == 3 ...`) — the fallback should handle 4D analytically.

- [ ] **Step 3: Visual smoke check (manual)**

Open `outputs/spherical_hopf.png`. Expected: see the two colored balls, plus (because of `max_arc = 2π`) ghost reflections from wrap-around. If totally black: ray wrap or camera initialization is wrong.

- [ ] **Step 4: Commit**

```bash
git add examples/scenes/spherical_hopf.json
git commit -m "feat(scenes): add spherical Hopf-rings validation scene"
```

---

## Task 12: Final regression and merge prep

- [ ] **Step 1: Full test suite**

```bash
go -C engine test ./... -count=1
```

Expected: PASS, no skipped tests.

- [ ] **Step 2: Build the binary**

```bash
go -C engine build -o ../bin/ray .
```

Expected: builds cleanly, no warnings.

- [ ] **Step 3: Re-render the default scene to confirm K=0 regression**

```bash
rm -f outputs/output.png
go -C engine run .
ls -la outputs/output.png
```

Expected: produces `outputs/output.png` identical-ish to baseline. (Visual diff if you have a baseline; otherwise eyeball.)

- [ ] **Step 4: Confirm both new scenes still render**

```bash
go -C engine run . -scene ../examples/scenes/hyperbolic_chessboard.json
go -C engine run . -scene ../examples/scenes/spherical_hopf.json
```

Expected: both succeed.

- [ ] **Step 5: Final commit (no-op if nothing changed)**

```bash
git status
```

If clean, no commit needed. If there are uncommitted tweaks from steps above, commit:

```bash
git add -A
git commit -m "chore: final regression pass"
```

- [ ] **Step 6: Hand back to user for merge / PR**

Branch `feature/non-euclidean` is ready. Do not merge or push without explicit user direction.

---

## Self-Review Notes

**Spec coverage:**
- §3 `Geometry` package → Tasks 1, 2, 3 ✓
- §4 interface signature → Task 1 (matches spec exactly) ✓
- §5 data flow (arc translation, frame, projection, wrap) → Tasks 6, 7 ✓
- §6.1 Ray fields → Task 4 ✓
- §6.2 Frame in geometry → Task 5 ✓
- §6.3 Hyperbolic/Spherical cameras → Task 8 ✓
- §6.4 trace_ray edits → Tasks 6, 7 ✓
- §6.5 scene schema → Task 9 ✓
- §6.6 validation scenes → Tasks 10, 11 ✓
- §7 testing strategy → unit tests in each task; integration scenes 10/11; regression in 6/9/12 ✓
- §9 risks: Klein boundary clamp (`maxArcClamp` in Task 2); S³ wrap closed-form (Task 3); Frame degenerate fallback (Task 5 returns `false`); BSDF/embedded ip note (Task 6 step 4 comment) ✓

**Placeholder scan:** No "TBD"/"TODO"/"add appropriate error handling"-style steps. Task 9 step 5 hands the engineer a search command + concrete field names because the exact line numbers depend on how the renderer currently wires `Handler` to `Scene` — but the required edit is fully specified.

**Type consistency:**
- `Geometry` method signatures identical between `geometry.go`, `euclidean.go`, `klein.go`, `spherical.go`.
- `Ray.G()` defined Task 4, used Tasks 6, 7.
- `NewFrameFromNormalInGeometry(g, p, n)` signature consistent between Task 5 definition and Task 6 use.
- `Scene.Geometry` / `Scene.MaxArc` (Task 9 step 1) used as `scene.Geometry` / `scene.MaxArc` (Task 9 step 5).
- `Handler.MaxArc` / `Handler.SceneGeometry` defined Tasks 7/9, set by factory in Task 9 step 5.
- Camera factory dispatch keys (`"hyperbolic"`, `"spherical"`) match scene JSON in Tasks 10, 11.

No drift detected.