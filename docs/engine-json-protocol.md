# Engine JSON Protocol

This file describes the JSON protocol consumed directly by `engine`.

`engine` is the execution layer. It should receive normalized scene JSON that is
ready to parse into cameras, media, materials, objects, shapes, and render jobs.
Authoring conveniences belong in `studio`; `engine` expects the canonical forms
below.

## Command

```bash
npm run ray -- --script path/to/scene.json
go -C engine run . --script ../path/to/scene.json
```

Engine accepts exactly one `--script`. Use `studio` when multiple authoring
scripts need to be merged before execution. Render fields can be overridden from
the CLI:

```text
--dimension
--camera-index
--threads
--width
--height
--samples
--output-film
--exposure
--tone-mapping
--gamma
--spectrum-mode
--wavelength-samples
--working-space
```

## Top Level

```json
{
  "media": {},
  "materials": [],
  "objects": [],
  "cameras": [],
  "render": {},
  "renders": []
}
```

`engine` does not resolve `includes` or merge scene files. Use `studio` for
authoring composition; it writes one normalized intermediate JSON file for
engine execution.

`engine` only writes Film data through `output_film`. It does not read
`resume_film` and does not write image files. Use `studio` when a Film should be
resumed, merged, converted to an image, or checkpointed over multiple runs.

Unknown fields are ignored unless a parser explicitly reads them, but canonical
engine JSON should omit authoring-only fields such as `includes`.

## Objects

Each object is a renderable geometry instance:

```json
{
  "id": "glass-ball",
  "shape": "sphere",
  "center": [0, 0, 0],
  "r": 1,
  "material_id": "glass",
  "medium_boundary": {
    "outside": "air",
    "inside": "glass",
    "priority": 10,
    "thin": false
  }
}
```

`engine` does not parse `shape: "group"`. Group expansion must happen before
the JSON reaches `engine`.

## Shape Protocol

Canonical shape fields for normalized engine JSON:

| Shape | Canonical fields |
| --- | --- |
| `cuboid`, `hypercuboid` | `pmin`, `pmax` |
| `sphere`, `hypersphere` | `center`, `r` |
| `circle` | `center`, `normal`, `r` |
| `cylinder`, `finite cylinder` | `center`, `axis`, `r`, `height` |
| `triangle` | `p1`, `p2`, `p3` |
| `quadratic equation` | `a` length 9, `b` length `render.dimension`, `c` |
| `cubic equation` | `a` length 64, or sparse `a`/`A` object |
| `four-order equation` | `a` length 256, or sparse `a`/`A` object |
| `polynomial surface` | `mode`, `input_dim`, `degree`, `coefficients` |
| `implicit equation` | `field`, `bounds` |
| `stl` | `file`, `center`, `z_dir`, `x_dir`, `scale` |

`plane` is recognized but intentionally returns an error because it is declared
but not implemented.

`hypercube` is a studio authoring shape, not an engine shape. Studio validates
equal local side lengths and emits a canonical `cuboid`.

### Bounds

Objects may declare `bounds` to clip intersections to an axis-aligned cuboid.
Bounds clip the surface only; they do not add cap faces and should not be used
to imply a closed dielectric container.

Canonical form:

```json
{
  "bounds": {
    "pmin": [-1, -1, -1],
    "pmax": [1, 1, 1]
  }
}
```

Authoring forms such as `bounds.center` + `bounds.size` belong in `studio`.
Engine JSON must use `bounds.pmin` + `bounds.pmax`.

### Polynomial Equations

For cubic and four-order equations, tensor index `0` is the constant factor `1`,
while `1`, `2`, and `3` are `x`, `y`, and `z`.

`quadratic equation`, `cubic equation`, and `four-order equation` are expected
to arrive with baked world-space coefficients. `studio` is responsible for
applying authoring `center`/`scale`/`basis` before invoking `engine`.

### Polynomial Surface

`polynomial surface` is the generic sparse polynomial shape. It represents one
implicit zero level set `F(x,y,z)=0`.

```json
{
  "shape": "polynomial surface",
  "input_dim": 3,
  "degree": 2,
  "transform": [
    [1, 0, 0, 0],
    [0, 1, 0, 0],
    [0, 0, 1, 0],
    [0, 0, 0, 1]
  ],
  "coefficients": {
    "format": "coo",
    "shape": [3, 3, 3],
    "terms": [
      { "index": [2, 0, 0], "value": 1 },
      { "index": [0, 2, 0], "value": 1 },
      { "index": [0, 0, 2], "value": 1 },
      { "index": [0, 0, 0], "value": -1 }
    ]
  }
}
```

### Implicit Equation

`implicit equation` uses an expression field:

```json
{
  "shape": "implicit equation",
  "field": {
    "type": "expr",
    "expr": "sin(f*x)*cos(f*y) + sin(f*y)*cos(f*z) + sin(f*z)*cos(f*x) - offset",
    "constants": {
      "f": 3.2,
      "offset": 0
    }
  },
  "bounds": {
    "pmin": [-1, -1, -1],
    "pmax": [1, 1, 1]
  },
  "step": 0.01,
  "value_tol": 1e-7
}
```

Supported fields:

```text
expr: expr, constants, optional gradient
```

For `field.type: "expr"`, the expression is evaluated in local coordinates
`x`, `y`, and `z`, and the surface is the zero level set. `constants` may define
finite numeric symbols. If `gradient` is omitted, the engine tries symbolic
automatic differentiation for supported smooth expressions before falling back
to finite differences. Optional `gradient.x`, `gradient.y`, and `gradient.z`
provide local partial derivatives for analytic normals:

```json
{
  "shape": "implicit equation",
  "field": {
    "type": "expr",
    "expr": "x*x + y*y + z*z - r*r",
    "constants": { "r": 1 },
    "gradient": {
      "x": "2*x",
      "y": "2*y",
      "z": "2*z"
    }
  },
  "bounds": {
    "pmin": [-1.2, -1.2, -1.2],
    "pmax": [1.2, 1.2, 1.2]
  }
}
```

The legacy top-level `"function"` field and built-in implicit field names such
as `torus` and `gyroid` are not accepted. Use `field.type: "expr"`.

## Materials, Media, Cameras, Render

Engine camera JSON must already be normalized. A 3D, hyperbolic, or Klein
camera requires an explicit `direction`; default camera values are studio
authoring features.

```json
{
  "type": "3d",
  "position": [-4, 0, 1],
  "direction": [4, 0, -1],
  "up": [0, 0, 1],
  "field_of_views": [60, 60],
  "ortho": false
}
```

If the scene has no cameras, engine returns an error. Use studio to generate the
default authoring camera.

Materials, media, cameras, and render settings are shared with studio input.
They are parsed by:

```text
engine/controller/factory/materials.go
engine/controller/factory/media.go
engine/controller/factory/cameras.go
engine/controller/render_config.go
```

For detailed material and renderer behavior, see:

- [`material-system-design.md`](material-system-design.md)
- [`material-capability-coverage.md`](material-capability-coverage.md)
- [`current-renderer-architecture.md`](current-renderer-architecture.md)
