# Engine Shape Categories, Mathematics, and Input Schemas

> Scope: this document describes only the current code under `engine/`. Studio behavior, example generators, and aliases found only in older documentation are intentionally excluded.
>
> Primary sources: `engine/controller/factory/shapes.go`, the specialized factory files, `engine/model/shape/`, `engine/model/object/intersection.go`, and `engine/ray_tracing/trace_ray.go`.

## 1. Executive Summary

The word "Shape" has three distinct meanings in the Engine and they should not be merged into a single list:

1. **JSON discriminator values**: the values accepted by an object's `shape` field. The factory declares 18 values.
2. **Runtime Shape types**: the Go types that actually implement `shape.Shape`. Several JSON aliases map to the same Go type, while STL expands into many `Triangle` instances.
3. **Internal adapter types**: `BaseShape` supplies default behavior and `BoundedShape` adds clipping to another Shape. Neither is a JSON `shape` value.

Important facts about the current JSON factory:

- Of the 18 declared discriminator values, 17 can be loaded. `plane` reaches its own switch branch but is explicitly rejected.
- `cuboid`/`hypercuboid`, `sphere`/`hypersphere`, and `cylinder`/`finite cylinder` are three pairs of aliases for the same runtime implementations.
- `stl` is an importer, not a runtime Shape type. Every STL facet becomes one `*shape.Triangle`.
- When an object has `bounds`, most single Shapes are wrapped in `*shape.BoundedShape`. `implicit equation` is the main exception because it stores the bounds directly in its own `Range`. Every triangle produced by STL is wrapped separately.
- Non-Euclidean support is not automatic. Klein geometry reuses affine embedded-space intersection, while Spherical geometry requires each Shape to implement `IntersectGeodesic` explicitly.

## 2. Real Shape Category Tables

### 2.1 JSON `shape` Values and Runtime Types

| JSON `shape` | Factory status | Runtime result | Actual category | Dimension constraints and facts |
| --- | --- | --- | --- | --- |
| `cuboid` | Loadable | `*shape.Cuboid` | Axis-aligned box | Uses the current render dimension `D` |
| `hypercuboid` | Loadable | `*shape.Cuboid` | Alias of `cuboid` | Uses `D` |
| `sphere` | Loadable | `*shape.Sphere` | Sphere or hypersphere surface | Uses `D` |
| `hypersphere` | Loadable | `*shape.Sphere` | Alias of `sphere` | Uses `D` |
| `circle` | Loadable | `*shape.Circle` | Finite filled disk, not a circumference | Affine intersection supports `D`; area sampling is 3D only |
| `cylinder` | Loadable | `*shape.FiniteCylinder` | Closed finite cylinder | Alias of `finite cylinder`; affine math supports `D` |
| `finite cylinder` | Loadable | `*shape.FiniteCylinder` | Closed finite cylinder | Uses `D` |
| `triangle` | Loadable | `*shape.Triangle` | Single triangle | **Only safe in 3D** because construction uses a 3D cross product |
| `quadratic equation` | Loadable | `*shape.QuadraticEquation` | Implicit quadratic surface | `A` is always 3 by 3, so practical use is 3D |
| `cubic equation` | Loadable | `*shape.CubicEquation` | Implicit cubic algebraic surface | Uses only `x`, `y`, and `z`; at least three coordinates are required |
| `four-order equation` | Loadable | `*shape.FourOrderEquation` | Implicit quartic algebraic surface | Uses only `x`, `y`, and `z`; at least three coordinates are required |
| `implicit equation` | Loadable | `*shape.ImplicitEquation` | Zero level set of a scalar field | Every current field reads `x`, `y`, and `z`; treat it as 3D |
| `parametric equation` | Loadable | `*shape.ParametricEquation` | Two-parameter surface | The factory explicitly requires $D=3$ |
| `parametric curve` | Loadable | `*shape.ParametricCurve` | Variable-radius swept-sphere tube | The factory explicitly requires $D=3$ |
| `polynomial surface` | Loadable | `*shape.PolynomialSurface` | Sparse arbitrary-degree implicit polynomial | `input_dim` is 1 through 3; affine rays must have at least three dimensions |
| `klein_bottle` | Loadable | `*shape.KleinBottle4D` | Boundary of an epsilon tube around a 4D Klein-bottle surface | The factory explicitly requires $D=4$ |
| `stl` | Loadable | Multiple `*shape.Triangle` instances | Importer, not an independent Shape | Only safe in 3D |
| `plane` | **Explicitly rejected** | No object is created | `*shape.Plane` exists in source | `ParseShape` returns "declared but not implemented" |

If the object contains `bounds`, the result shown above is normally wrapped in `*shape.BoundedShape`. `implicit equation` is the exception: it stores the bounds in `Range` instead.

### 2.2 Runtime Capability Matrix

"Spherical" means explicit great-circle intersection support. Euclidean and Klein geometry both use `IntersectAffine`; Klein compatibility still requires a valid 3D Shape inside the unit-ball model.

