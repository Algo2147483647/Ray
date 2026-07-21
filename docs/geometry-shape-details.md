# Current Geometry Shape Details

This document is the current geometry shape detail table for the renderer. It is
based on the JSON factory and shape implementations in:

- `engine/controller/factory/shapes.go`
- `engine/controller/factory/implicit_equation.go`
- `engine/controller/factory/polynomial_surface.go`
- `engine/model/shape/`
- `scene-editor/src/data/shapeSchemas.ts`

The existing `geometry-intersection-theory.md` explains the math and intersection
ideas. This file is the operational checklist: supported `shape` values, fields,
runtime status, bounds, dimensions, and known caveats.

For interface-level JSON contracts, use
[`engine-json-protocol.md`](engine-json-protocol.md) and
[`studio-json-protocol.md`](studio-json-protocol.md). This file focuses on
shape behavior and support status.

## Current Coverage Summary

| JSON `shape` value | Runtime implementation | JSON load status | Main use |
| --- | --- | --- | --- |
| `cuboid` | `shape.Cuboid` | Supported | Axis-aligned box / rectangular solid |
| `hypercuboid` | `shape.Cuboid` | Supported | N-dimensional axis-aligned box |
| `hypercube` | `shape.Cuboid` after studio adaptation | Studio-only | N-dimensional equal-sided box authoring shortcut |
| `sphere` | `shape.Sphere` | Supported | 3D sphere, also used as N-dimensional sphere when dimension changes |
| `hypersphere` | `shape.Sphere` | Supported | N-dimensional sphere alias |
| `circle` | `shape.Circle` | Supported | Finite disk in a plane |
| `cylinder` | `shape.FiniteCylinder` | Supported | Closed finite cylinder; alias of `finite cylinder` |
| `finite cylinder` | `shape.FiniteCylinder` | Supported | Closed finite cylinder with caps |
| `triangle` | `shape.Triangle` | Supported | Single triangle primitive; STL is expanded into these |
| `quadratic equation` | `shape.QuadraticEquation` | Supported | 3D implicit quadratic surface |
| `cubic equation` | `shape.CubicEquation` | Supported | 3D implicit cubic algebraic surface |
| `four-order equation` | `shape.FourOrderEquation` | Supported | 3D implicit quartic algebraic surface |
| `implicit equation` | `shape.ImplicitEquation` | Supported | Registered non-polynomial scalar fields such as torus and gyroid |
| `polynomial surface` | `shape.PolynomialSurface` | Supported | Sparse arbitrary-degree implicit or explicit polynomial surface |
| `stl` | many `shape.Triangle` values | Supported | ASCII or binary STL mesh import |
| `plane` | `shape.Plane` exists | Recognized but rejected by JSON factory | Infinite mathematical plane, not currently scene-loadable |

## Common Object Fields

| Field | Required | Applies to | Notes |
| --- | --- | --- | --- |
| `id` | Recommended by scene merge rules | All objects | Duplicate object ids are rejected during scene composition. |
| `shape` | Yes | All objects | Must match one of the JSON shape values above. |
| `material_id` | Yes for renderable scene objects | All objects | Resolved by the object factory against `materials`. |
| `medium_boundary` | Optional | Closed boundary objects | Material/medium feature, not a geometry field. Avoid using it on open clipped surfaces. |
| `bounds` | Optional for most, required for `implicit equation` | Most shapes | Clips hit search to an axis-aligned box. Engine accepts canonical `bounds.pmin` + `bounds.pmax`; studio accepts authoring `bounds.center` + `bounds.size` and may inherit bounds through groups. |

`bounds` clips intersections but does not add cap faces. For example, a bounded
paraboloid remains an open clipped surface, not a closed solid.

## Shape Field Table

