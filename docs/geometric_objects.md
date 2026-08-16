# Geometric Objects

> Scope: this document describes only the current code under `engine/`. Studio behavior, example generators, and aliases found only in older documentation are intentionally excluded.
>

## 1. Real Shape Category Tables

### 1.1 Mathematical and Ray-Tracing Summary

| Geometry name | Mathematical description | Parameter types | Ray-tracing computation |
| --- | --- | --- | --- |
| Triangle | $T=\{p_1+u(p_2-p_1)+v(p_3-p_1)\mid u,v\ge0,\ u+v\le1\}$ | $p_1,p_2,p_3\in\mathbb{R}^3$ | Moller-Trumbore barycentric intersection |
| Circle | $D(c,n,r)=\{x\mid n\cdot(x-c)=0,\ \|x-c\|\le r\}$ | $c,n\in\mathbb{R}^D$, $\|n\|>0$, $r>0$ | Supporting-plane intersection followed by a radial containment test; analytic great-circle plane candidates |
| Sphere, Hypersphere | $S^{D-1}=\{x\in\mathbb{R}^D\mid\|x-c\|^2=r^2\}$ | $c\in\mathbb{R}^D$, $r\in\mathbb{R}_{>0}$ | Quadratic substitution along an affine ray; sampled and bisected scalar roots along a great circle |
| Axis-aligned Cuboid, Hyper-Cuboid | $C=\{x\in\mathbb{R}^D\mid p_{\min,i}\le x_i\le p_{\max,i}\}$ | $p_{\min},p_{\max}\in\mathbb{R}^D$ with $p_{\min,i}<p_{\max,i}$ | Per-axis slab intervals for affine rays; analytic face candidates for spherical great circles |
| Plane, hyperplane | $H=\{x\in\mathbb{R}^D\mid A^Tx+b=0\}$ | $A\in\mathbb{R}^D\setminus\{0\}$, $b\in\mathbb{R}$ | One linear affine root; scalar root search along a great circle; JSON factory currently rejects it |
| Quadratic Surface | $F(x)=x^TAx+b^Tx+c=0$ | $A\in\mathbb{R}^{3\times3}$, $b\in\mathbb{R}^3$, $c\in\mathbb{R}$; symmetric $A$ recommended | Ray substitution produces a polynomial of degree at most two; real-root selection |
| Cubic Algebraic Surface | $F(x,y,z)=\sum\limits_{i,j,k=0}^3A_{ijk}f_if_jf_k=0$, $f=(1,x,y,z)$ | $A\in\mathbb{R}^{4\times4\times4}$, dense or sparse | Tensor entries are merged into monomials; ray substitution produces a cubic polynomial |
| Four-order Surface | $F(x,y,z)=\sum\limits_{i,j,k,l=0}^3A_{ijkl}f_if_jf_kf_l=0$ | $A\in\mathbb{R}^{4\times4\times4\times4}$, dense or sparse | Monomial merging followed by quartic ray-polynomial solution |
| Polynomial Surface | $F(q)=\sum\limits_e c_e\prod\limits_iq_i^{e_i}=0$ | $1\le d\le3$, $e\in\mathbb{N}_0^d$, $c_e\in\mathbb{R}$, optional transform | Exact expansion into a univariate ray polynomial, real-root solution, and transformed gradient |
| Implicit Equation | $S=\{x\in\mathbb{R}^3\mid F(Tx)=0\}$ | Scalar field $F:\mathbb{R}^3\to\mathbb{R}$, world-to-local transform $T$, optional bounds and numerical tolerances | Bounded interval scan, near-zero/sign-change detection, bisection, and gradient normal evaluation |
| Parametric Surface | $S=\{P(u,v)\in\mathbb{R}^3\mid(u,v)\in U\times V\}$ | $P:U\times V\to\mathbb{R}^3$, parameter intervals, derivatives, sampling and Newton tolerances | Patch BVH followed by a three-variable Newton solve of $o+td=P(u,v)$ |
| Parametric Curve | $S=\partial\bigcup\limits_{t\in I}B(C(t),r(t))$ | $C:I\to\mathbb{R}^3$, $r:I\to\mathbb{R}_{>0}$, derivative and sampling controls | Segment BVH, capsule overlap, and golden-section refinement of the earliest swept-sphere entry |
| 4D Klein-bottle tube | $S_\tau=\{p\in\mathbb{R}^4\mid\operatorname{dist}(p,S)=\tau\}$ | $c\in\mathbb{R}^4$, $R>r>0$, $\tau>0$ | AABB clipping, numerical closest-point optimization on $S(u,v)$, and sphere tracing with bisection |
| Triangulated Surface Mesh | $M=\bigcup\limits_{j=1}^NT_j$ | File path and affine frame $(c,x_{\mathrm{dir}},z_{\mathrm{dir}},s)\in\mathbb{R}^3$ | ASCII/binary import expands every facet into a runtime `Triangle`; normal triangle/BVH intersection follows |
| Finite Cylinder | $\partial\{x\mid\|(x-c)-[(x-c)\cdot a]a\|\le r,\ \lvert(x-c)\cdot a\rvert\le h/2\}$ | $c,a\in\mathbb{R}^D$, $\|a\|>0$, $r,h>0$ | Quadratic side roots plus two cap-plane disk tests; nearest valid candidate |


The table lists mathematical geometry, not only factory strings. The word "Shape" has three distinct meanings in the Engine:

1. **JSON discriminator values** are accepted by an object's `shape` field. The factory declares 18 values, including aliases and the rejected `plane` branch.
2. **Runtime Shape types** are the Go types that implement `shape.Shape`. Several JSON aliases map to one Go type, while STL expands into many `Triangle` instances.
3. **Internal adapter types** include `BaseShape`, which supplies default behavior, and `BoundedShape`, which clips another Shape. Neither is a JSON geometry category.

### 1.2 Capability Matrix