| Runtime type | Affine intersection | Spherical great-circle intersection | Tight intrinsic AABB | `SurfaceSampler` |
| --- | --- | --- | --- | --- |
| `Cuboid` | Yes | Yes | Yes | Yes, 3D only |
| `Sphere` | Yes | Yes | Yes | Yes, 3D only |
| `Circle` | Yes | Yes | Yes | Yes, 3D only |
| `FiniteCylinder` | Yes | No | Yes | No |
| `Triangle` | Yes | No | Yes | Yes, 3D only |
| `QuadraticEquation` | Yes | Yes | No; approximately infinite by default | No |
| `CubicEquation` | Yes | No | No; approximately infinite by default | No |
| `FourOrderEquation` | Yes | No | No; approximately infinite by default | No |
| `ImplicitEquation` | Yes | Yes | Yes with a range; otherwise approximately infinite | No |
| `ParametricEquation` | Yes | No | Estimated from sampled patches | No |
| `ParametricCurve` | Yes | No | Estimated from sampled segments | No |
| `PolynomialSurface` | Yes | Yes | No; approximately infinite by default | No |
| `KleinBottle4D` | Yes | No | Yes | No |
| `Plane` | Yes | Yes | No; approximately infinite by default | No |
| `BoundedShape` | Delegates after clipping | Delegates, then checks containment | Uses the external bounds | Only when the bounds fully contain the inner Shape and the inner Shape is sampleable |
| `BaseShape` | Always misses | Always misses | $[-\mathtt{MaxFloat64}/2,+\mathtt{MaxFloat64}/2]^D$ | No |

Consequently, `triangle`, finite cylinders, cubic and quartic equations, parametric surfaces, parametric curves, and `klein_bottle` cannot currently produce hits in a Spherical scene. The 4D `klein_bottle` works only through a 4D **affine/Euclidean** render path. Its name does not make it usable in 3D Klein geometry or on Spherical great-circle paths.

## 3. Common Input Conventions

The schemas below use JSONC to describe structure. They are documentation, not standalone JSON Schema files stored in the repository.

```jsonc
{
  "id": "optional string",
  "material_id": "required non-empty string",
  "shape": "required discriminator",
  "medium_boundary": { /* optional; belongs to the medium system */ },
  "bounds": {                   // optional for most Shapes
    "pmin": [/* D finite numbers */],
    "pmax": [/* D finite numbers */]
  }
}
```

- $D$ comes from `render.dimension`. It defaults to 3 when omitted or non-positive and must be at least 2.
- Scalar and vector elements must be finite numbers. Fields documented as positive are additionally checked with `> 0`.
- Whenever `center` is read through `RequiredVec`, a top-level `position` field is accepted as a compatibility alias. This applies to sphere, circle, cylinder, and `klein_bottle`; it does not apply to cuboid.
- Every bounds axis must satisfy $p_{\min,i}<p_{\max,i}$.
- Bounds clip intersection searches. They do not add cap faces to an open surface and do not change the mathematical equation of the Shape.
- Factories read selected fields from `map[string]interface{}` and do not perform a global unknown-field rejection pass. A misspelled optional field may therefore be silently ignored.

## 4. Mathematics, Processing, and Schema by Shape

### 4.1 Cuboid / Hypercuboid to `shape.Cuboid`

Schema:

```jsonc
{
  "shape": "cuboid | hypercuboid",
  "pmin": [/* D numbers */],
  "pmax": [/* D numbers */],
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // optional
}
```

The represented set satisfies $p_{\min,i}\le x_i\le p_{\max,i}$ on every axis. For the affine ray

$$
x(t)=o+td,
$$

the implementation computes a slab interval on every axis:

$$
t_{\mathrm{near},i}=\frac{p_{\min,i}-o_i}{d_i},
\qquad
t_{\mathrm{far},i}=\frac{p_{\max,i}-o_i}{d_i}.
$$

It intersects all axis intervals. A ray starting inside the box returns the exit root; otherwise it returns the entry root. The normal is $+e_i$ or $-e_i$ for the boundary axis that was hit.

The 3D implementation uses an expanded x/y/z fast path. Other dimensions use the generic per-axis loop. The Spherical path writes a great circle as

$$
\gamma(s)=o\cos s+v\sin s,
$$

solves for candidate values where each coordinate equals a box-face constant, and validates the other coordinates against the box.

The Shape's own `pmin` and `pmax` do **not** pass through the strict ordering validation used by external bounds. Callers should guarantee $p_{\min,i}<p_{\max,i}$ on every axis. In 3D, surface area is

$$
A=2(\Delta x\,\Delta y+\Delta x\,\Delta z+\Delta y\,\Delta z),
$$

and the sampler selects one of the six faces with probability proportional to its area.

### 4.2 Sphere / Hypersphere to `shape.Sphere`

Schema:

