# Non-Euclidean Coordinate System

This document specifies the coordinate-system contract that the engine uses
when `geometry.type` is `klein` (H³) or `spherical` (S³). It is the
reference for anyone who:

- writes a non-euclidean scene by hand or by code-gen,
- adds a new shape and needs it to behave correctly in H³ / S³,
- extends the engine to a new geometry (Poincaré, hyperboloid, custom),
- debugs a rendered image that "doesn't feel like the metric I asked for".

The rules apply whether the file is parsed from JSON or written
programmatically. The engine code that implements them lives in
`engine/maths/geometry/`, `engine/model/optics/ray.go`,
`engine/model/camera/camera_hyperbolic.go`, and `engine/ray_tracing/`.

Companion documents: [`mathematical-foundations.md`](mathematical-foundations.md)
for the underlying math, and the design spec under
[`../.agents/spec.md`](../.agents/spec.md).

---

## 1. Three Layers, Not One

The renderer keeps three distinct coordinate layers in mind at all times.
Mixing them is the single most common source of "this should be hyperbolic
but it looks euclidean" bugs.

```
┌─────────────────────────────────────────────────────────┐
│ Layer 3 — INTRINSIC METRIC                              │
│   H^3 distances d_H, angles, areas, volumes, geodesic   │
│   arc length s, parallel transport, dihedral angles.    │
│   This is what "the geometry IS".                       │
└──────────────────────┬──────────────────────────────────┘
                       │ closed-form formulas (Section 3)
┌──────────────────────▼──────────────────────────────────┐
│ Layer 2 — KLEIN EMBEDDING CHART                         │
│   Points: open unit ball B^3 ⊂ R^3, |x|² < 1.           │
│   THIS is what JSON literals mean.                      │
│   THIS is what shape primitives store.                  │
│   THIS is what the camera position is given in.         │
└──────────────────────┬──────────────────────────────────┘
                       │ trivial: identity for points,    │
                       │ project-to-tangent for vectors    │
┌──────────────────────▼──────────────────────────────────┐
│ Layer 1 — HOST EUCLIDEAN R^3                            │
│   The flat ambient space the BVH, ray–AABB, ray–sphere, │
│   and ray–triangle code already lives in.               │
│   Rays are euclidean lines; t is the euclidean ray      │
│   parameter.                                            │
└─────────────────────────────────────────────────────────┘
```

The chart layer (Layer 2) is **the only one that has a literal
representation** in the engine. JSON, in-memory `*mat.VecDense`, BVH AABBs,
and shape fields all store Layer-2 coordinates. Layer 3 is computed from
Layer 2 on demand. Layer 1 is what the host euclidean ray-tracer "thinks"
it is operating on, and the geometry abstraction translates between
Layer 1 ray parameters and Layer 3 physical quantities at well-defined
choke points.

**Hard rule.** A point inside the Klein ball is *both* a Layer-2 chart
coordinate and a real H³ point at Layer 3. A point outside the ball is
neither. The shape, BVH, and intersection code may transiently produce
points outside the ball (e.g. a ray that misses); the integrator rejects
those before any Layer-3 computation.

---

## 2. Why Klein, and What That Costs

H³ has many models (Poincaré ball, upper half-space, hyperboloid in
Minkowski R³,¹, etc.). The engine uses the **Beltrami–Klein** model for
exactly one reason:

> In the Klein model, **H³ geodesics are euclidean line segments (chords)
> of the unit ball**. So the ray (a euclidean line) is automatically a
> geodesic with no extra work.

This makes the ray–shape intersection contract identical to euclidean
ray tracing. Every existing primitive — sphere, cuboid, plane, triangle,
cylinder, quadric, polynomial surface, STL — keeps working. The BVH,
SAH split, AABB tests, and traversal code do not need to know anything
about the metric.

The cost: **Klein is not conformal**. Angles you measure with a
protractor on the rendered image are NOT H³ angles. Shapes you draw
intuitively in Klein coordinates are NOT shapes that are equivariant
under H³ isometries. In particular:

- A Klein "sphere" centered away from the origin is not a hyperbolic
  sphere; it is a hyperbolic ellipsoid.
- A Klein "cube" (axis-aligned cuboid) does not have H³-90° dihedral
  angles unless it is centered at the origin AND its size is solved from
  the Coxeter relation.
- The Klein-coordinate distance `|p − q|` is NOT the H³ distance.