| Geometry name | Runtime type | Affine intersection | Spherical great-circle intersection | Intrinsic AABB | Surface measure | Area sampling and PDF | Differential geometry | Numerical, algebraic, and optimization capabilities |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Triangle | `Triangle` | Moller-Trumbore barycentric solve | No | Exact | $A=\|e_1\times e_2\|/2$ in 3D | Uniform square-root barycentric sampling, $p_A=1/A$ | UV, $\partial p/\partial u=e_1$, $\partial p/\partial v=e_2$, constant normal | Determinant/barycentric solve |
| Circle | `Circle` | Plane root plus radial test | Analytic plane-coordinate candidates | Exact projected box | $A=\pi r^2$ in 3D | Polar square-root sampling, $p_A=1/A$ | Constant normal and local orthonormal frame | Linear plane root; analytic trigonometric candidates |
| Sphere, Hypersphere | `Sphere` | Quadratic | Shared scalar great-circle scan | Exact | $A=4\pi r^2$ in 3D | Uniform sphere sampling, $p_A=1/A$ | Radial unit normal | Real quadratic roots; great-circle scan and bisection |
| Axis-aligned Cuboid, Hyper-Cuboid | `Cuboid` | Yes | Analytic face-coordinate candidates | Exact | $A=2(d_xd_y+d_xd_z+d_yd_z)$ in 3D | Face-area sampling, $p_A=1/A$ | Piecewise-constant face normal | Slab clipping; generic-$D$ and optimized 3D paths |
| Plane, hyperplane | `Plane` | Linear | Shared scalar great-circle scan | Approximately infinite | Not exposed | No | Constant normal $A/\|A\|$ | Linear affine root; great-circle scan despite analytic trigonometric form |
| Quadratic Surface | `QuadraticEquation` | Degree-at-most-two ray polynomial | Shared scalar great-circle scan | Approximately infinite | Not exposed | No | Implemented normal $\operatorname{normalize}(2Ax+b)$ | Quadratic real roots; great-circle scan and bisection |
| Cubic Algebraic Surface | `CubicEquation` | Cubic ray polynomial | No | Approximately infinite | Not exposed | No | Analytic merged-monomial gradient | Tensor-to-monomial reduction and general real-polynomial roots |
| Four-order Surface | `FourOrderEquation` | Quartic ray polynomial | No | Approximately infinite | Not exposed | No | Analytic merged-monomial gradient | Tensor-to-monomial reduction and general real-polynomial roots |
| Polynomial Surface | `PolynomialSurface` | Exact univariate ray-polynomial expansion | Shared scalar great-circle scan | Approximately infinite | Not exposed | No | Sparse analytic gradient transformed by $L^T$ | Binomial expansion, polynomial convolution, and general real roots |
| Implicit Equation | `ImplicitEquation` | Deterministic field scan | Shared scalar great-circle scan | Exact with range; otherwise approximately infinite | Not exposed | No surface sampler; deterministic ray-field samples | Explicit, symbolic, or centered finite-difference gradient | AABB clipping, adaptive default step, sign-change detection, and bisection |
| Parametric Surface | `ParametricEquation` | Patch candidates plus Newton solve | No | Estimated from sampled patches | Not exposed | No area sampler; nine deterministic samples per patch | $P_u$, $P_v$, UV, and $P_u\times P_v$ normal | Patch BVH, three-variable Newton iteration, and backtracking |
| Parametric Curve | `ParametricCurve` | Swept-sphere envelope search | No | Estimated from sampled segments | Not exposed | No area sampler; `samples + 1` spine samples | Spine tangent and selected-sphere radial normal | Segment BVH, capsule rejection, and golden-section refinement |
| 4D Klein-bottle tube | `KleinBottle4D` | Distance-field marching | No | Exact analytic box | Not exposed | No; fixed $16\times8$ closest-point seed grid | Optimized $(u,v)$ and offset normal | Multi-seed least-squares/Newton refinement, line search, sphere tracing, and bisection |
| Triangulated Surface Mesh | Per-facet `Triangle` | Per-facet triangle solve | No | Exact per facet | Exact per facet; no mesh aggregate | Per-facet uniform sampling; no mesh-level distribution | Inherited triangle UV, derivatives, and normal | ASCII/binary parsing, affine frame transform, and ObjectTree BVH |
| Finite Cylinder | `FiniteCylinder` | Quadratic side plus two caps | No | Exact projected box | Not exposed | No | Radial side normal and constant cap normals | Perpendicular decomposition, side quadratic, and cap-plane tests |

#### Internal Adapter Capabilities

| Adapter | Affine intersection | Spherical great-circle intersection | AABB | Surface measure and sampling | Differential geometry | Numerical behavior |
| --- | --- | --- | --- | --- | --- | --- |
| `BoundedShape` | Delegates after slab clipping | Delegates, then checks containment | Uses external bounds | Forwards only when bounds contain the complete inner Shape | Delegates | Interval clipping and post-hit containment |
| `BaseShape` | Always misses | Always misses | $[-\mathtt{MaxFloat64}/2,+\mathtt{MaxFloat64}/2]^D$ | Zero area; no sampling | None | Default no-op behavior |

"Spherical" means explicit great-circle intersection support. Euclidean and Klein geometry both use `IntersectAffine`; Klein compatibility still requires a valid 3D Shape inside the unit-ball model. Consequently, `triangle`, finite cylinders, cubic and quartic equations, parametric surfaces, parametric curves, and `klein_bottle` cannot currently produce hits in a Spherical scene. The 4D `klein_bottle` works only through a 4D **affine/Euclidean** render path. Its name does not make it usable in 3D Klein geometry or on Spherical great-circle paths.

## Triangle

### Mathematical Definition

For three vertices $p_1,p_2,p_3\in\mathbb{R}^3$, let $e_1=p_2-p_1$ and $e_2=p_3-p_1$. The filled triangle is the 2-simplex

$$
T=\left\{p_1+ue_1+ve_2\mid u\ge0,\ v\ge0,\ u+v\le1\right\}.
$$

For non-collinear vertices, its normal and area are

$$
n=\frac{e_1\times e_2}{\|e_1\times e_2\|},
\qquad
A=\frac12\|e_1\times e_2\|.
$$

It is compact and convex, and its exact AABB is the coordinate-wise minimum and maximum of its vertices.

### Ray Intersection

Intersection uses a Moller-Trumbore-style solution of

$$
o+td=p_1+ue_1+ve_2.
$$

The accepted domain is $u\ge0$, $v\ge0$, and $u+v\le1$. A hit returns $\mathrm{UV}=(u,v)$, $\partial p/\partial u=e_1$, and $\partial p/\partial v=e_2$. Square-root remapping provides uniform barycentric surface sampling.

Although the factory reads vertices with length $D$, the constructor immediately calls a cross-product function that accepts only 3D vectors. $D\ne3$ is therefore not safe. Degenerate triangles are not rejected by the factory; their normal and area collapse to zero.

### Parameters and Schema

- $p_1,p_2,p_3\in\mathbb{R}^3$ are the vertices.
- The non-degeneracy condition is $\|(p_2-p_1)\times(p_3-p_1)\|>0$.

```jsonc
{
  "shape": "triangle",
  "p1": [/* exactly 3 finite numbers in safe use */],
  "p2": [/* exactly 3 finite numbers in safe use */],
  "p3": [/* exactly 3 finite numbers in safe use */],
  "bounds": { "pmin": [/* 3 */], "pmax": [/* 3 */] } // optional
}
```

### Uniform Barycentric Sampling

