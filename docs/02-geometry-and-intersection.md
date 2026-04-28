# Geometry and Intersection Theory

This document explains how the project models shape and how it turns a ray into a concrete hit point on geometry.

## 1. The Central Geometric Problem

For every ray, the renderer must answer:

1. Which object does the ray hit first?
2. At what distance `t` along the ray?
3. What is the surface normal at the hit point?

Everything in the geometry subsystem exists to answer these three questions efficiently and robustly.

Relevant code:

- `model/shape/shape.go`
- `model/object/object_tree.go`
- `ray_tracing/trace_ray.go`

## 2. Shape Interface as a Mathematical Contract

Every shape implements:

- `Intersect(rayStart, rayDir) float64`
- `GetNormalVector(intersect, res) *mat.VecDense`
- `BuildBoundingBox() (pmin, pmax *mat.VecDense)`

This corresponds to three mathematical responsibilities:

- solve the ray-surface equation,
- recover the local differential orientation,
- provide a conservative spatial enclosure for acceleration.

Relevant code:

- `model/shape/shape.go`

## 3. Sphere Intersection

A sphere is defined by center `c` and radius `R`:

```text
||x - c||^2 = R^2
```

Substitute the ray equation `x(t) = o + t d`:

```text
||o + t d - c||^2 = R^2
```

Expanding yields a quadratic equation in `t`. The implementation solves that equation and returns the smallest positive root above `EPS`.

Normal computation:

```text
n = normalize(x_hit - c)
```

Relevant code:

- `model/shape/sphere.go`

## 4. Plane Intersection

A plane is represented as:

```text
f(x) = a^T x + b = 0
```

Substitute `x(t) = o + t d`:

```text
a^T (o + t d) + b = 0
```

Solving for `t` gives:

```text
t = -(a^T o + b) / (a^T d)
```

If the denominator is near zero, the ray is parallel to the plane and there is no stable intersection.

The plane normal is just the normalized coefficient vector `a`.

Important implementation note:

- A plane type exists mathematically and programmatically.
- The current JSON shape parser explicitly reports `"plane"` as declared but not implemented for script loading.

Relevant code:

- `model/shape/plane.go`
- `controller/parse_shape.go`

## 5. Triangle Intersection

The triangle intersector uses the classic Moller-Trumbore idea: solve the hit point directly in barycentric coordinates without first intersecting the supporting plane and then performing a separate point-in-triangle test.

The method represents the hit as:

```text
x = P1 + u (P2 - P1) + v (P3 - P1)
```

with constraints:

- `u >= 0`
- `v >= 0`
- `u + v <= 1`

This gives both:

- a ray-plane solve,
- an inclusion test inside the triangle.

The triangle normal is computed from edge vectors:

```text
n = normalize((P2 - P1) x (P3 - P1))
```

Relevant code:

- `model/shape/triangle.go`

## 6. Cuboid Intersection

The cuboid uses the slab method. For each dimension, the ray intersects a pair of parallel planes, producing an interval of valid `t` values. Intersecting those intervals across all dimensions gives the final entry and exit times.

In 3D, this is the standard axis-aligned bounding-box intersection algorithm.

Mathematically:

1. Solve for entry and exit `t` along each coordinate axis.
2. Track the largest entry time `t0`.
3. Track the smallest exit time `t1`.
4. If `t0 > t1`, the ray misses the box.

This same idea is used both for real cuboid geometry and for acceleration-structure bounding boxes.

Relevant code:

- `model/shape/cuboid.go`

## 7. Quadratic Surfaces

The project supports a general quadratic implicit surface:

```text
F(x) = x^T A x + b^T x + c = 0
```

This is a broad family that can represent spheres, ellipsoids, paraboloids, hyperboloids, and more depending on `A`, `b`, and `c`.

Substituting the ray equation produces a scalar quadratic equation in `t`. The hit normal comes from the gradient:

```text
grad F(x) = 2 A x + b
```