The chart's job is to make ray-tracing tractable, not to make scene
authoring intuitive. **Authoring tools (Python generators, hand-derived
JSON) live at Layer 2 but reason at Layer 3.**

The other models are listed for completeness:

| Model | Geodesics | Angle preserved? | Cost |
|---|---|---|---|
| Klein (chosen) | euclidean chords | no | shape primitives reused as-is |
| Poincaré ball | circular arcs perpendicular to ∂B³ | yes | every shape's intersection routine must be rewritten |
| Upper half-space | vertical lines + arcs perpendicular to {z=0} | yes | likewise |
| Hyperboloid | great hyperbolas in R³,¹ | yes (preserves Minkowski angles) | dimension +1, all primitives lifted |

Adding any of these means implementing a new `geometry.Geometry`
interface plus, in the non-Klein cases, a parallel set of intersection
routines.

---

## 3. The Layer-2 ↔ Layer-3 Translation Table

These are the closed-form formulas the engine uses. They live in
`engine/maths/geometry/klein.go`. All quantities below are evaluated in
Klein coordinates `p, q ∈ B³`, with `n` a euclidean unit vector and
`c ∈ (−1, 1)`.

### 3.1 Distance

```
d_H(p, q) = acosh( (1 − p·q) / sqrt((1 − |p|²)(1 − |q|²)) )
```

For `q = 0`: `d_H(0, p) = atanh(|p|)`. So `|p| → 1` corresponds to
H³ distance `→ ∞`. The Klein boundary is the H³ sphere at infinity.

### 3.2 Tangent-space inner product (Klein metric)

At a point `p`:

```
g_p(u, v) = (u·v)/(1 − |p|²) + (p·u)(p·v)/(1 − |p|²)²
```

Used for: BSDF cosines, frame normalization, ray-direction renormalization
after a bounce. At the origin this collapses to the euclidean dot
product (which is why frames at the origin are exact and frames far
from the origin pick up an `O(|p|²)` error if the engine takes a
shortcut — see Section 8).

### 3.3 Geodesic parameterization

`Exp_p(v, s)` walks the chord through `p` in euclidean direction `v̂`
for *hyperbolic* arc length `s`. The engine inverts `d_H(p, p + t·v̂) = s`
by monotone bisection (Klein has no closed-form for `Exp` in general
chart coordinates; bisection is cheap because `d_H` is monotone in `t`).

The companion routine `ArcLengthFromEmbedT(p, dir, t_eucl)` converts a
Layer-1 ray parameter to a Layer-3 arc length:

```
arc_H = d_H(p, p + t_eucl · dir)
```

This is the choke point where the integrator translates "how far did the
ray go in BVH-land" into "how far did the photon actually travel". After
this conversion, *all* downstream physics (medium absorption, Russian
roulette path budget, S³ wrap budget) uses `arc_H`, never `t_eucl`.

### 3.4 Totally-geodesic surfaces (a.k.a. "H³ planes")

A 2-dimensional totally-geodesic submanifold of H³ — the H³ analogue
of a euclidean plane — is exactly the intersection of an affine
euclidean plane with the open unit ball:

```
{ x ∈ B³ : n · x = c },   |n| = 1,  |c| < 1
```

Every such surface is a Layer-2 affine plane and an H³ geodesic surface
simultaneously. This is why a thin `cuboid` or a `triangle` mesh in
Klein coordinates can act as a real H³ wall: the host euclidean
intersection routine and the metric agree on the surface as a set.

### 3.5 H³ dihedral angle of two totally-geodesic planes

Two H³ planes given as `n₁·x = c₁` and `n₂·x = c₂` (both with unit
euclidean normals) meet at H³ dihedral angle θ where

```
cos θ = (n₁ · n₂ − c₁ c₂) / sqrt((1 − c₁²)(1 − c₂²))
```

Note this is NOT just `n₁ · n₂` (the euclidean angle between the
normals). The euclidean angle and the H³ angle differ unless both
planes pass through the origin (`c₁ = c₂ = 0`), in which case the
metric collapses to euclidean at the origin.

This formula is the **inverse design tool**: given a desired H³ angle
θ, you solve for `(n_i, c_i)` placement.

### 3.6 H³ reflection across a Klein plane

Reflection across the H³ plane `n · x = c` is the projective
involution

```
k = 2 (n · x − c) / (1 − c²)
x' = (x − k n) / (1 − k c)
```