Surface sampling exists only in 3D. Given $(u,v)\in[0,1]^2$, the implementation sets

$$
s=\sqrt{u},
\qquad
\alpha=1-s,
\qquad
\beta=vs,
$$

and returns

$$
x=p_1+\alpha e_1+\beta e_2.
$$

The corresponding vertex barycentric weights are

$$
(\lambda_1,\lambda_2,\lambda_3)
=(1-\alpha-\beta,\alpha,\beta),
$$

which remain non-negative and sum to one. The square-root map makes the density uniform over the triangle:

$$
p_A(x)=\frac{1}{A}
=\frac{2}{\|e_1\times e_2\|}.
$$

## Circle

### Mathematical Definition

The Engine's Circle is a **filled disk**, not a circumference.

For center $c$, unit normal $n$, and radius $r$, the disk is

$$
D(c,n,r)=\left\{x\in\mathbb{R}^D\mid
n\cdot(x-c)=0,\ \|x-c\|\le r\right\}.
$$

It is a compact convex subset of its supporting hyperplane. Its boundary is a $(D-2)$-sphere. In 3D, its area is $\pi r^2$.

### Ray Intersection

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

### Parameters and Schema

- $c,n\in\mathbb{R}^D$, with $n\ne0$ before normalization.
- $r\in\mathbb{R}_{>0}$.
- `position` is accepted as a compatibility alias for `center`.

```jsonc
{
  "shape": "circle",
  "center": [/* D; position may be used instead */],
  "normal": [/* D finite numbers, non-zero */],
  "r": "positive number",
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // optional
}
```

### Spherical Great-Circle Geometry

With $\gamma(s)=o\cos s+v\sin s$, the supporting-hyperplane equation becomes

$$
(n\cdot o)\cos s+(n\cdot v)\sin s=n\cdot c.
$$

The implementation solves this trigonometric linear-coordinate equation analytically over the permitted arc interval, evaluates every candidate point, and retains only candidates satisfying $\|\gamma(s)-c\|^2\le r^2+\mathtt{EPS}$.

### Uniform Disk Sampling

In 3D, the sampler builds an orthonormal frame $(e_1,e_2,n)$ and maps $(u,v)\in[0,1]^2$ through

$$
\rho=r\sqrt{u},
\qquad
\phi=2\pi v,
$$

$$
x=c+\rho(\cos\phi\,e_1+\sin\phi\,e_2).
$$

The square root cancels the polar-coordinate Jacobian, giving the constant area density

$$
p_A(x)=\frac{1}{\pi r^2}.
$$

## Sphere, Hypersphere

### Mathematical Definition

A hypersphere is the codimension-one level set

$$
S^{D-1}(c,r)=\left\{x\in\mathbb{R}^D\mid
F(x)=\|x-c\|^2-r^2=0\right\}.
$$

It is compact, smooth for $r>0$, rotationally symmetric, and has exact AABB

$$
[c-(r,\ldots,r),\ c+(r,\ldots,r)].
$$

Its outward unit normal is

$$
n(p)=\frac{p-c}{\|p-c\|}.
$$

In three dimensions its surface area is $A=4\pi r^2$.

### Ray Intersection

Substitution of $x(t)=o+td$ gives

$$
a=d\cdot d,
\qquad
b=2d\cdot(o-c),
\qquad
c_0=\|o-c\|^2-r^2.
$$

The implementation solves $at^2+bt+c_0=0$ and takes the smallest root inside the query range.

The Spherical path scans $F(\gamma(s))$ over 2048 fixed segments. It accepts a near-zero value directly or bisects an interval with a sign change. Area and surface sampling are enabled only when the center is 3D, with $A=4\pi r^2$ and uniform sphere sampling.

### Parameters and Schema

- $c\in\mathbb{R}^D$ is the center.
- $r\in\mathbb{R}_{>0}$ is the radius.
- `position` is accepted as a compatibility alias for `center`.

```jsonc
{
  "shape": "sphere | hypersphere",
  "center": [/* D finite numbers; position may be used instead */],
  "r": "positive number",
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // optional
}
```

### Spherical Great-Circle Geometry

The Spherical path evaluates the same level-set function on

$$
\gamma(s)=o\cos s+v\sin s
$$

and seeks roots of

$$
G(s)=\|\gamma(s)-c\|^2-r^2.
$$

It uses the shared scalar great-circle solver: 2048 scan segments, direct acceptance of near-zero samples, and bisection across sign-changing intervals. This numerical scan can miss an even-multiplicity or tangent root when no sampled value is sufficiently close to zero.

### Uniform Surface Sampling

Uniform area sampling is implemented only for a 3D sphere. For $(u,v)\in[0,1]^2$,

$$
z=1-2u,
\qquad
\phi=2\pi v,
\qquad
\rho=\sqrt{1-z^2},
$$

$$
n=(\rho\cos\phi,\rho\sin\phi,z),
\qquad
x=c+rn.
$$

Because $z$ and $\phi$ are uniform in their natural area coordinates, the area PDF is

$$
p_A(x)=\frac{1}{4\pi r^2}.
$$

## Axis-aligned Cuboid, Hyper-Cuboid

### Mathematical Definition

An axis-aligned hyperrectangle is the compact convex set

$$
C=\left\{x\in\mathbb{R}^D\mid
p_{\min,i}\le x_i\le p_{\max,i},\ i=1,\ldots,D\right\}.
$$

Its boundary is the union of $2D$ axis-aligned $(D-1)$-dimensional faces. It has a finite exact AABB equal to $[p_{\min},p_{\max}]$ and piecewise-constant outward normals $\pm e_i$. In three dimensions, its surface area is

$$
A=2(\Delta x\,\Delta y+\Delta x\,\Delta z+\Delta y\,\Delta z).
$$

### Ray Intersection

For the affine ray

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

The Shape's own `pmin` and `pmax` do **not** pass through the strict ordering validation used by external bounds. Callers should guarantee $p_{\min,i}<p_{\max,i}$ on every axis. In 3D, the sampler selects one of the six faces with probability proportional to its area.

### Parameters and Schema

- $p_{\min},p_{\max}\in\mathbb{R}^D$ are the opposite AABB corners.
- The intended invariant is $p_{\min,i}<p_{\max,i}$ for every axis.
- `cuboid` and `hypercuboid` are equivalent factory discriminators.

```jsonc
{
  "shape": "cuboid | hypercuboid",
  "pmin": [/* D finite numbers */],
  "pmax": [/* D finite numbers */],
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // optional outer clip
}
```

### Spherical Great-Circle Geometry

For a Spherical scene, the implementation first constructs the unit tangent $v$ at the ray origin $o$, so that

$$
\|o\|=\|v\|=1,
\qquad
o\cdot v=0,
$$

and parameterizes the great circle by