```jsonc
{
  "shape": "sphere | hypersphere",
  "center": [/* D numbers; position may be used instead */],
  "r": "positive number",
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // optional
}
```

The surface is

$$
F(x)=\|x-c\|^2-r^2=0.
$$

Substitution of the affine ray gives:

$$
a=d\cdot d,
\qquad
b=2d\cdot(o-c),
\qquad
c_0=\|o-c\|^2-r^2.
$$

The implementation solves $at^2+bt+c_0=0$ and takes the smallest root inside the query range. The normal and AABB are

$$
n(p)=\frac{p-c}{\|p-c\|},
\qquad
[c-(r,\ldots,r),\,c+(r,\ldots,r)].
$$

The Spherical path scans $F(\gamma(s))$ over 2048 fixed segments. It accepts a near-zero value directly or bisects an interval with a sign change. Area and surface sampling are enabled only when the center is 3D, with $A=4\pi r^2$ and uniform sphere sampling.

### 4.3 Circle to `shape.Circle`

The Engine's Circle is a **filled disk**, not a circumference.

```jsonc
{
  "shape": "circle",
  "center": [/* D; position may be used instead */],
  "normal": [/* D, non-zero */],
  "r": "positive number",
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // optional
}
```

Construction normalizes the supplied normal to $n$. The affine path first solves the supporting plane:

$$
t=\frac{n\cdot(c-o)}{n\cdot d}.
$$

If $|n\cdot d|<\mathtt{EPS}$, the ray misses. It then checks

$$
\|o+td-c\|^2\le r^2+\mathtt{EPS}.
$$

The normal is always $n$. The projected AABB radius on axis $i$ is $r\sqrt{1-n_i^2}$.

The Spherical path solves $n\cdot\gamma(s)=n\cdot c$ analytically and then applies the disk-radius test. In 3D, area is $\pi r^2$; the sampler builds a local orthonormal frame and uses $\rho=r\sqrt{u}$ for a uniform disk sample.

### 4.4 Cylinder / Finite Cylinder to `shape.FiniteCylinder`

```jsonc
{
  "shape": "cylinder | finite cylinder",
  "center": [/* D; position may be used instead */],
  "axis": [/* D, non-zero */],
  "r": "positive number",
  "height": "positive number",
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // optional
}
```

Construction normalizes the axis $a$. Let $q=o-c$ and decompose the ray into components perpendicular to the axis:

$$
d_\perp=d-(d\cdot a)a,
\qquad
q_\perp=q-(q\cdot a)a.
$$

Side roots satisfy

$$
\|q_\perp+td_\perp\|^2-r^2=0,
\qquad
|(q+td)\cdot a|\le\frac{h}{2}.
$$

The two caps are disks centered at $c\pm(h/2)a$ with normals $+a$ and $-a$. The smallest valid candidate from the side and both caps is returned.

The side normal is the radial vector after removing its axis component. Cap normals are $+a$ and $-a$. The AABB extent on axis $i$ is:

$$
e_i=\frac{h}{2}|a_i|+r\sqrt{1-a_i^2}.
$$

The type currently has neither `IntersectGeodesic` nor `SurfaceSampler`, so it is not available on Spherical paths and cannot be sampled as a BDPT area light.

### 4.5 Triangle to `shape.Triangle`

```jsonc
{
  "shape": "triangle",
  "p1": [/* D */],
  "p2": [/* D */],
  "p3": [/* D */],
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // optional
}
```

Let $e_1=p_2-p_1$ and $e_2=p_3-p_1$. Intersection uses a Moller-Trumbore-style solution of:

$$
o+td=p_1+ue_1+ve_2.
$$

The accepted domain is $u\ge0$, $v\ge0$, and $u+v\le1$. A hit returns $\mathrm{UV}=(u,v)$, $\partial p/\partial u=e_1$, and $\partial p/\partial v=e_2$. The normal is

$$
n=\frac{e_1\times e_2}{\|e_1\times e_2\|}.
$$

The AABB is the per-axis minimum and maximum of the three vertices. Area is $\|e_1\times e_2\|/2$, and square-root remapping provides uniform barycentric sampling.

Although the factory reads vertices with length $D$, the constructor immediately calls a cross-product function that accepts only 3D vectors. $D\ne3$ is therefore not safe. Degenerate triangles are not rejected by the factory; their normal and area collapse to zero.

### 4.6 Quadratic Equation to `shape.QuadraticEquation`

```jsonc
{
  "shape": "quadratic equation",
  "a": [/* exactly 9 numbers, 3 by 3 row-major */],
  "b": [/* D numbers */],
  "c": "number",
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // strongly recommended
}
```

The surface is

$$
F(x)=x^TAx+b^Tx+c=0.
$$

Substituting $x=o+td$ produces:

$$
\alpha=d^TAd,
\qquad
\beta=o^TAd+d^TAo+b^Td,
\qquad
\gamma=o^TAo+b^To+c.
$$

