# Controller JSON Rules

This document describes the JSON consumed by `engine/controller/parser` and `engine/controller/factory`.

## Entry Points

```text
engine/controller/parser/schema.go
engine/controller/parser/script.go
engine/controller/factory/scene.go
engine/controller/factory/materials.go
engine/controller/factory/media.go
engine/controller/factory/cameras.go
engine/controller/factory/shapes.go
```

The controller layer owns file loading, command-line render overrides, and conversion from JSON data into `model.Scene`.

## Top-Level Shape

```json
{
  "media": {},
  "materials": [],
  "objects": [],
  "cameras": [],
  "render": {}
}
```

Top-level fields:

| Field | Type | Notes |
| --- | --- | --- |
| `media` | object | Optional named homogeneous media. A default `air` medium is always created. |
| `materials` | array | Material definitions. Each material needs an `id` and at least one of `surface` or `emission`. |
| `objects` | array | Geometry instances with `shape`, `material_id`, and optional `medium_boundary`. |
| `cameras` | array | Optional camera definitions. If omitted, the render handler creates a default 3D camera. |
| `render` | object | Dimension, samples, output paths, tone mapping, spectrum mode, and camera selection. |

## Load Flow

1. `ReadScriptFile(path)` unmarshals JSON into `parser.Script`.
2. `LoadSceneFromScript(script, scene)` resets the scene object tree and camera list.
3. `render.dimension` sets `utils.Dimension`; omitted or non-positive means `3`.
4. `ParseMaterials` builds an `id -> *material.Material` map.
5. `ParseMediaRegistry` creates the default `air` medium and user-defined media.
6. Each object resolves `material_id`, parses one or more shapes, parses `medium_boundary`, then adds objects to the BVH input list.
7. `ParseCameras` builds camera instances from `cameras`.
8. `scene.ObjectTree.Build()` builds the BVH.

The factory accumulates contextual parse errors with labels such as `object[2] id="glass-sphere"` instead of silently skipping invalid scene entries.

## Materials

Supported `surface.type` values:

```text
lambert
specular_reflection
specular_dielectric
rough_conductor
rough_dielectric_transmission
```

Supported `emission.type` values:

```text
constant
```

The current material model is BSDF/BxDF based:

```text
engine/model/material/material.go
engine/model/material/bsdf/
engine/model/material/bxdf/
engine/model/material/emission/
engine/model/material/medium/
engine/model/material/microfacet/
```

For the current coverage of real-world material families and implemented BxDFs,
see [`material-capability-coverage.md`](material-capability-coverage.md).

Legacy fields such as `color`, `reflectivity`, `refractivity`, `radiate`, and `diffuse_loss` are not translated by the current parser. Use `surface` and `emission` blocks instead.

## Spectral Parameters

Spectrum-valued fields accept either legacy RGB arrays:

```json
"albedo": [0.8, 0.4, 0.2]
```

or typed objects:

```json
{ "type": "rgb", "space": "linear_srgb", "value": [0.8, 0.4, 0.2] }
{ "type": "constant", "value": 0.8 }
{ "type": "sampled", "wavelengths_nm": [400, 500, 600], "values": [0.1, 0.5, 0.9] }
{ "type": "blackbody", "temperature": 3000, "scale": 8 }
```

Supported RGB color spaces:

```text
linear_srgb
srgb
acescg
```

Implementation lives in `engine/model/optics/spectrum_parameter/`.

## Media and IOR

Named media are homogeneous and currently provide IOR for dielectric boundary transitions:

```json
{
  "media": {
    "glass": {
      "type": "homogeneous",
      "ior": { "type": "constant", "eta": 1.5 }
    },
    "prism": {
      "type": "homogeneous",
      "ior": { "type": "cauchy", "a": 1.5046, "b": 0.0042, "c": 0 }
    }
  }
}
```