This is NOT euclidean reflection. Euclidean reflection across the same
plane is `x' = x − 2(n·x − c)·n` — affine, no denominator. The Klein
formula above is the chart projection of the SO(3,1) Lorentz reflection
on the hyperboloid, which is the genuine H³ isometry. It preserves
`d_H`, the H³ angle of every plane, and the "5 cubes per edge" property
of the {4,3,5} honeycomb.

This is what the honeycomb code-gen
(`scene-editor/tools/gen_hyperbolic_honeycomb.py`) uses to produce
neighbor cells.

### 3.7 Equilateral H³ triangle (the formula used in Plan A)

For an H³-equilateral triangle with vertices on the equatorial Klein
circle `|p| = r, z = 0`:

```
cosh L = (1 + r² / 2) / (1 − r²)            (edge length)
cos α  = cosh L / (cosh L + 1)               (interior angle)
```

Inverting: given α, solve the system for `r`. With α = 30°,
`cos α = √3/2`, so `cosh L = 2√3 + 3` and `r ≈ 0.8858`.

---

## 4. The Authoring Convention

This is the convention every scene file and code-gen tool follows:

1. **JSON literals are Layer-2 (Klein) coordinates.** A `position` of
   `[0.5, 0, 0]` means the Klein chart point `0.5 ê_x`. Its H³ distance
   from the origin is `atanh(0.5) ≈ 0.549`, NOT `0.5`.

2. **There is no "Layer-3 literal" in JSON.** If you want an object at
   H³ distance 1.6 from the origin along +x, you must compute the
   Klein coordinate yourself: `r = tanh(1.6) ≈ 0.922`, then write
   `[0.922, 0, 0]`. This means hand-written non-euclidean scenes are
   awkward; non-trivial scenes should be code-generated.

3. **Validity domain.** Every position must satisfy `|p|² < 1`, strictly.
   The engine clamps near-boundary cases inside `klein.go`, but the BVH
   builder will accept any euclidean point — a Klein-illegal coordinate
   produces a primitive that intersects, but the rendering will be
   undefined-behavior near the boundary (typically: `arc_H` saturates to
   `1e6` and the segment is treated as fully absorbed). Treat this as a
   bug in your generator, not a feature.

4. **Camera and light positions follow the same rule.** Klein
   coordinates throughout. The hyperbolic camera (see
   `camera_hyperbolic.go`) merely tags rays with `Geometry = Klein()`;
   ray spawning math is unchanged from the standard 3D camera.

5. **Sizes are Klein-relative, not H³-relative.** A `sphere` of radius
   0.05 at position `[0.5, 0, 0]` is a euclidean ball of radius 0.05 in
   Klein coordinates — which represents a hyperbolic *ellipsoid*, not a
   hyperbolic ball. If you want a true H³ ball of H³-radius `R`
   centered at a non-origin point, you must build it with a quadric
   (`quadratic equation` shape).

6. **Angles between primitives are Klein-euclidean, not H³.** Two
   `triangle` walls that look like they make a 60° corner in Klein
   coordinates do *not* make a 60° H³ corner unless they pass through
   the origin. To author H³ angles, derive `(n_i, c_i)` from the
   formula in §3.5 and place vertices accordingly.

This convention is *honest*: the file format never lies about what
"position" or "size" means. The price is that authoring scenes with
specific H³ properties requires an inverse-design step — done in Python
for Plan C, done by hand for Plan A.

---

## 5. The Engine's Three Choke Points

When `ray.Geometry == Klein()` (or any non-trivial geometry), only three
points in the integrator do anything different. They live in
`engine/ray_tracing/trace_ray.go`.

### 5.1 Distance translation (after every hit)

```go
arc_H := g.ArcLengthFromEmbedT(ray.Origin, ray.Direction, hit.Distance)
```

Every downstream consumer of "how far did the ray go" — medium
absorption, S³ wrap budget, RR throughput accounting — uses `arc_H`,
never `hit.Distance` directly.

### 5.2 Frame construction at the hit point

```go
frame, ok := maths.NewFrameFromNormalInGeometry(ray.G(), hit.Point, hit.ShadingNormal)
```

The geometry-aware frame builder ensures the shading basis is orthonormal
under the Klein metric (Section 3.2), not the euclidean dot product.
See Section 8 for a known limitation.

### 5.3 Direction reprojection after a bounce

```go
ray.G().ProjectTangent(ray.Origin, ray.Direction, ray.Direction)
normalizeDirectionInGeometry(ray.G(), ray.Origin, ray.Direction)
```