The real quadratic roots are solved and the nearest root in range is selected. The Spherical path scans the same scalar function along a great circle.

The normal implementation uses $\operatorname{normalize}(2Ax+b)$. This equals the exact gradient $(A+A^T)x+b$ only when $A$ is symmetric. The schema does not enforce symmetry, so callers should provide a symmetric matrix. Because $A$ is fixed at 3 by 3 while $b$ uses $D$, this Shape should be restricted to $D=3$. It has no tight intrinsic AABB, so external bounds are strongly recommended.

### 4.7 Cubic Equation to `shape.CubicEquation`

```jsonc
{
  "shape": "cubic equation",
  "a": [/* exactly 64 numbers */],
  // "A" may be used instead, but a and A cannot both be present.
  // The value may also be an object with flat keys or "i,j,k" keys.
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // recommended
}
```

Let the factor vector be $f=[1,x,y,z]^T$. The mathematical form is:

$$
F(x,y,z)=\sum_{i,j,k=0}^{3}A_{ijk}f_if_jf_k.
$$

Every index is in $[0,3]$. Different tensor permutations may represent the same monomial, so construction merges non-zero entries by their $(x\text{ power},y\text{ power},z\text{ power})$ tuple.

For intersection, the implementation precomputes coefficient tables for powers such as $(o_x+td_x)^k$, accumulates a univariate polynomial of degree at most three, calls the general real polynomial solver, and selects the nearest valid root. The normal is the normalized gradient obtained by differentiating the merged monomials.

Sparse object keys must use either flat indexes `0..63` exclusively or three comma-separated coordinates exclusively. The two styles cannot be mixed. Only the first three coordinates are used. There is no Spherical intersection or tight intrinsic AABB.

### 4.8 Four-Order Equation to `shape.FourOrderEquation`

```jsonc
{
  "shape": "four-order equation",
  "a": [/* exactly 256 numbers */],
  // "A" may be used instead. A sparse object may use all-flat or all-coordinate keys.
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // recommended
}
```

Using the same $f=[1,x,y,z]^T$ factor vector:

$$
F(x,y,z)=\sum_{i,j,k,l=0}^{3}A_{ijkl}f_if_jf_kf_l.
$$

Index permutations are merged into monomials. Substituting the ray produces a univariate polynomial of degree at most four. The implementation solves all real roots and returns the nearest root in range. The normal is the normalized monomial gradient.

Sparse flat indexes range from `0` through `255`. Coordinate keys must contain exactly four coordinates, each in `[0,3]`. This type uses only the first three coordinates and has neither Spherical intersection nor a tight intrinsic AABB. The source and JSON names use `FourOrder` and `four-order`, but the mathematical meaning is quartic.

### 4.9 Implicit Equation to `shape.ImplicitEquation`

Top-level schema:

```jsonc
{
  "shape": "implicit equation",
  "field": { "type": "expr | gyroid | lp_power_sum | metaballs" },

  // These are alternative world-to-local placement forms.
  // transform takes precedence when present.
  "transform": [[/* 4 */], [/* 4 */], [/* 4 */], [/* 4 */]], // optional
  "center": [/* D; optional, default zero */],
  "scale": "positive number | D positive numbers",             // optional, default one
  "basis": [[/* 3 */], [/* 3 */], [/* 3 */]],                  // optional orthonormal row vectors

  "bounds": { "pmin": [/* D */], "pmax": [/* D */] },       // optional, recommended
  "step": "positive number",                                  // optional
  "value_tol": "positive number"                              // optional, default 1e-7
}
```

Every variant represents the local zero level set $F(q)=0$. The local coordinates are formed by the documented world-to-local 4 by 4 transform. In the implementation, rows 1 through 3 are applied to $[1,x,y,z]^T$. The `center`, `scale`, and `basis` form is compiled into the same transform representation. Bounds always remain a world-space AABB.

Field schemas:

```jsonc
// expr
{
  "type": "expr",
  "expr": "string",
  "constants": { "name": 1.0 },
  "gradient": { "x": "string", "y": "string", "z": "string" }
}

// gyroid
{
  "type": "gyroid",
  "frequency": "positive number",
  "offset": "number"
}

// lp_power_sum
{
  "type": "lp_power_sum",
  "power": "positive number",
  "radius": "positive number",
  "gradient_epsilon": "positive number; optional, default 1e-12"
}

// metaballs
{
  "type": "metaballs",
  "k": "positive number",
  "iso": "number",
  "balls": [{ "weight": "number", "center": [/* exactly 3 */] }]
}
```

The corresponding scalar fields are:

- `expr`: a user expression $F(x,y,z)$.
- `gyroid`:

  $$
  F(x,y,z)=
  \sin(fx)\cos(fy)+\sin(fy)\cos(fz)+\sin(fz)\cos(fx)-o.
  $$

- `lp_power_sum`:

  $$
  F(x,y,z)=|x|^p+|y|^p+|z|^p-r.
  $$