| Shape | Engine canonical fields | Studio authoring conveniences | Dimension notes | Intersection and normal |
| --- | --- | --- | --- | --- |
| `cuboid` | `pmin` + `pmax` | `center`/`position` + `size`, group placement | Uses current `render.dimension`; works for 3D and N-dimensional scenes. | Slab interval test; normal is the hit face axis. |
| `hypercuboid` | `pmin` + `pmax` | Same as `cuboid` | Alias intended for N-dimensional scenes. | Same as `cuboid`. |
| `hypercube` | Not accepted directly by engine | Studio validates equal local side lengths and emits `cuboid` | Alias intended for N-dimensional authoring. | Same as `cuboid` after adaptation. |
| `sphere` | `center`, `r` | Group field inheritance only; engine still accepts `position` as legacy center alias | Uses vector length of center/ray; works as N-dimensional sphere when scene dimension changes. | Quadratic ray-sphere solve; normal is normalized `hit - center`. |
| `hypersphere` | Same as `sphere` | Same as `sphere` | Alias intended for N-dimensional scenes. | Same as `sphere`. |
| `circle` | `center`, `normal`, `r` | Engine still accepts `position` as legacy center alias | Uses vector operations, but conceptually a finite disk. | Ray-plane solve plus radius check; normal is the normalized disk normal. |
| `cylinder` | `center`, `axis`, `r`, `height` | Engine still accepts `position` as legacy center alias | Alias of `finite cylinder`. | Side quadratic plus two cap disk tests; normal is side radial direction or cap axis. |
| `finite cylinder` | `center`, `axis`, `r`, `height` | Same as `cylinder` | Same implementation as `cylinder`. | Same as `cylinder`. |
| `triangle` | `p1`, `p2`, `p3` | Object `center` is baked by studio into points | Current normal/intersection path relies on 3D cross products; treat as 3D-only. | Moller-Trumbore style barycentric test; normal from edge cross product. |
| `quadratic equation` | `a` length 9, `b`, `c` | `center`/`scale` are baked by studio into coefficients | Effectively 3D: `a` is parsed as a 3 x 3 matrix. | Solves ray-substituted quadratic; normal is gradient `2Ax + b`. |
| `cubic equation` | `a` or `A` with 64 coefficients | `center`/`scale` are baked by studio into coefficients | 3D algebraic surface using basis indices `0=1`, `1=x`, `2=y`, `3=z`. | Substitutes ray into cubic polynomial; normal from tensor gradient. |
| `four-order equation` | `a` or `A` with 256 coefficients | `center`/`scale`/`basis` are baked by studio into coefficients | 3D algebraic surface using basis indices `0=1`, `1=x`, `2=y`, `3=z`. | Substitutes ray into quartic polynomial; normal from tensor gradient. |
| `implicit equation` | `field.type` or legacy `function`, plus `bounds` | Pass-through today | Current registered fields are 3D. Bounds are required. | Clips to bounds, scans along ray, detects sign changes, refines by bisection; normal from registered or numerical gradient. |
| `polynomial surface` | `input_dim`, `degree`, `coefficients.terms`, optional `transform` | `center`/`scale`/`basis` are combined into `transform` by studio | Ray intersection currently requires at least 3 ray dimensions; common use is 3D. | Builds a one-variable ray polynomial and solves real roots; normal from sparse polynomial gradient. |
| `stl` | `file`, `center`, `z_dir`, `x_dir`, `scale` | Pass-through today | 3D mesh import. | Parses ASCII or binary STL facets, transforms vertices, emits triangles. |
| `plane` | Source type has `A`, `b` | None through JSON | `shape.Plane` exists in code, but `ParseShape` returns an error for JSON `plane`. | Infinite plane ray solve exists in code; not scene-loadable until factory parsing is implemented. |

## Algebraic Coefficient Formats

| Shape | Dense form | Sparse form | Index meaning |
| --- | --- | --- | --- |
| `quadratic equation` | `a` is 9 numbers, row-major 3 x 3; `b` is a vector; `c` is scalar | Not supported by the quadratic parser | Surface is `x^T A x + b^T x + c = 0`. |
| `cubic equation` | `a` / `A` is 64 numbers | Object keys may be flat indices like `"21"` or coordinate keys like `"1, 1, 1"` | Each tensor axis uses `0=1`, `1=x`, `2=y`, `3=z`. |
| `four-order equation` | `a` / `A` is 256 numbers | Object keys may be flat indices or coordinate keys like `"1, 1, 1, 1"` | Same basis as cubic, with four tensor axes. |
| `polynomial surface` | Not a flat dense list | `coefficients.terms` contains `{ "index": [...], "value": number }` entries | For implicit mode, index is the monomial exponent tuple. For `output_dim > 1`, the first index is the output channel. |

Sparse cubic/four-order coefficient objects cannot mix flat keys and coordinate
keys in the same shape.

## Implicit Field Registry

| Field type | Parameters | Defaults | Validation |
| --- | --- | --- | --- |
| `torus` | `major_radius`, `minor_radius` | `major_radius = 0.58`, `minor_radius = 0.22` | Both must be positive; `minor_radius < major_radius`. |
| `gyroid` | `frequency`, `offset` | `frequency = 3.2`, `offset = 0.0` | `frequency` must be positive. |

Preferred form:

```json
{
  "shape": "implicit equation",
  "field": {
    "type": "gyroid",
    "frequency": 3.2,
    "offset": 0
  },
  "bounds": {
    "pmin": [-1, -1, -1],
    "pmax": [1, 1, 1]
  }
}
```

Legacy form is still accepted:

```json
{
  "shape": "implicit equation",
  "function": "torus",
  "bounds": {
    "pmin": [-1, -1, -1],
    "pmax": [1, 1, 1]
  }
}
```

## Bounds Details

| Form | Example | Current parser status |
| --- | --- | --- |
| `pmin` + `pmax` | `"bounds": { "pmin": [-1, -1, -1], "pmax": [1, 1, 1] }` | Engine canonical form. Each `pmin[i]` must be smaller than `pmax[i]`. |
| `center` + `size` | `"bounds": { "center": [0, 0, 0], "size": [2, 2, 2] }` | Studio authoring form. Studio emits `pmin` + `pmax`; each size component must be positive. |
| `position` + `size` inside `bounds` | `"bounds": { "position": [0, 0, 0], "size": [2, 2, 2] }` | Studio authoring alias for `center` + `size`. |

For object placement outside `bounds`, several primitive parsers do accept
`position` as a center alias: `cuboid`, `sphere`, `circle`, `cylinder`, and
`finite cylinder`.

## Scene Editor Coverage

The React scene editor currently exposes only this narrower shape union:

```text
cuboid
sphere
triangle
plane
quadratic equation
four-order equation
```

This editor list is not the renderer's full support list. In particular, the
editor includes `plane`, while the Go JSON factory currently rejects `plane`.
The renderer supports additional shapes such as `circle`, `cylinder`,
`cubic equation`, `implicit equation`, `polynomial surface`, STL meshes, and
the N-dimensional aliases.
