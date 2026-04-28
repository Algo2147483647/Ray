# Mathematical Foundations

This project is built on a compact but important set of mathematical ideas. Most of them come from linear algebra, analytic geometry, numerical root solving, and Monte Carlo estimation.

## 1. Vector Spaces and Coordinate Representation

The renderer treats points, directions, normals, and colors as vectors.

- Geometric vectors are stored with `gonum`'s `mat.VecDense`.
- RGB color is represented as a 3D vector.
- Rays live in `utils.Dimension` spatial dimensions. The current runtime constant is `3`, but parts of the codebase already prepare for higher-dimensional rendering.

Key consequences:

- A point and a direction have the same storage type but different physical meaning.
- Colors are multiplied component-wise to model attenuation by materials.
- Normals are normalized before use so that projection formulas remain correct.

Relevant code:

- `model/optics/ray.go`
- `model/camera/camera_3d.go`
- `model/camera/camera_n_dim.go`
- `utils/global.go`

## 2. Core Linear Algebra Operations

The renderer repeatedly uses the following operations:

- **Dot product** `a · b`
  Used for projection, angle tests, Fresnel terms, visibility tests, and intersection formulas.

- **Cross product** `a x b`
  Used in 3D to build perpendicular directions and triangle intersection tests.

- **Normalization**
  Converts a vector `v` into `v / ||v||`, keeping only direction.

- **Affine ray evaluation**
  A point along a ray is computed as:

  ```text
  x(t) = o + t d
  ```

  where `o` is the ray origin and `d` is the ray direction.

Relevant code:

- `utils/geometrical_optics.go`
- `model/shape/sphere.go`
- `model/shape/triangle.go`
- `ray_tracing/trace_ray.go`

## 3. Camera Bases and Orthonormalization

The N-dimensional camera uses Gram-Schmidt orthonormalization to convert a user-provided set of basis directions into an orthonormal frame.

Conceptually:

1. Start from a set of linearly independent direction vectors.
2. Remove components parallel to earlier vectors.
3. Normalize each remaining vector.

This matters because perspective offsets should be measured along mutually perpendicular image axes. Without orthonormalization, the image axes would be skewed and the camera model would become inconsistent.

Relevant code:

- `model/camera/camera_n_dim.go`

## 4. Field of View and Perspective Scaling

The camera converts normalized pixel coordinates into angular offsets using:

```text
halfHeight = tan(FOV / 2)
halfWidth  = aspectRatio * halfHeight
```

This is standard pinhole-camera geometry. The tangent appears because the image plane is placed at unit distance from the camera center, turning angular aperture into linear displacement on that plane.

In 3D, the outgoing direction is:

```text
d' = normalize(d + u * halfWidth * right - v * halfHeight * up)
```

where:

- `d` is the forward direction,
- `right` is the camera's horizontal basis vector,
- `up` is the vertical basis vector,
- `u` and `v` are normalized screen coordinates in `[-1, 1]`.

Relevant code:

- `model/camera/camera_3d.go`

## 5. Random Variables and Monte Carlo Estimation

Pixel color is estimated by averaging multiple randomized samples:

```text
L_hat = (1 / N) sum_i L_i
```

where each `L_i` is the radiance estimate from one sampled ray path.

The project uses randomness in two places:

- **Camera jitter**: sub-pixel offsets reduce aliasing.
- **Material branching**: a surface randomly chooses reflection, refraction, or diffuse scattering based on material probabilities.

This is a Monte Carlo estimator because deterministic integrals over image area and scattering events are replaced by random sampling plus averaging.

Important implementation note:

- The renderer is stochastic, but it does not implement a full physically unbiased path-tracing estimator with explicit BRDF/PDF weighting.
- Instead, it uses random branch selection plus user-defined loss factors, which makes it a practical Monte Carlo ray tracer rather than a strict radiometric reference implementation.

Relevant code:

- `ray_tracing/trace_pixel.go`
- `model/camera/camera_3d.go`
- `model/camera/camera_n_dim.go`
- `model/optics/material.go`

## 6. Polynomial Root Solving

Ray-surface intersection often reduces to solving a polynomial equation in the ray parameter `t`.

### Quadratic Case

For a quadratic surface, substituting `x(t) = o + t d` into the surface equation yields:

```text
a t^2 + b t + c = 0
```

The physically relevant answer is the smallest positive root larger than a numerical threshold `EPS`.

### Quartic Case

For the fourth-order implicit surface, the substitution yields:

```text
a4 t^4 + a3 t^3 + a2 t^2 + a1 t + a0 = 0
```

The implementation computes all roots, filters to real positive roots, and chooses the nearest valid one.

This is a standard geometric pattern:

1. Write the surface as an algebraic equation.
2. Substitute the ray equation.
3. Solve for path parameter `t`.
4. Keep the nearest forward intersection.

Relevant code:

- `model/shape/quadratic_equation.go`
- `model/shape/forth-order_equation.go`

## 7. Gradients as Surface Normals

For an implicit surface defined by:

```text
F(x) = 0
```

the normal direction is given by the gradient:

```text
n proportional to grad F(x)
```

After normalization:

```text
n = grad F(x) / ||grad F(x)||
```

This is why the quadratic and fourth-order surfaces compute partial derivatives instead of storing explicit normals.

Examples in the project:

- Quadratic surface:

  ```text
  F(x) = x^T A x + b^T x + c
  grad F(x) = 2 A x + b
  ```

- Fourth-order surface:
  The gradient is assembled by differentiating the fourth-order tensor expression term by term.

Relevant code:

- `model/shape/quadratic_equation.go`
- `model/shape/forth-order_equation.go`

## 8. Numerical Thresholds

The project uses:

```text
EPS = 1e-5
```

This threshold is used to:

- reject intersections behind or extremely close to the ray origin,
- avoid self-intersection loops after a surface hit,
- stabilize parallelism checks in plane and slab-style tests.

This is a standard numerical-analysis safeguard in ray tracing because floating-point arithmetic cannot reliably distinguish exact contact from tiny penetration.

Relevant code:

- `utils/global.go`
- `model/shape/sphere.go`
- `model/shape/plane.go`
- `model/object/object_tree.go`

## 9. Tensors and Higher-Order Coefficients

The fourth-order implicit surface stores its coefficients in a 4 x 4 x 4 x 4 tensor-like structure. The index convention includes:

- index `0` for the constant factor `1`,
- index `1` for `x`,
- index `2` for `y`,
- index `3` for `z`.

This lets a general fourth-degree polynomial be assembled as a multilinear combination of basis factors. It is mathematically flexible enough to express a wide class of algebraic surfaces.

Relevant code:

- `model/shape/forth-order_equation.go`

## 10. What the Math Enables

These mathematical tools directly support:

- projection from camera space to rays,
- exact or algebraic intersection tests,
- normal construction for shading,
- stochastic anti-aliasing,
- wave-dependent refraction,
- acceleration structures based on bounding boxes,
- experiments with higher-dimensional imaging.