$$
\gamma(s)=o\cos s+v\sin s.
$$

For every axis $i$ and face coordinate $b\in\{p_{\min,i},p_{\max,i}\}$, candidate arc lengths solve

$$
o_i\cos s+v_i\sin s=b.
$$

The remaining coordinates are then tested against their slab intervals. The nearest accepted $s$ is stored as both distance and arc length. In Klein geometry there is no Shape-specific curved solver: the engine traces the Euclidean chord in the Klein ball and uses the ordinary affine slab algorithm.

### Surface-Area Sampling

Surface sampling exists only in 3D. If the side lengths are $(d_x,d_y,d_z)$, the three distinct face areas are

$$
A_x=d_yd_z,
\qquad
A_y=d_xd_z,
\qquad
A_z=d_xd_y,
$$

and the total area is $A=2(A_x+A_y+A_z)$. Each signed face is selected with probability $A_i/A$. The residual of the face-selection sample supplies one in-face coordinate; the other is decorrelated with

$$
b=\operatorname{frac}(v+\varphi^{-1}u),
\qquad
\varphi^{-1}\approx0.61803398875.
$$

The resulting density with respect to surface area is constant:

$$
p_A(x)=\frac{1}{A}.
$$

## Plane, hyperplane

### Mathematical Definition

For a non-zero covector $A\in\mathbb{R}^D$ and scalar $b\in\mathbb{R}$, an affine hyperplane is

$$
H=\left\{x\in\mathbb{R}^D\mid F(x)=A^Tx+b=0\right\}.
$$

It is an unbounded, flat, codimension-one affine subspace with constant unit normal $A/\|A\|$. It has no finite intrinsic AABB.

### Ray Intersection

Its affine intersection is

$$
t=-\frac{A\cdot o+b}{A\cdot d}.
$$

The normal is $A/\|A\|$. Its Spherical implementation scans the same scalar function along a great circle.

### Parameters and Schema

- $A\in\mathbb{R}^D\setminus\{0\}$ determines orientation.
- $b\in\mathbb{R}$ determines offset.
- The Go type exists, but the JSON factory returns an error immediately after matching `plane` and reads neither parameter.

The Go fields correspond conceptually to the following structure:

```jsonc
{
  "shape": "plane",
  "A": [/* D; callers should ensure non-zero */],
  "b": "number"
}
```

This is source-structure documentation, **not a currently valid input schema**. The factory returns an error immediately after matching `plane`; it never reads A or b and never constructs the type.

### Spherical Great-Circle Geometry

The source type evaluates the hyperplane equation along a great circle:

$$
G(s)=A\cdot\gamma(s)+b
=(A\cdot o)\cos s+(A\cdot v)\sin s+b.
$$

Although this equation admits an analytic trigonometric solution, the current code delegates to the shared scalar scan-and-bisection routine. This capability is reachable only through direct Go construction because scene JSON cannot construct `shape.Plane`.

## Quadratic Surface

### Mathematical Definition

A quadric is the algebraic level set

$$
Q=\left\{x\in\mathbb{R}^3\mid
F(x)=x^TAx+b^Tx+c=0\right\}.
$$

The symmetric part $(A+A^T)/2$ determines the scalar quadratic form. Depending on its rank, signature, linear term, and constant, the level set can represent ellipsoids, hyperboloids, paraboloids, cones, cylinders, planes, or degenerate unions. Its exact differential is

$$
\nabla F(x)=(A+A^T)x+b.
$$

The implementation uses $2Ax+b$, so exact agreement requires symmetric $A$.

### Ray Intersection

Substituting $x=o+td$ produces

$$
\alpha=d^TAd,
\qquad
\beta=o^TAd+d^TAo+b^Td,
\qquad
\gamma=o^TAo+b^To+c.
$$

The real quadratic roots are solved and the nearest root in range is selected. The Spherical path scans the same scalar function along a great circle.

It has no tight intrinsic AABB, so external bounds are strongly recommended.

### Parameters and Schema

- $A\in\mathbb{R}^{3\times3}$ is encoded row-major by exactly nine finite values.
- $b\in\mathbb{R}^3$ and $c\in\mathbb{R}$.
- Symmetric $A$ is required for the implemented normal to equal the mathematical gradient.
- Although `b` is read with length $D$, the fixed $3\times3$ matrix makes safe use three-dimensional.

```jsonc
{
  "shape": "quadratic equation",
  "a": [/* exactly 9 finite numbers, 3 by 3 row-major */],
  "b": [/* exactly 3 finite numbers in safe use */],
  "c": "finite number",
  "bounds": { "pmin": [/* 3 */], "pmax": [/* 3 */] } // strongly recommended
}
```

### Spherical Great-Circle Geometry

For a Spherical scene, the code evaluates

$$
G(s)=\gamma(s)^TA\gamma(s)+b^T\gamma(s)+c,
\qquad
\gamma(s)=o\cos s+v\sin s,
$$

with the shared 2048-segment scalar root scanner and bisection refinement. The implementation does not derive the equivalent trigonometric polynomial analytically. Consequently, tangent or even-multiplicity contacts may be missed when the scan does not land near zero.

## Cubic Algebraic Surface

### Mathematical Definition

Let the factor vector be $f=[1,x,y,z]^T$. The mathematical form is:

$$
F(x,y,z)=\sum\limits_{i,j,k=0}^{3}A_{ijk}f_if_jf_k.
$$

Every index is in $[0,3]$. Different tensor permutations may represent the same monomial, so construction merges non-zero entries by their $(x\text{ power},y\text{ power},z\text{ power})$ tuple.

The zero set $F^{-1}(0)$ is a degree-at-most-three algebraic surface. It may be unbounded, disconnected, singular, or reducible. At regular points, the normal direction is $\nabla F/\|\nabla F\|$.

### Ray Intersection

For intersection, the implementation precomputes coefficient tables for powers such as $(o_x+td_x)^k$, accumulates a univariate polynomial of degree at most three, calls the general real polynomial solver, and selects the nearest valid root. The normal is the normalized gradient obtained by differentiating the merged monomials.

There is no Spherical intersection or tight intrinsic AABB.

### Parameters and Schema

- $A\in\mathbb{R}^{4\times4\times4}$ has 64 entries.
- Dense arrays and sparse objects are accepted.
- Sparse keys must use either flat indexes `0..63` exclusively or three comma-separated coordinates exclusively; the styles cannot be mixed.
- Only coordinates $(x,y,z)$ participate in the equation.

```jsonc
{
  "shape": "cubic equation",
  "a": [/* exactly 64 finite numbers */],
  // "A" may be used instead, but a and A cannot both be present.
  // A sparse object may use exclusively flat or exclusively "i,j,k" keys.
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // strongly recommended
}
```