This is the most mathematically expressive primitive in the code after the fourth-order surface.

Relevant code:

- `model/shape/quadratic_equation.go`

## 8. Fourth-Order Algebraic Surfaces

The project also supports a fourth-degree implicit surface built from a fourth-order coefficient tensor.

This allows surfaces that are not easily expressible as basic primitives, such as more stylized or research-oriented algebraic shapes.

The implementation works in two stages:

1. Substitute the ray into the polynomial surface and accumulate the quartic coefficients.
2. Solve the quartic and keep the smallest positive real root.

The normal is the normalized gradient of the polynomial at the hit point.

This part of the system embeds real symbolic-geometry thinking inside a ray tracer.

Relevant code:

- `model/shape/forth-order_equation.go`

## 9. Implicit Surfaces as a Design Direction

`ImplicitEquation` exists as a conceptual placeholder for a more generic implicit-surface framework:

```text
F(x) = 0
```

The current `IntersectPure` implementation returns no hit, so this type documents an intended architectural direction rather than a completed feature.

What it suggests mathematically:

- the project is designed to grow from analytic primitives toward general numerical root-finding on arbitrary scalar fields,
- future implementations could use Newton iteration, interval methods, sphere tracing, or subdivision.

Relevant code:

- `model/shape/implicit_equation.go`

## 10. STL Meshes and Coordinate Transforms

The parser can load STL geometry and convert each facet into a triangle.

The transform pipeline includes:

- translation by `position`,
- orientation via user-provided `x_dir` and `z_dir`,
- construction of `y_dir` by cross product,
- non-uniform scaling.

This is a local-to-world transform expressed as a 4 x 4 homogeneous matrix. Each vertex is transformed into world coordinates before being wrapped as a triangle.

That means STL support is not just file parsing; it is also a concrete application of rigid-body orientation, basis construction, and affine transformations.

Relevant code:

- `controller/parse_shape.go`

## 11. Bounding Boxes and Spatial Acceleration

Every shape provides a bounding box. Those boxes are used to build an object tree that functions like a simple bounding-volume hierarchy.

### Why Bounding Boxes Matter

Without acceleration, every ray would test every object:

```text
cost per ray ~ O(number of objects)
```

With a bounding hierarchy, many objects are rejected early by testing cheap box intersections before expensive shape intersections.

### Tree Construction Strategy

The tree builder:

1. creates one leaf node per object,
2. computes a bounding box enclosing a subset of objects,
3. chooses a split dimension using center variance,
4. sorts by box centers along that dimension,
5. recursively partitions the objects.

This is a geometric-statistical heuristic: the split dimension is chosen where object centers are most spread out.

Relevant code:

- `model/object/object_tree.go`
- `model/object/object_node.go`

## 12. First-Hit Selection

Once both child nodes are queried, the renderer keeps the smaller valid distance. Distances smaller than `EPS` are discarded to reduce self-hit loops near a surface contact.

This logic is critical physically because light transport depends on the *first* visible interface along the ray, not just any intersection.

Relevant code:

- `model/object/object_tree.go`

## 13. Engraving Functions as Geometric Masks

Some shapes can attach an `EngravingFunc`, which conditionally suppresses otherwise valid intersections.

Geometrically, this behaves like a procedural mask on the surface. For example, the sample sphere engraving computes spherical angles and applies a sinusoidal stripe test.

This is an interesting hybrid idea:

- base geometry comes from analytic intersection,
- fine detail comes from procedural acceptance or rejection of the hit.

Relevant code:

- `utils/example_lib/engraving_func.go`
- `controller/parse_shape.go`
- `model/shape/sphere.go`
- `model/shape/cuboid.go`

## 14. Summary of Embedded Geometry Knowledge

The geometry subsystem contains:

- analytic intersection theory,
- barycentric geometry,
- implicit surface differentiation,
- affine mesh transformation,
- bounding-volume acceleration,
- procedural geometric masking.

That is why this project is mathematically richer than a minimal sphere-only teaching renderer.