After the BSDF spits out a sampled direction in the local frame and we
transform it to world coordinates, we project back into the chart's
tangent space at the new origin. For Klein this is the identity (the
Klein chart is a flat embedding); for spherical S³ it subtracts the
radial component to enforce `⟨v, p⟩ = 0`.

Everything else — BVH, AABB, ray-sphere, ray-triangle, BSDF sampling,
emission, ray spawn — is byte-identical to the euclidean code path. This
is the entire reason the design is tractable.

---

## 6. Worked Example: Plan A (30°-30°-30° Triangle)

Goal: an equilateral triangular room whose interior angles sum to 90°,
not 180°. This is impossible in euclidean geometry but routine in H³.

### Layer 3 (intrinsic specification)

- Three H³-totally-geodesic walls.
- Equilateral triangle in some H³-plane.
- Interior angle α = 30° at each vertex.
- Floor and ceiling are also H³ planes (parallel-displaced copies of
  the triangle's plane).

### Layer 3 → Layer 2 derivation

Step 1. Solve for the Klein vertex radius:
```
α = 30°  ⇒  cos α = √3/2
cosh L = cos α / (1 − cos α)    = √3/2 / (1 − √3/2) ≈ 6.464

       (1 + r²/2)
cosh L = ─────────  ⇒  r² = (cosh L − 1) / (cosh L + ½) ≈ 0.7846
       (1 − r²)
       r ≈ 0.8858
```

Step 2. Place vertices on the equator:
```
V0 = ( r,    0,        0)
V1 = (−r/2,  r √3/2,   0)
V2 = (−r/2, −r √3/2,   0)
```
The plane `z = 0` passes through the origin → it is an H³ plane (§3.4),
so the triangle is genuinely planar in H³.

Step 3. Walls. Each wall is a vertical strip carved by two triangles.
Verticality (`z` axis) is fine because the planes spanned by `(V_i, V_j,
ê_z)` pass through the origin in their normal direction → they are H³
planes too.

Step 4. **Independent verification** of the dihedral angle. Two adjacent
walls have unit normals
```
n_W01 = (−½, −√3/2, 0),   c_W01 = −r/2
n_W20 = (−½, +√3/2, 0),   c_W20 = −r/2
```
By §3.5:
```
cos θ = (n_W01 · n_W20 − c_W01 c_W20) / sqrt((1 − c²)(1 − c²))
      = (¼ − ¾ − r²/4) / (1 − r²/4)
      = √3/2     ⇒  θ = 30°    ✓
```
Two formulas (vertex-side §3.7 and face-side §3.5) computed independently
both give 30°. If they disagreed, the scene would be wrong.

### Layer 2 (the JSON file)

`examples/scenes/hyperbolic_triangle.json` records the values above
verbatim as `"p1": [0.8858, 0, 0]`, etc. The engine never does any
H³-specific reasoning at parse time; it just lays the triangles into
the BVH.

### Where the H³-ness shows up at render

- The walls visually meet at sharp 30° apexes — euclidean intuition,
  given the layout, would predict 60° (because Klein-euclidean angles
  at each vertex still look like 60° to a euclidean protractor laid on
  the *image*; the difference between 30° and 60° here is what the
  metric does to the *light propagation* from the apex to the camera).
- The hyperbolic fog (`media.air.sigma_a` × `arc_H`) attenuates the
  back wall asymmetrically because hyperbolic distance from the camera
  to the back wall is non-uniform across the wall.
- The 5 cells of the {4,3,5} signature do not appear here (that's
  Plan C), but the angle-defect *is* the simplest non-euclidean
  witness.

---

## 7. Worked Example: Plan C ({4,3,5} Honeycomb, 25 Cells)

Goal: a 3D honeycomb where 5 cubes meet along every edge, instead of
the euclidean 4. This is the {4,3,5} regular honeycomb of H³.

### Layer 3 → Layer 2 derivation (per `scene-editor/tools/gen_hyperbolic_honeycomb.py`)

Step 1. Cube dihedral angle = 2π/5 = 72° (the "5 around an edge"
condition). Coxeter relation for a regular cube in H³ with origin at
its center, faces axis-aligned:

```
a² = √5 − 2     ⇒    a = √(√5 − 2) ≈ 0.4859
```

so the base cube vertices are `(±a, ±a, ±a)`, all with
`|V|² = 3a² ≈ 0.708 < 1` — safely inside the Klein ball.

Step 2. Each face is a Klein plane `n·x = c` with `n` an axis vector
and `c = ±a`. Compute outward unit normal from the face vertices,
ensuring the cell center has negative signed distance.

Step 3. Generate neighbor cells by **H³ reflection** (§3.6) across
each face. The Python generator does this in
`reflect_point()` — note the `(1 − k c)` denominator, which is what
makes this a non-euclidean isometry rather than an affine reflection.

Step 4. BFS up to layer 2: 1 base + 6 face-neighbors + 18 second-shell
= 25 cells. Deduplicate by rounded vertex hash to absorb numerical
drift.

### Layer-2 fact made visible at render

Every cell is H³-congruent to every other cell — they all have the
same edge length, same dihedral angle, same volume. But in Klein
coordinates they look exponentially smaller as you walk away from the
origin, because Klein distance saturates at the boundary while H³
distance grows linearly. The receding "infinite hallway" appearance of
the rendered honeycomb is a direct visualization of this disparity.

The "5 cubes per edge" property survives reflection (it's an H³
isometric invariant). You can count cells around any layer-1 edge in
the rendered image and find five — the base cube plus four neighbors
sharing that edge.

---

## 7b. Worked Example: Plan D (H³ Ball as a Quadric)

Goal: render side by side a *true H³ ball* (constant-`d_H` set, intrinsic
H³ object) and a *Klein-euclidean sphere* (a euclidean ball drawn in
Klein coordinates), at progressively non-origin centers, so that the
metric is directly visible.

### The H³-ball-as-quadric derivation

A point `x` is on the boundary of the H³ ball of H³-radius `R_H` at
center `q ∈ B³` iff `d_H(x, q) = R_H`. Apply §3.1 and square:

```
acosh(...) = R_H
⇒  (1 − x·q)² = cosh²(R_H) · (1 − |x|²)(1 − |q|²)
```

Let `K = cosh²(R_H) · (1 − |q|²)`. Then

```
(1 − x·q)² − K (1 − |x|²) = 0
```

Expand `(1 − x·q)² = 1 − 2 q^T x + x^T (q q^T) x`:

```
x^T [q q^T + K · I] x   +   (−2 q)^T x   +   (1 − K)   =   0
```

So the engine's `quadratic equation` shape with coefficients

| | value |
|---|---|
| `A` (3×3 symmetric) | `q q^T + K · I` |
| `b` (3) | `−2 q` |
| `c` | `1 − K` |
| `K` | `cosh²(R_H) · (1 − |q|²)` |

is the *exact* boundary of the H³ ball. Inside H³ this set is a perfect
sphere of H³-radius `R_H`; in the Klein chart it is an oblate ellipsoid
whose minor axis points along `q` (radial direction).

For comparison, the Klein-euclidean "sphere" of euclidean radius
`tanh(R_H)` at the same center has matching extent at `q = 0` (where
the chart is conformal) but stays round in Klein coordinates regardless
of where you put it. So:

- Same primitive Klein appearance at the origin → same visual.
- At `q ≠ 0` the Klein-euclidean sphere is round in chart but
  hyperbolic-elliptic in H³; the true H³ ball is hyperbolic-round but
  chart-elliptic. The two shapes "swap their distortion" between the
  two layers.

### Why this is the cleanest metric witness

Plan A shows angles violating euclidean expectations; Plan C shows
adjacency (5-around-an-edge) violating euclidean possibilities. Plan D
shows the **metric tensor itself** by isolating it: same intrinsic
size, two chart representations, and the disagreement is *exactly* the
metric distortion `g_p` of §3.2.

This section is the honest list of where the implementation is not yet
end-to-end faithful to the metric.

### 8.1 Frame orthogonalization (`maths/frame_geometry.go`)

The geometry-aware frame builder for Klein currently delegates to the
euclidean Cross2 path. This is **exact at the origin** and accumulates
an `O(|p|²)` angular error away from it. The error is invisible for
Lambert / specular reflectors (cosθ-only sensitivity) but would bias
microfacet anisotropy or measured BRDFs near the Klein boundary.

To fix: do Gram-Schmidt under `g_p` (§3.2) instead of euclidean dot
product when constructing the basis. The `Geometry.InnerProduct` hook
is already there; it just isn't called from the frame builder yet.

### 8.2 Shape semantics

Of the existing primitives, only `triangle`, `plane`, and (origin-
centered, axis-aligned) `cuboid` produce shapes whose Layer-2 set
*coincides* with an H³-natural shape (totally-geodesic plane, geodesic
plane, regular cube at origin). All others are **Klein-coordinate
shapes**:

- `sphere` at non-origin position: a hyperbolic ellipsoid, not an H³
  ball.
- `cuboid` not at origin: not an H³ cube; faces aren't all H³ planes
  unless their normal axes pass through the origin.
- `cylinder`, `circle`, quadric / cubic / quartic / implicit /
  polynomial / STL: all Klein-shape, no H³-natural meaning.

If you want a true H³ ball centered at a non-origin point, build it
with `quadratic equation` and the H³ ball equation in Klein
coordinates (a non-trivial quadric, not a sphere).

### 8.3 Boundary behavior

Klein has no boundary in H³ (it's at infinity), but the BVH does
intersect Klein-illegal points if you give them. The generator should
test `|p|² < 1 − ε` for some safety margin. The honeycomb code drops
any reflection that pushes vertices outside `|p|² < 0.999`; this acts
as a "stop growing the layer when it hits the floating-point cliff"
fallback rather than as a principled hyperbolic-volume-bound check.

### 8.4 No Layer-3 authoring DSL

There is no JSON syntax for "place this point at H³ distance 1.6 along
+x". You must derive Layer-2 coordinates yourself. A future
`"position_h3": [...]` field with optional `"distance_h3": ...` would
remove this burden but adds parser work.

### 8.5 Spherical (S³) coordinates

Spherical follows the same three-layer model with Layer 2 = unit
3-sphere in R⁴. The wrap-around behavior (rays leaving the visible
hemisphere come back through the antipode) is implemented in
`Geometry.WrapBeyond`. This document is Klein-focused; the S³
counterparts are the obvious analogues with `cos`/`sin` replacing
`cosh`/`sinh`/`atanh` and "great circle" replacing "chord".

---

## 9. Quick Reference: When You Add a New H³ Scene

1. Write down the **intrinsic specification** at Layer 3 (distances,
   angles, adjacency, symmetry group). Don't draw Klein coordinates yet.
2. Pick the inverse-design formulas from §3 needed to translate that
   spec into Klein vertex/normal data.
3. **Verify with a second formula.** If you derived a Klein placement
   from §3.7 (vertex-side), check it with §3.5 (face-side), or vice
   versa. Two independent computations agreeing is the smoke test.
4. Confirm every Klein coordinate satisfies `|p|² < 1 − ε` with a margin
   that survives downstream reflections / BVH.
5. Emit JSON with the Klein coordinates verbatim. Add comments
   explaining what intrinsic invariant each block enforces — not what
   it "looks like in the picture".
6. Render. If the metric isn't visible, check that `geometry.type =
   "klein"`, the camera type is `"hyperbolic"`, and that the scene
   straddles enough Klein radius (typical: closest objects at
   `|p| ≥ 0.5`, farthest at `|p| ≥ 0.85`) for the metric to dominate
   over euclidean perspective.

---

## 10. Source Files Cross-Reference

| Layer / role | File |
|---|---|
| Geometry interface and Klein impl | `engine/maths/geometry/geometry.go`, `engine/maths/geometry/klein.go` |
| Spherical impl (parallel) | `engine/maths/geometry/spherical.go` |
| Ray geometry tagging | `engine/model/optics/ray.go` |
| Camera with Klein-tagging | `engine/model/camera/camera_hyperbolic.go` |
| Spherical camera | `engine/model/camera/camera_spherical.go` |
| Integrator choke points | `engine/ray_tracing/trace_ray.go` (lines 33, 53, 100, 154) |
| Geometry-aware frame | `engine/maths/frame_geometry.go` |
| JSON schema (`geometry`, `media`) | `engine/controller/parser/schema.go` |
| Honeycomb generator | `scene-editor/tools/gen_hyperbolic_honeycomb.py` |
| Plan A scene | `examples/scenes/hyperbolic_triangle.json` |
| Plan C scene (generated) | `examples/scenes/hyperbolic_honeycomb.json` |
| Plan C generator | `scene-editor/tools/gen_hyperbolic_honeycomb.py` |
| Plan D scene (generated) | `examples/scenes/hyperbolic_ball_compare.json` |
| Plan D generator | `scene-editor/tools/gen_hyperbolic_ball_compare.py` |
| Original gallery scene | `examples/scenes/hyperbolic_gallery.json` |
| Validation chessboard | `examples/scenes/non-euclidean/hyperbolic_chessboard.json` |
| Validation S³ scene | `examples/scenes/non-euclidean/spherical_hopf.json` |
| Design spec | `.agents/spec.md` |