## Four-order Surface

### Mathematical Definition

Using the same $f=[1,x,y,z]^T$ factor vector:

$$
F(x,y,z)=\sum\limits_{i,j,k,l=0}^{3}A_{ijkl}f_if_jf_kf_l.
$$

Index permutations are merged into ordinary monomials with combined exponent tuples.

The zero set is a degree-at-most-four algebraic surface and may contain multiple components, singularities, self-intersections, or repeated factors. At regular points, $\nabla F$ defines its normal.

### Ray Intersection

Substituting $x(t)=o+td$ expands the merged monomials into a univariate polynomial of degree at most four. The implementation solves all real roots, selects the nearest root in range, and evaluates the normalized monomial gradient at the hit.

This type has neither Spherical intersection nor a tight intrinsic AABB.

### Parameters and Schema

- $A\in\mathbb{R}^{4\times4\times4\times4}$ has 256 entries.
- Sparse flat indexes range from `0` through `255`.
- Coordinate keys contain exactly four coordinates, each in `[0,3]`.
- Only $(x,y,z)$ participate. `FourOrder` and `four-order` are source/schema names for a quartic.

```jsonc
{
  "shape": "four-order equation",
  "a": [/* exactly 256 finite numbers */],
  // "A" may be used instead. A sparse object may use one key style only.
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // strongly recommended
}
```

## Polynomial Surface

### Mathematical Definition

Without an output axis, the polynomial is:

$$
F(q)=\sum\limits_e c_e\prod\limits_i q_i^{e_i}.
$$

When the index rank is $\mathtt{input\_dim}+1$, the first index is an output channel. Shape intersection uses output channel 0 only. Other channels are stored but do not participate in geometry intersection.

The geometric degree is inferred from the support of the coefficient tensor:

$$
\deg F=\max_{e:c_e\ne0}\sum\limits_i e_i.
$$

The zero set may be unbounded, disconnected, singular, or reducible. At every regular point, $\nabla F$ is normal to the hypersurface. Although `input_dim` may be 1 through 3, the current affine-ray integration expects at least three ray coordinates.

### Ray Intersection

The transform is world-to-local. The local ray origin includes translation, while the local direction uses only the linear part. Every term expands $(q_{0,i}+tq_{d,i})^{e_i}$; polynomial convolution accumulates the final univariate ray polynomial, whose real roots are solved before the nearest valid root is selected. The local polynomial gradient is mapped back with the transpose of the world-to-local linear part.

The Spherical path scans $F(\operatorname{local}(\gamma(s)))$ along a great circle. The Shape's intrinsic AABB is approximately infinite, so external bounds are strongly recommended in practical scenes.

### Parameters and Schema

- `input_dim` is an integer $d\in\{1,2,3\}$.
- Each exponent index $e\in\mathbb{N}_0^d$ has a finite coefficient $c_e\in\mathbb{R}$.
- An optional leading output-channel index is stored, but only channel 0 defines geometry.
- The `coefficients` wrapper is optional; `format`, `shape`, and `terms` may be top-level.
- No `degree` field is read. Legacy non-implicit `mode` values and every `explicit_axis` field are rejected.

```jsonc
{
  "shape": "polynomial surface",
  "input_dim": "integer in [1,3]",
  "coefficients": {
    "format": "hash | coo",                           // optional, default hash
    "shape": [/* input_dim or input_dim+1 positive integers */], // optional
    "terms": [{
      "index": [/* input_dim or input_dim+1 non-negative integers */],
      "value": "finite number"
    }]
  },
  "transform": [[/* 4 */], [/* 4 */], [/* 4 */], [/* 4 */]], // optional; flat 16 also accepted
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // strongly recommended
}
```

### Multinomial Ray Expansion

Each coordinate power is expanded with the binomial theorem:

$$
(q_{0,i}+tq_{d,i})^{e_i}
=\sum\limits_{k=0}^{e_i}
\binom{e_i}{k}q_{0,i}^{e_i-k}q_{d,i}^{k}t^k.
$$

Polynomial convolution then evaluates the product over coordinates and accumulation over sparse terms. Thus the multivariate level-set equation becomes one exact univariate polynomial in $t$, up to floating-point arithmetic. The solver enumerates its real roots and applies the query interval afterward.

### Spherical Great-Circle Geometry

The Spherical path evaluates the transformed field along

$$
G(s)=F\!\left(T\gamma(s)\right),
\qquad
\gamma(s)=o\cos s+v\sin s,
$$

using the shared scan-and-bisection scalar solver. Unlike the current `ImplicitEquation` Spherical path, this implementation applies its world-to-local transform before evaluating $F$. It remains subject to the shared scanner's tangent-root and even-multiplicity limitations.

## Implicit Equation

### Mathematical Definition

Every variant represents the local zero level set $F(q)=0$. The local coordinates are formed by the documented world-to-local 4 by 4 transform. In the implementation, rows 1 through 3 are applied to $[1,x,y,z]^T$. The `center`, `scale`, and `basis` form is compiled into the same transform representation. Bounds always remain a world-space AABB.

The resulting world-space surface is

$$
S=\left\{x\in\mathbb{R}^3\mid F(Tx)=0\right\},
$$

where $T$ denotes the affine world-to-local map in homogeneous coordinates. At a regular point where $\nabla F\ne0$, the gradient is normal to the level set. Singular points satisfy both $F=0$ and $\nabla F=0$.

The implemented scalar-field families are:

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
  F(x)=\sum\limits_i w_i\exp\!\left(-k\|x-c_i\|^2\right)-\mathrm{iso}.
  $$