`sigma_a` and `sigma_s` are parsed as spectral parameters. `sigma_a` is applied during transport as homogeneous Beer-Lambert absorption. `sigma_s` is parsed and stored for the medium model, but participating-media distance sampling and phase-function scattering are not implemented yet.

Objects can declare a boundary:

```json
{
  "medium_boundary": {
    "outside": "air",
    "inside": "glass",
    "priority": 10,
    "thin": false
  }
}
```

`inside` is required when `medium_boundary` is present. `outside` defaults to `air`.

## Shapes

Supported `shape` values:

```text
cuboid
sphere
circle
cylinder
finite cylinder
triangle
quadratic equation
cubic equation
four-order equation
polynomial surface
stl
```

`plane` is recognized but intentionally returns an error because it is declared but not implemented.

Important shape fields:

| Shape | Required fields |
| --- | --- |
| `cuboid` | Either `position` + `size`, or `pmin` + `pmax` |
| `sphere` | `position`, `r` |
| `circle` | `position`, `normal`, `r` |
| `cylinder` / `finite cylinder` | `position`, `axis`, `r`, `height` |
| `triangle` | `p1`, `p2`, `p3` |
| `quadratic equation` | `a` length 9, `b` length `render.dimension`, `c` |
| `cubic equation` | `a` length 64 |
| `four-order equation` | `a` length 256 |
| `polynomial surface` | `input_dim`, `degree`, `coefficients.terms` |
| `stl` | `file`, `position`, `z_dir`, `x_dir`, `scale` |

`stl` expands into many `triangle` shapes.

`polynomial surface` is a sparse arbitrary-degree polynomial shape. It accepts
`mode: "implicit"` for `F(x,y,z)=0` or `mode: "explicit"` for `z=P(x,y)`.
Coefficients are stored as sparse tensor terms:

```json
{
  "shape": "polynomial surface",
  "mode": "implicit",
  "input_dim": 3,
  "degree": 2,
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

The example above represents `x^2 + y^2 + z^2 - 1 = 0`.

## Cameras

Supported camera types:

```text
3d
camera3d
n_dim
ndim
n-dimensional
```

An omitted camera type defaults to `3d`.

3D camera fields:

```json
{
  "type": "3d",
  "position": [-4, 0, 1],
  "look_at": [0, 0, 0],
  "up": [0, 0, 1],
  "field_of_view": 60,
  "aspect_ratio": 1,
  "ortho": false
}
```

`direction` may be used instead of `look_at`.
`width` and `height` are render settings, not camera fields.

N-dimensional camera fields:

```json
{
  "type": "n_dim",
  "position": [0, 0, 0, 0],
  "widths": [800, 800],
  "field_of_views": [60, 60],
  "coordinates": [
    [1, 0, 0, 0],
    [0, 1, 0, 0],
    [0, 0, 1, 0]
  ]
}
```

For `n_dim`, `len(coordinates)` must equal `len(widths) + 1`.

## Render Settings

Common fields:

```json
{
  "render": {
    "dimension": 3,
    "samples": 100,
    "thread_num": 8,
    "camera_index": 0,
    "width": 800,
    "height": 800,
    "output_image": "../outputs/render.png",
    "output_film": "../outputs/render.bin",
    "resume_film": "",
    "exposure": 1,
    "tone_mapping": "linear",
    "gamma": 2.2,
    "spectrum_mode": "hero_wavelength",
    "wavelength_samples": 1,
    "color_space": "acescg"
  }
}
```

Supported tone mappers:

```text
linear
reinhard
aces
```

Supported spectrum modes:

```text
rgb
hero_wavelength
sampled
```

Supported film color spaces:

```text
linear_srgb
acescg
xyz
```

## Recommended Practices

- Prefer `cameras`, not legacy `camera`.
- Prefer explicit `surface`, `emission`, and `ior` objects.
- Keep `render.dimension` consistent with all vector lengths.
- Treat unknown fields as ignored unless the parser explicitly reads them.
- Put new parsing code under `engine/controller/factory`, not under renderer core packages.