- `metaballs`:

  $$
  F(x)=\sum_i w_i\exp\!\left(-k\|x-c_i\|^2\right)-\mathrm{iso}.
  $$

The expression environment provides `x`, `y`, `z`, `pi`, `e`, and the functions `abs`, `sqrt`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`, `sinh`, `cosh`, `tanh`, `exp`, `log`, `log10`, `floor`, `ceil`, `round`, `pow`, `min`, `max`, `clamp`, and `sign`. Constants cannot override reserved names. When an explicit gradient is absent, the factory first attempts symbolic differentiation. If that is unavailable, the Shape falls back to a centered finite-difference normal.

Affine intersection proceeds as follows:

1. If bounds exist, clip the query interval with a box slab test.
2. Scan the resulting interval with `step`. Without an explicit step, a bounded field derives it from the bounds diagonal divided by approximately 512 samples. An unbounded query with an infinite maximum uses 0.02.
3. Scan at most 2048 steps. A value within `value_tol` is accepted immediately; a sign change is refined by bisection with an internal root tolerance of `1e-6`.
4. Compute the normal from the analytic or automatically differentiated gradient when available, otherwise use centered differences with epsilon `1e-5`. Transform the local gradient back to world space with the transpose of the world-to-local linear part.

This scan can miss two crossings inside one step, an even-multiplicity root, or a tangent root whose sampled values never become sufficiently small. High-frequency fields should use tighter bounds and a smaller `step`.

The Spherical implementation has two current limitations. It passes world points directly to `Function` without applying the world-to-local transform. It also finds the first root on the entire arc before checking bounds; if that root lies outside the bounds, it does not continue to search for later roots inside the bounds.

### 4.10 Parametric Equation to `shape.ParametricEquation`

```jsonc
{
  "shape": "parametric equation",
  "surface": { "type": "expr | spherical_harmonic" },
  "u_range": ["min", "max"],                        // optional, default [0,1], increasing
  "v_range": ["min", "max"],                        // optional, default [0,1], increasing
  "center": [/* 3 */],                                // optional
  "scale": "positive number | 3 positive numbers",   // optional
  "samples_u": "positive integer; default 32",
  "samples_v": "positive integer; default 32",
  "newton_max_iter": "positive integer; default 32",
  "newton_tol": "positive number; default 1e-6",
  "derivative_eps": "positive number; default 1e-5",
  "bounds_padding": "non-negative number; default 1e-6",
  "residual_tol": "positive number; default 1e-5",
  "bounds": { "pmin": [/* 3 */], "pmax": [/* 3 */] } // optional outer clip
}
```

Expression surface schema:

```jsonc
{
  "type": "expr",
  "x": "x(u,v) expression",
  "y": "y(u,v) expression",
  "z": "z(u,v) expression",
  "constants": { "name": 1.0 },
  "derivative": {
    "du": { "x": "string", "y": "string", "z": "string" },
    "dv": { "x": "string", "y": "string", "z": "string" }
  }
}
```

Spherical harmonic surface schema:

```jsonc
{
  "type": "spherical_harmonic",
  "terms": [{
    "l": "integer >= 0",
    "m": "integer in [0,l]",
    "weight": "number; default 1",
    "basis": "cos | sin; default cos"
  }]
}
```

The surface is $P(u,v)$. Expression mode supplies its three coordinates directly. When explicit derivatives are absent, the factory first attempts symbolic differentiation and then falls back to finite differences. Placement produces

$$
P_{\mathrm{world}}(u,v)=c+\operatorname{diag}(s)P(u,v).
$$

Spherical harmonic mode evaluates a real harmonic sum

$$
\psi(u,v)=\sum_j w_jY_{l_jm_j}(u,v),
\qquad
r(u,v)=|\psi(u,v)|,
$$

and returns:

$$
P(u,v)
=
r(u,v)
\begin{pmatrix}
\sin u\cos v\\
\sin u\sin v\\
\cos u
\end{pmatrix}.
$$

Intersection proceeds as follows:

1. Split the parameter domain into $\mathtt{samples\_u}\times\mathtt{samples\_v}$ patches.
2. Estimate each patch AABB from nine evaluations: four corners, the center, and four edge midpoints. Add padding and build a patch BVH.
3. When a ray hits a patch AABB, seed a solve with $(t_{\mathrm{near}},u_{\mathrm{center}},v_{\mathrm{center}})$ for the three-equation system $o+td-P(u,v)=0$.
4. Use a Newton Jacobian with columns $[d,-P_u,-P_v]$, plus backtracking line search. Validate $t$, the parameter ranges, and the final residual.
5. Return $(P_u\times P_v)/\|P_u\times P_v\|$ as the normal and normalized parameter-range coordinates as UV.

The factory accepts any positive integer for the sample counts, while the runtime accessors replace values below 2 with the default 32. The effective minimum is therefore 2. The sampled AABB is not an analytic bound; a high-frequency or high-curvature surface may extend outside it. Increase sample counts or padding when necessary. The type has no Spherical great-circle intersection and no area sampler.

### 4.11 Parametric Curve to `shape.ParametricCurve`

```jsonc
{
  "shape": "parametric curve",
  "curve": {
    "type": "expr",
    "x": "x(t)",
    "y": "y(t)",
    "z": "z(t)",
    "radius": "positive number | expression", // r is a compatibility alias
    "constants": { "name": 1.0 },
    "derivative": { "x": "dx/dt", "y": "dy/dt", "z": "dz/dt" }
  },
  "t_range": ["min", "max"],                    // optional, default [0,1]
  "center": [/* 3 */],                            // optional
  "scale": "positive scalar or [s,s,s]",         // optional, must be uniform
  "samples": "positive integer; default 256",
  "refine_iter": "positive integer; default 40",
  "derivative_eps": "positive number; default 1e-5",
  "bounds_padding": "non-negative number; default 1e-6",
  "bounds": { "pmin": [/* 3 */], "pmax": [/* 3 */] } // optional
}
```

This is not a zero-width curve. It is the boundary of a swept-sphere union: for each curve parameter $t$ there is a sphere centered at $C(t)$ with radius $r(t)$. The result is a variable-radius tube with spherical endpoint caps. Top-level scale affects both $C$ and the radius, so non-uniform scale is rejected.

The implementation takes $\mathtt{samples}+1$ uniform curve samples. Adjacent samples form a segment whose AABB uses both endpoints, the midpoint, the maximum sampled radius, the chord sagitta, and padding. It builds a segment BVH, performs a capsule overlap test for candidate segments, and uses golden-section refinement to minimize the earliest ray-entry distance over the spheres along that interval. The globally nearest hit is returned.

The normal is the hit point minus its corresponding sphere center. Tangents prefer explicit or symbolic derivatives and fall back to finite differences. As with parametric surfaces, sample counts below 2 are replaced by the runtime default. A dynamic radius expression is validated as finite and positive only when it is evaluated. The type has neither Spherical great-circle intersection nor area sampling.

### 4.12 Polynomial Surface to `shape.PolynomialSurface`

```jsonc
{
  "shape": "polynomial surface",
  "input_dim": "integer in [1,3]",
  "coefficients": {
    "format": "hash | coo",                           // optional, default hash
    "shape": [/* input_dim or input_dim+1 positive integers */], // optional
    "terms": [{
      "index": [/* input_dim or input_dim+1 non-negative integers */],
      "value": "number"
    }]
  },
  "transform": [[/* 4 */], [/* 4 */], [/* 4 */], [/* 4 */]], // optional; flat 16 also accepted
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // strongly recommended
}
```

The `coefficients` wrapper is optional; `format`, `shape`, and `terms` may be placed directly on the object. The current parser does **not** require or read a `degree` field. Degree is inferred as the largest total exponent among all terms. Legacy `mode` values other than `implicit` and every `explicit_axis` field are explicitly rejected.

Without an output axis, the polynomial is:

$$
F(q)=\sum_e c_e\prod_i q_i^{e_i}.
$$

When the index rank is $\mathtt{input\_dim}+1$, the first index is an output channel. Shape intersection uses output channel 0 only. Other channels are stored but do not participate in geometry intersection.

The transform is world-to-local. The local ray origin includes translation, while the local direction uses only the linear part. Every term expands $(q_{0,i}+tq_{d,i})^{e_i}$; polynomial convolution accumulates the final univariate ray polynomial, whose real roots are solved before the nearest valid root is selected. The local polynomial gradient is mapped back with the transpose of the world-to-local linear part.

The Spherical path scans $F(\operatorname{local}(\gamma(s)))$ along a great circle. The Shape's intrinsic AABB is approximately infinite, so external bounds are strongly recommended in practical scenes.

### 4.13 Klein Bottle 4D to `shape.KleinBottle4D`

```jsonc
{
  "shape": "klein_bottle",
  "center": [/* exactly 4; position may be used instead */],
  "r_major": "positive number",
  "r_minor": "positive number",
  "thickness": "positive number",
  "bounds": { "pmin": [/* 4 */], "pmax": [/* 4 */] } // optional
}
```

The additional constraint is $r_{\mathrm{major}}>r_{\mathrm{minor}}$, and `render.dimension` must be 4. The base two-dimensional surface is embedded in $\mathbb{R}^4$ as:

$$
S(u,v)
=
\begin{pmatrix}
(R+r\cos v)\cos u\\
(R+r\cos v)\sin u\\
r\sin v\cos(u/2)\\
r\sin v\sin(u/2)
\end{pmatrix}.
$$

A two-dimensional surface has codimension two in 4D, so a one-dimensional ray almost never intersects it directly. The rendered geometry is therefore the epsilon-tube boundary:

$$
F(p)=\operatorname{dist}(p,S)-\tau=0,
$$

where $\tau$ is `thickness`.

Closest-point evaluation starts with a $16\times8$ parameter seed grid, retains the eight nearest seeds, and refines $(u,v)$ with a first-order least-squares/Newton-style iteration and line search. The ray is first clipped to an analytic AABB, then sphere-traced for at most 128 iterations. The step is approximately $0.75|\mathrm{SDF}|$ and never smaller than $\max(10^{-6},0.02\tau)$. A near-zero value is accepted directly; a sign change is refined by bisection.

The normal is

$$
n(p)=\frac{p-S(u^*,v^*)}{\|p-S(u^*,v^*)\|}.
$$

UV contains the optimized parameters after applying the Klein-bottle twist and wrapping rules.

Because the SDF depends on numerical closest-point optimization, it is not guaranteed to be a globally exact distance field. Thin tubes and regions with competing nearby surface points can be sensitive to seed density and marching parameters. These tuning fields are not exposed in JSON. The type does not implement `IntersectGeodesic`, so it cannot participate in Spherical great-circle tracing.

### 4.14 STL to Multiple `shape.Triangle` Instances

```jsonc
{
  "shape": "stl",
  "file": "path string",
  "center": [/* D numbers */],
  "z_dir": [/* D numbers */],
  "x_dir": [/* D numbers */],
  "scale": [/* D numbers */],
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // optional
}
```

The parser opens the file and treats it as ASCII when its first line begins with the literal `solid`. It scans every `vertex` line and creates one Triangle for every group of three vertices. Otherwise it reads binary STL: an 80-byte header, the facet count, and each 50-byte facet record. STL normals are ignored; normals are recomputed from transformed triangle vertices.

The transform columns are $x_{\mathrm{dir}}$, $(z_{\mathrm{dir}}\times x_{\mathrm{dir}})/\|z_{\mathrm{dir}}\times x_{\mathrm{dir}}\|$, and $z_{\mathrm{dir}}$, each multiplied by its corresponding scale and followed by center translation. The x and z directions are individually normalized, but the factory does not validate non-zero input, mutual orthogonality, handedness, or positive/non-zero scale. Callers should provide a valid orthonormal frame explicitly.

Although input vector lengths are checked against $D$, the transformation matrix and output vertices are fixed at 3D, and Triangle itself is safe only in 3D. STL must therefore be used with $D=3$. The simple `solid` detection can misclassify a binary STL whose header starts with that word.

### 4.15 Plane to `shape.Plane` (Not JSON-Loadable)

The source type represents

$$
F(x)=A^Tx+b=0.
$$

Its affine intersection is:

$$
t=-\frac{A\cdot o+b}{A\cdot d}.
$$

The normal is $A/\|A\|$. Its Spherical implementation scans the same scalar function along a great circle.

The Go fields correspond conceptually to:

```jsonc
{
  "shape": "plane",
  "A": [/* D; callers should ensure non-zero */],
  "b": "number"
}
```

This is source-structure documentation, **not a currently valid input schema**. The factory returns an error immediately after matching `plane`; it never reads A or b and never constructs the type.

## 5. Internal Adapters and the Shared Intersection Contract

### 5.1 `shape.Shape`

The interface contains a name, affine intersection, geometry-aware geodesic intersection, normal evaluation, and AABB construction. `SurfaceInteraction` may carry:

- `Distance`: an affine parameter or the distance chosen by the implementation;
- `ArcLength`: arc length for Spherical hits;
- `Point`, `GeometricNormal`, and `ShadingNormal`;
- `UV`, `DPDU`, and `DPDV`;
- `PrimitiveID`.

Every intersection must respect the closed interval $[\mathtt{Min},\mathtt{Max}]$ stored in `IntersectOptions.Range`. Once ObjectTree finds a closer hit, it tightens `Max` to improve BVH pruning.

### 5.2 `BaseShape`

`BaseShape` is an embedded default implementation, not renderable geometry. Both intersection methods always miss, its normal method performs no calculation, and its AABB uses very large finite values instead of infinity to avoid overflow in downstream calculations. Any Shape that does not override a method inherits this behavior.

### 5.3 `BoundedShape`

The affine path first clips the query range with a bounds slab test and then calls the inner Shape. This allows the inner Shape to search directly for its first root inside the box.

The Spherical path behaves differently: it asks the inner Shape for the first root over the complete query interval and checks whether that hit point lies in the box afterward. If the first root lies outside, it returns a miss and does not search for a later root inside the box.

Its AABB is always the external bounds. Area sampling is forwarded only when the bounds completely contain the inner Shape's intrinsic AABB. If the bounds truly clip the inner surface, `SurfaceArea()` returns zero because the wrapper does not recompute the clipped area or PDF.

## 6. How Scene Geometry Selects an Intersection Path

| Scene geometry | Ray path | Shape requirement |
| --- | --- | --- |
| Euclidean | Use `EmbeddedRay`, ObjectTree BVH, and `IntersectAffine` | Any type with valid affine intersection |
| Klein / hyperbolic | Geodesics are Euclidean chords inside the Klein ball, so the Engine still uses the BVH and `IntersectAffine`. After the hit, it converts the ambient gradient to a Klein intrinsic normal and converts embedded $t$ to arc length | Valid 3D affine math and AABB |
| Spherical | ObjectTree linearly scans all objects and calls `IntersectGeodesic`; each current segment has a parameter limit of $\pi$ | The Shape must explicitly override `IntersectGeodesic` |

The common Spherical scalar root finder uses:

$$
\gamma(s)=p\cos s+v\sin s.
$$

Here $v$ is the input direction projected into the sphere tangent space and normalized. The root finder scans 2048 segments, then bisects with a value threshold of $10^{-7}$ and an interval threshold of $10^{-8}$. Like the implicit affine scanner, sign-based scanning can miss tangent roots and even-multiplicity roots.

## 7. Bounds, BVH, Area Lights, and Performance

### 7.1 Shapes That Strongly Benefit from Bounds

`QuadraticEquation`, `CubicEquation`, `FourOrderEquation`, an unbounded `ImplicitEquation`, `PolynomialSurface`, and `Plane` have no tight intrinsic AABB. They inherit an approximately infinite box. They can still enter the BVH, but spatial partitioning and pruning quality become poor. For infinite or open algebraic surfaces, bounds are important both for performance and for controlling the root-search interval.

Parametric surfaces and curves have internal sampled BVHs, but their boxes are discrete estimates. Too few samples or insufficient `bounds_padding` can cause missed intersections rather than merely lower performance.

### 7.2 Area Sampling Support

Only 3D `Cuboid`, `Sphere`, `Circle`, and `Triangle` implement a useful `SurfaceSampler`. BDPT area-light collection includes only emissive objects whose `SurfaceArea()` is positive. Therefore:

- An emissive cylinder, parametric surface, implicit surface, or algebraic surface can emit when hit by a camera or transport path, but it cannot be selected as a finite sampleable BDPT area light.
- A sampleable Shape that is genuinely clipped by `BoundedShape` reports zero area and is excluded from area-light collection.
- BDPT additionally requires 3D Euclidean geometry.

## 8. Current Risks and Common Misreadings

| Topic | Current fact | Recommendation |
| --- | --- | --- |
| Plane | The type exists, but the JSON factory explicitly rejects it | Do not use it in scene JSON without adding a factory implementation |
| Triangle and STL dimensions | The factory does not reject non-3D input, but the cross products and transforms are fixed at 3D | Use only with $D=3$ |
| Quadratic dimension | $A$ is fixed at 3 by 3 while $b$ is read with length $D$ | Use only with $D=3$ |
| Quadratic normal | The implementation uses $2Ax+b$ | Require symmetric $A$ |
| Implicit dimension | Fields and the 4 by 4 transform implementation are built around $x$, $y$, and $z$ | Use only with $D=3$ |
| Implicit Spherical transform | The great-circle path does not apply the world-to-local transform | Do not rely on this combination with a non-identity transform |
| Spherical bounds | The first root is found before containment is checked | A later valid root inside the box may be missed |
| Cubic and quartic coefficients | Tensor indexes select factors from $[1,x,y,z]$ | Interpret indexes as factor selectors, not a conventional exponent tensor |
| Polynomial degree | There is no current `degree` input field | Degree is inferred only from term indexes |
| Polynomial output axis | Multiple channels can be stored, but intersection uses channel 0 only | Put geometric coefficients in output channel 0 |
| Parametric AABB | It is a discrete estimate rather than an analytic bound | Increase samples and padding for high-frequency or high-curvature geometry |
| STL frame | Non-zero vectors, orthogonality, handedness, and positive scale are not validated | Validate and orthonormalize the frame before loading |
| Unknown JSON fields | They are usually ignored | Add strict validation upstream or test-load every scene |

## 9. Sources of Truth for Maintenance

When adding or changing a Shape, inspect at least these locations:

1. `engine/controller/factory/shapes.go` for discriminator values, dispatch, basic fields, and bounds wrapping.
2. The corresponding `engine/controller/factory/*.go` files for complex field, surface, curve, and coefficient schemas.
3. `engine/model/shape/base.go` for the Shape and SurfaceSampler contracts.
4. The corresponding `engine/model/shape/*.go` implementation for mathematics, intersection, normals, AABB behavior, and defaults.
5. `engine/model/object/intersection.go` and `engine/ray_tracing/trace_ray.go` for actual Euclidean, Klein, and Spherical dispatch.
6. `engine/ray_tracing/bdpt.go` for area-sampling and BDPT restrictions.

Adding a Go type does not make it scene-JSON-loadable; a factory branch is also required. Implementing only `IntersectAffine` does not make it available in Spherical geometry; a correct `IntersectGeodesic` implementation is required as well.