The expression environment provides `x`, `y`, `z`, `pi`, `e`, and the functions `abs`, `sqrt`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`, `sinh`, `cosh`, `tanh`, `exp`, `log`, `log10`, `floor`, `ceil`, `round`, `pow`, `min`, `max`, `clamp`, and `sign`. Constants cannot override reserved names. When an explicit gradient is absent, the factory first attempts symbolic differentiation. If that is unavailable, the Shape falls back to a centered finite-difference normal.

### Ray Intersection

Affine intersection proceeds as follows:

1. If bounds exist, clip the query interval with a box slab test.
2. Scan the resulting interval with `step`. Without an explicit step, a bounded field derives it from the bounds diagonal divided by approximately 512 samples. An unbounded query with an infinite maximum uses 0.02.
3. Scan at most 2048 steps. A value within `value_tol` is accepted immediately; a sign change is refined by bisection with an internal root tolerance of `1e-6`.
4. Compute the normal from the analytic or automatically differentiated gradient when available, otherwise use centered differences with epsilon `1e-5`. Transform the local gradient back to world space with the transpose of the world-to-local linear part.

This scan can miss two crossings inside one step, an even-multiplicity root, or a tangent root whose sampled values never become sufficiently small. High-frequency fields should use tighter bounds and a smaller `step`.

The Spherical implementation has two current limitations. It passes world points directly to `Function` without applying the world-to-local transform. It also finds the first root on the entire arc before checking bounds; if that root lies outside the bounds, it does not continue to search for later roots inside the bounds.

### Parameters and Schema

- $F:\mathbb{R}^3\to\mathbb{R}$ is selected by `field.type`.
- $T\in\mathbb{R}^{4\times4}$ is a world-to-local homogeneous transform. Explicit `transform` takes precedence over `center`, `scale`, and `basis`.
- Bounds are a world-space AABB and are strongly recommended.
- `step` is a positive scan increment; `value_tol` is a positive near-root threshold with default $10^{-7}$.

```jsonc
{
  "shape": "implicit equation",
  "field": { "type": "expr | gyroid | lp_power_sum | metaballs" },

  // Alternative world-to-local placement forms; transform takes precedence.
  "transform": [[/* 4 */], [/* 4 */], [/* 4 */], [/* 4 */]], // optional
  "center": [/* D; optional, default zero */],
  "scale": "positive number | D positive numbers",             // optional, default one
  "basis": [[/* 3 */], [/* 3 */], [/* 3 */]],                  // optional orthonormal rows

  "bounds": { "pmin": [/* D */], "pmax": [/* D */] },       // optional, recommended
  "step": "positive number",                                  // optional
  "value_tol": "positive number"                              // optional, default 1e-7
}
```

```jsonc
// expr
{
  "type": "expr",
  "expr": "F(x,y,z) expression",
  "constants": { "name": 1.0 },
  "gradient": { "x": "dF/dx", "y": "dF/dy", "z": "dF/dz" }
}

// gyroid
{
  "type": "gyroid",
  "frequency": "positive number",
  "offset": "finite number"
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
  "iso": "finite number",
  "balls": [{ "weight": "finite number", "center": [/* exactly 3 */] }]
}
```

### Differential Geometry

For an analytic local field $F(q)$ and affine world-to-local map $q=Lx+t$, the world-space gradient is

$$
\nabla_x(F\circ T)(x)=L^T\nabla_qF(q).
$$

Expression fields prefer an explicit gradient, then symbolic differentiation. If neither is available, centered finite differences use

$$
\frac{\partial F}{\partial q_i}(q)
\approx
\frac{F(q+\varepsilon e_i)-F(q-\varepsilon e_i)}{2\varepsilon},
\qquad
\varepsilon=10^{-5}.
$$

### Deterministic Field Sampling

The root search samples $F(o+td)$ on a one-dimensional interval rather than sampling the geometric surface. For bounded fields, the default step is derived from the AABB diagonal to target approximately 512 samples; the hard scan limit is 2048 steps. A smaller explicit `step` improves high-frequency coverage but increases field evaluations. Sign-change sampling does not guarantee detection of tangent roots, roots of even multiplicity, or two crossings contained in one step.

### Spherical Great-Circle Geometry

The Spherical path searches roots of

$$
G(s)=F(\gamma(s)),
\qquad
\gamma(s)=o\cos s+v\sin s.
$$

Current code passes $\gamma(s)$ directly to `Function`; it does not apply the configured world-to-local transform. Moreover, it obtains the first root before applying bounds. If that root lies outside the bounds, later roots inside the bounds are not considered.

## Parametric Surface

### Mathematical Definition

A parametric surface is the image of a two-dimensional parameter domain under a map

$$
P:U\times V\subset\mathbb{R}^2\longrightarrow\mathbb{R}^3.
$$

At a regular point, $P_u$ and $P_v$ are linearly independent and span the tangent plane. The oriented normal is

$$
n(u,v)=\frac{P_u\times P_v}{\|P_u\times P_v\|}.
$$

Expression mode supplies the three coordinate functions directly. When explicit derivatives are absent, the factory first attempts symbolic differentiation and then falls back to finite differences. Placement produces

$$
P_{\mathrm{world}}(u,v)=c+\operatorname{diag}(s)P(u,v).
$$

Spherical harmonic mode evaluates a real harmonic sum

$$
\psi(u,v)=\sum\limits_j w_jY_{l_jm_j}(u,v),
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

Spherical-harmonic mode is a radial parameterization. Zeros and critical points of $\psi$ can produce geometric singularities or orientation changes because the radius is $|\psi|$.

### Ray Intersection

Intersection proceeds as follows:

1. Split the parameter domain into $\mathtt{samples\_u}\times\mathtt{samples\_v}$ patches.
2. Estimate each patch AABB from nine evaluations: four corners, the center, and four edge midpoints. Add padding and build a patch BVH.
3. When a ray hits a patch AABB, seed a solve with $(t_{\mathrm{near}},u_{\mathrm{center}},v_{\mathrm{center}})$ for the three-equation system $o+td-P(u,v)=0$.
4. Use a Newton Jacobian with columns $[d,-P_u,-P_v]$, plus backtracking line search. Validate $t$, the parameter ranges, and the final residual.
5. Return $(P_u\times P_v)/\|P_u\times P_v\|$ as the normal and normalized parameter-range coordinates as UV.

The factory accepts any positive integer for the sample counts, while the runtime accessors replace values below 2 with the default 32. The effective minimum is therefore 2. The sampled AABB is not an analytic bound; a high-frequency or high-curvature surface may extend outside it. Increase sample counts or padding when necessary. The type has no Spherical great-circle intersection and no area sampler.

### Parameters and Schema

- $U=[u_{\min},u_{\max}]$ and $V=[v_{\min},v_{\max}]$ must be increasing intervals.
- $P$, $P_u$, and $P_v$ map into $\mathbb{R}^3$.
- `samples_u` and `samples_v` control the patch discretization; Newton and residual fields control numerical acceptance.
- `center` and positive scalar/vector `scale` define world placement.

```jsonc
{
  "shape": "parametric equation",
  "surface": { "type": "expr | spherical_harmonic" },
  "u_range": ["min", "max"],                        // optional, default [0,1]
  "v_range": ["min", "max"],                        // optional, default [0,1]
  "center": [/* exactly 3 */],                        // optional
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

```jsonc
// Expression surface
{
  "type": "expr",
  "x": "x(u,v) expression",
  "y": "y(u,v) expression",
  "z": "z(u,v) expression",
  "constants": { "name": 1.0 },
  "derivative": {
    "du": { "x": "dx/du", "y": "dy/du", "z": "dz/du" },
    "dv": { "x": "dx/dv", "y": "dy/dv", "z": "dz/dv" }
  }
}

// Spherical-harmonic surface
{
  "type": "spherical_harmonic",
  "terms": [{
    "l": "integer >= 0",
    "m": "integer in [0,l]",
    "weight": "finite number; default 1",
    "basis": "cos | sin; default cos"
  }]
}
```

### Parameter-Domain Sampling and Patch BVH

The implementation performs deterministic parameter-domain sampling, not stochastic surface-area sampling. It divides $U\times V$ into a regular patch grid. Each patch AABB is estimated from nine evaluations: four corners, four edge midpoints, and the center. After `bounds_padding` is applied, the patch boxes form an internal BVH.

This construction is conservative only when those samples capture the extrema of $P$ on each patch. High curvature or frequency can place the true surface outside the estimated AABB, producing a missed intersection. Increasing `samples_u`, `samples_v`, or `bounds_padding` reduces this risk. The Shape does not implement `SurfaceSampler`, so these patch samples do not make it a sampleable area light.

### Newton Geometry

For each candidate patch, the unknown vector is $y=(t,u,v)^T$ and the residual is

$$
R(y)=o+td-P(u,v).
$$

The Newton Jacobian is

$$
J(y)=\begin{bmatrix}d&-P_u&-P_v\end{bmatrix}.
$$

An iteration solves

$$
J(y_k)\,\Delta y_k=-R(y_k),
\qquad
y_{k+1}=y_k+\alpha_k\Delta y_k,
$$

where a backtracking line search selects $\alpha_k$. The result is accepted only when $t$, $(u,v)$, and the final residual satisfy their configured ranges and tolerances. A singular or ill-conditioned $J$ corresponds geometrically to a degenerate tangent frame or a ray configuration that does not locally determine a stable intersection.

## Parametric Curve

### Mathematical Definition

This Shape is not a zero-width curve. Given a spine $C:I\to\mathbb{R}^3$ and a positive radius function $r:I\to\mathbb{R}_{>0}$, it is the boundary of the swept-sphere union

$$
S=\partial\bigcup_{t\in I}B(C(t),r(t)).
$$

This construction is a variable-radius canal surface with spherical endpoint caps. A point on the numerically selected envelope sphere has normal

$$
n(p,t^*)=\frac{p-C(t^*)}{\|p-C(t^*)\|}.
$$

Top-level scale affects both $C$ and the radius, so non-uniform scale is rejected. Depending on the curvature of $C$ and the variation of $r$, the swept union can overlap itself or develop envelope singularities.

### Ray Intersection

The implementation takes $\mathtt{samples}+1$ uniform curve samples. Adjacent samples form a segment whose AABB uses both endpoints, the midpoint, the maximum sampled radius, the chord sagitta, and padding. It builds a segment BVH, performs a capsule overlap test for candidate segments, and uses golden-section refinement to minimize the earliest ray-entry distance over the spheres along that interval. The globally nearest hit is returned.

The normal is the hit point minus its corresponding sphere center. Tangents prefer explicit or symbolic derivatives and fall back to finite differences. As with parametric surfaces, sample counts below 2 are replaced by the runtime default. A dynamic radius expression is validated as finite and positive only when it is evaluated. The type has neither Spherical great-circle intersection nor area sampling.

### Parameters and Schema

- $I=[t_{\min},t_{\max}]$ is an increasing interval.
- $C(t)=(x(t),y(t),z(t))\in\mathbb{R}^3$ is the spine.
- $r(t)>0$ is either constant or expression-defined; `r` is a compatibility alias for `radius`.
- Scale is a positive uniform scalar, optionally written as `[s,s,s]`.

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
  "center": [/* exactly 3 */],                    // optional
  "scale": "positive scalar or [s,s,s]",         // optional, must be uniform
  "samples": "positive integer; default 256",
  "refine_iter": "positive integer; default 40",
  "derivative_eps": "positive number; default 1e-5",
  "bounds_padding": "non-negative number; default 1e-6",
  "bounds": { "pmin": [/* 3 */], "pmax": [/* 3 */] } // optional
}
```

### Spine Sampling and Segment Bounds

The implementation takes `samples + 1` uniformly spaced parameter samples

$$
t_i=t_{\min}+\frac{i}{N}(t_{\max}-t_{\min}),
\qquad
i=0,\ldots,N.
$$

Each interval $[t_i,t_{i+1}]$ receives an AABB derived from its endpoints, midpoint, sampled maximum radius, chord sagitta, and padding. These boxes form an internal segment BVH. As with parametric surfaces, this is a discrete bound estimate rather than an analytic enclosure; insufficient sampling can miss high-curvature or rapidly varying-radius portions of the canal surface.

### One-Dimensional Envelope Optimization

For a fixed spine parameter $q$, intersection with the corresponding sphere solves

$$
\|o+td-C(q)\|^2-r(q)^2=0.
$$

Writing $m(q)=o-C(q)$ and $a=d\cdot d$, its earliest real root is

$$
t_-(q)=
\frac{-d\cdot m(q)-
\sqrt{[d\cdot m(q)]^2-a(\|m(q)\|^2-r(q)^2)}}{a}.
$$

After a capsule overlap test rejects impossible intervals, golden-section refinement searches the parameter interval for the minimum admissible $t_-(q)$. The nearest result over all candidate segments is retained. This optimization approximates the swept-sphere envelope; its accuracy depends on segment sampling and `refine_iter`.

## 4D Klein-bottle tube

### Mathematical Definition

The Klein bottle is a compact non-orientable two-manifold. Unlike its familiar self-intersecting representation in $\mathbb{R}^3$, it admits an embedding in $\mathbb{R}^4$. The implementation uses

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

Here $u,v$ are periodic parameters, $R=r_{\mathrm{major}}$, and $r=r_{\mathrm{minor}}$. A two-dimensional surface has codimension two in 4D, so a one-dimensional ray almost never intersects it directly. The rendered geometry is therefore the boundary of a radius-$\tau$ tubular neighborhood:

$$
S_\tau=\left\{p\in\mathbb{R}^4\mid
F(p)=\operatorname{dist}(p,S)-\tau=0\right\},
$$

where $\tau$ is `thickness`.

### Ray Intersection

Closest-point evaluation starts with a $16\times8$ parameter seed grid, retains the eight nearest seeds, and refines $(u,v)$ with a first-order least-squares/Newton-style iteration and line search. The ray is first clipped to an analytic AABB, then sphere-traced for at most 128 iterations. The step is approximately $0.75|\mathrm{SDF}|$ and never smaller than $\max(10^{-6},0.02\tau)$. A near-zero value is accepted directly; a sign change is refined by bisection.

The normal is

$$
n(p)=\frac{p-S(u^*,v^*)}{\|p-S(u^*,v^*)\|}.
$$

UV contains the optimized parameters after applying the Klein-bottle twist and wrapping rules.

Because the SDF depends on numerical closest-point optimization, it is not guaranteed to be a globally exact distance field. Thin tubes and regions with competing nearby surface points can be sensitive to seed density and marching parameters. These tuning fields are not exposed in JSON. The type does not implement `IntersectGeodesic`, so it cannot participate in Spherical great-circle tracing.

### Parameters and Schema

- $c\in\mathbb{R}^4$ is the translation center.
- $R=r_{\mathrm{major}}>r=r_{\mathrm{minor}}>0$.
- $\tau=\mathtt{thickness}>0$ is the tube radius.
- `render.dimension` must be 4; `position` is accepted as an alias for `center`.

```jsonc
{
  "shape": "klein_bottle",
  "center": [/* exactly 4; position may be used instead */],
  "r_major": "positive number",
  "r_minor": "positive number smaller than r_major",
  "thickness": "positive number",
  "bounds": { "pmin": [/* 4 */], "pmax": [/* 4 */] } // optional
}
```

### Closest-Point Optimization

For a query point $p\in\mathbb{R}^4$, the numerical core minimizes

$$
E(u,v)=\frac12\|p-S(u,v)\|^2.
$$

Stationary parameters satisfy

$$
\frac{\partial E}{\partial u}=-(p-S)\cdot S_u=0,
\qquad
\frac{\partial E}{\partial v}=-(p-S)\cdot S_v=0.
$$

The code evaluates a $16\times8$ seed grid, keeps the eight nearest seeds, and refines each with a first-order least-squares/Newton-style step plus line search. The best refined candidate estimates

$$
\operatorname{dist}(p,S)=\min_{u,v}\|p-S(u,v)\|.
$$

Because the optimizer can converge to a local rather than global minimum, the resulting distance is numerical rather than guaranteed exact.

### Distance-Field Marching

With

$$
\phi(p)=\operatorname{dist}(p,S)-\tau,
$$

the affine ray marcher advances approximately by

$$
\Delta t=
\max\!\left(0.75|\phi(o+td)|,
\max(10^{-6},0.02\tau)\right).
$$

It performs at most 128 iterations. Near-zero values are accepted directly, while a sign-changing interval is refined by bisection. Since $\phi$ depends on numerical closest-point optimization, the usual exact signed-distance guarantee for sphere tracing does not strictly hold.

### Geometric-Space Compatibility

`KleinBottle4D` refers to the topology of the embedded Klein bottle, not to the engine's 3D Klein model of hyperbolic space. The Shape implements only affine intersection in four dimensions and does not override `IntersectGeodesic`. It is therefore usable in a 4D Euclidean render path, not in Spherical great-circle tracing or as a 3D Klein-model Shape.

## Triangulated Surface Mesh

### Mathematical Definition

An STL file represents a finite triangle union

$$
M=\bigcup_{j=1}^{N}T_j,
$$

where each $T_j$ is a Euclidean 2-simplex in $\mathbb{R}^3$. The union is only a closed orientable 2-manifold when its facets have consistent orientation and satisfy the required edge-incidence conditions; STL itself does not guarantee those properties.

STL is an importer rather than an independent runtime Shape. Every facet becomes a `shape.Triangle`, and file normals are ignored. Normals are recomputed from the transformed vertices.

### Ray Intersection

After import and transformation, each facet uses the triangle's Moller-Trumbore intersection, barycentric domain test, gradient data, AABB, and surface sampling. The ObjectTree handles the resulting triangle collection through its normal BVH path. There is no separate STL intersection equation at runtime.

### Parameters and Schema

The parser treats a file as ASCII when its first line begins with the literal `solid`. It scans every `vertex` line and creates one Triangle for every group of three vertices. Otherwise it reads binary STL: an 80-byte header, the facet count, and each 50-byte facet record.

The transform columns are $x_{\mathrm{dir}}$, $(z_{\mathrm{dir}}\times x_{\mathrm{dir}})/\|z_{\mathrm{dir}}\times x_{\mathrm{dir}}\|$, and $z_{\mathrm{dir}}$, each multiplied by its corresponding scale and followed by center translation. The x and z directions are individually normalized, but the factory does not validate non-zero input, mutual orthogonality, handedness, or positive/non-zero scale. Callers should provide a valid orthonormal frame explicitly.

Although input vector lengths are checked against $D$, the transformation matrix and output vertices are fixed at 3D, and Triangle itself is safe only in 3D. STL must therefore be used with $D=3$. The simple `solid` detection can misclassify a binary STL whose header starts with that word.

```jsonc
{
  "shape": "stl",
  "file": "path string",
  "center": [/* exactly 3 finite numbers */],
  "z_dir": [/* exactly 3 finite numbers */],
  "x_dir": [/* exactly 3 finite numbers */],
  "scale": [/* exactly 3 finite numbers */],
  "bounds": { "pmin": [/* 3 */], "pmax": [/* 3 */] } // optional
}
```

### Inherited Triangle Sampling and Acceleration

STL import does not create a single mesh-level Shape or a global mesh-area distribution. It expands the file into independent triangles. Each non-degenerate 3D facet therefore inherits

$$
A_j=\frac12\|(p_{2,j}-p_{1,j})\times(p_{3,j}-p_{1,j})\|
$$

and the triangle sampler's constant conditional area density

$$
p_A(x\mid T_j)=\frac{1}{A_j}.
$$

The ObjectTree BVH accelerates the resulting facet objects. Any selection distribution across multiple emissive facets belongs to the light/object layer rather than to an STL Shape, because no runtime STL Shape remains after parsing.

## Finite Cylinder

### Mathematical Definition

Let $c\in\mathbb{R}^D$ be the center, let $a$ be a unit axis, and let $r,h>0$. The closed finite cylinder is the boundary of

$$
V=\left\{x\in\mathbb{R}^D\mid
\left\|(x-c)-[(x-c)\cdot a]a\right\|\le r,
\ |(x-c)\cdot a|\le\frac{h}{2}\right\}.
$$

Its boundary consists of a lateral surface and two disk caps. It is compact and convex. The lateral normal is radial and orthogonal to $a$; the cap normals are $\pm a$.

### Ray Intersection

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

### Parameters and Schema

- $c,a\in\mathbb{R}^D$, with $a\ne0$ before normalization.
- $r,h\in\mathbb{R}_{>0}$.
- `cylinder` and `finite cylinder` are equivalent discriminators.
- `position` is accepted as a compatibility alias for `center`.

```jsonc
{
  "shape": "cylinder | finite cylinder",
  "center": [/* D; position may be used instead */],
  "axis": [/* D finite numbers, non-zero */],
  "r": "positive number",
  "height": "positive number",
  "bounds": { "pmin": [/* D */], "pmax": [/* D */] } // optional
}
```

