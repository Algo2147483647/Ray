# Current Scene JSON

This document records the current scene JSON fields used by the Go renderer after the controller/parser/factory and BSDF/BxDF refactor.

## Top-Level Shape

```json
{
  "includes": [],
  "media": {},
  "materials": [],
  "objects": [],
  "cameras": [],
  "render": {},
  "renders": []
}
```

Use `cameras` for camera definitions. If the list is omitted or empty, the render handler creates a default 3D camera at render time. 3D, hyperbolic, and spherical cameras use `aspect_ratio` for their frame shape; `width` and `height` belong only to `render` output settings. N-dimensional cameras keep `widths` because those define the sampled film axes.

## Composing Multiple JSON Files

Scene files may include other scene JSON files. Include paths are resolved
relative to the JSON file that declares them. Included files are loaded first,
then the including file is merged on top:

```json
{
  "includes": [
    "studio.json",
    "geometry-heart.json",
    "materials-red.json"
  ],
  "render": {
    "samples": 64,
    "width": 960,
    "height": 960
  }
}
```

Merge rules:

```text
media: merged by id; duplicate ids fail
materials: appended by id; duplicate ids fail
objects: appended by id; duplicate ids fail
cameras: appended by id when present; duplicate camera ids fail
render: scalar render fields inherit from includes and are overridden by later files
renders: appended as separate render jobs
```

The CLI also accepts repeated `--script` flags. They are merged in the order
provided, using the same duplicate-id rules:

```bash
npm run ray -- --script studio.json --script geometry-heart.json --script renders.json
```

## Multiple Render Jobs

Use `renders` when one merged scene should produce multiple outputs. Each item
inherits defaults from the top-level `render` block and overrides only the
fields it declares:

```json
{
  "render": {
    "samples": 128,
    "width": 960,
    "height": 960,
    "thread_num": 8
  },
  "renders": [
    {
      "camera_index": 0,
      "output_image": "../outputs/front.png"
    },
    {
      "camera_index": 1,
      "samples": 512,
      "output_image": "../outputs/detail.png"
    }
  ]
}
```

When `renders` is omitted, the renderer behaves as before and runs the single
top-level `render` job.

## Media

`media` is optional. When omitted, the renderer creates a default `air` medium with IOR 1. Named media are used by object-level `medium_boundary` blocks so refraction can use explicit incident/transmitted media instead of only a scalar ray IOR.

```json
{
  "media": {
    "glass": {
      "type": "homogeneous",
      "ior": {
        "type": "constant",
        "eta": 1.5
      }
    },
    "dispersive_glass": {
      "type": "homogeneous",
      "ior": {
        "type": "cauchy",
        "a": 1.5046,
        "b": 0.0042,
        "c": 0
      }
    }
  }
}
```

Supported medium fields:

```text
type: homogeneous
ior: constant or cauchy IOR object
sigma_a: optional spectral absorption coefficient, applied as Beer-Lambert transmittance between surface hits
sigma_s: optional spectral scattering coefficient, parsed and stored but not yet sampled as participating-media scattering
```

`sigma_a` is evaluated in RGB or spectral context and applied during ray transport as `exp(-sigma_a * distance)`. `sigma_s` is part of the schema and medium model, but distance sampling, phase functions, and in-medium scattering events are not implemented yet.

## Materials

Each material requires an `id` and at least one of `surface` or `emission`.
For a support matrix that maps these JSON material types to real-world material
families, see [`material-capability-coverage.md`](material-capability-coverage.md).

### Lambert

```json
{
  "id": "matte",
  "surface": {
    "type": "lambert",
    "albedo": [0.8, 0.8, 0.8]
  }
}
```

### Specular Reflection

```json
{
  "id": "mirror",
  "surface": {
    "type": "specular_reflection",
    "reflectance": [1, 1, 1]
  }
}
```

`reflectance` is optional and defaults to `[1, 1, 1]`.

### Specular Dielectric

```json
{
  "id": "glass",
  "surface": {
    "type": "specular_dielectric",
    "reflectance": [1, 1, 1],
    "transmittance": [1, 1, 1],
    "eta_outside": 1,
    "ior": {
      "type": "constant",
      "eta": 1.5
    }
  }
}
```

`reflectance`, `transmittance`, and `eta_outside` are optional. `eta_inside` is still accepted as shorthand for constant IOR, but new scenes should prefer the explicit `ior` block.

### Cauchy Dispersion

```json
{
  "id": "dispersive-glass",
  "surface": {
    "type": "specular_dielectric",
    "eta_outside": 1,
    "ior": {
      "type": "cauchy",
      "a": 1.5046,
      "b": 0.0042,
      "c": 0
    }
  }
}
```

The Cauchy equation uses wavelength in micrometers:

```text
eta(lambda_nm) = A + B / lambda_um^2 + C / lambda_um^4
```

Renderer-level spectral sampling chooses one wavelength per camera path and propagates it through the path.

When an object has a `medium_boundary`, the boundary media provide the incident/transmitted eta for this BxDF. Without `medium_boundary`, legacy `eta_outside`, `eta_inside`, and `ior` behavior remains active for compatibility.

## Objects And Medium Boundaries

Objects may declare the media separated by a closed dielectric boundary:

```json
{
  "id": "glass-sphere",
  "shape": "sphere",
  "position": [0, 0, 0],
  "r": 1,
  "material_id": "glass",
  "medium_boundary": {
    "outside": "air",
    "inside": "dispersive_glass",
    "priority": 10,
    "thin": false
  }
}
```

`outside` defaults to `air`. `inside` is required when `medium_boundary` is present. `priority` participates in medium-stack overlap resolution. `thin: true` marks a non-container boundary and disables medium-stack mutation.

Objects may also declare an optional `bounds` block to clip the visible portion
of a shape to an axis-aligned box. This is primarily useful for finite previews
of otherwise open or unbounded implicit surfaces such as paraboloids,
hyperboloids, cones, cubic sheets, and quartic sheets:

```json
{
  "id": "bounded-paraboloid",
  "shape": "quadratic equation",
  "a": [10, 0, 0, 0, 10, 0, 0, 0, 0],
  "b": [0, 0, -3],
  "c": 0,
  "bounds": {
    "pmin": [-0.3, -0.3, 0],
    "pmax": [0.3, 0.3, 0.4]
  },
  "material_id": "matte"
}
```

`bounds` accepts either `pmin`/`pmax` or `position`/`size`. It clips only the
surface intersection and does not add cap faces, so open clipped surfaces should
not be treated as closed dielectric medium boundaries.

Implicit polynomial surfaces support these JSON shapes:

```text
quadratic equation: 9 coefficients for x^T A x + b^T x + c
cubic equation: 64 coefficients for A[i][j][k] factors
four-order equation: 256 coefficients for A[i][j][k][l] factors
polynomial surface: sparse arbitrary-degree coefficients
```

For cubic and four-order equations, tensor index `0` is the constant factor `1`,
while `1`, `2`, and `3` are `x`, `y`, and `z`.

Quadratic, cubic, and four-order equation objects may declare optional `center`
and `scale` fields. These fields transform the local polynomial coordinates
before rendering:

```text
local = (world - center) / scale
```

The parser bakes this transform into the polynomial coefficients at scene load
time, so ray intersection still evaluates only the final stored coefficients.
`scale` may be either a single positive number or a 3-value vector. `bounds`
remain a world-space clipping box and are not transformed by `center` or
`scale`.

`polynomial surface` is the generic sparse polynomial shape. It stores
coefficients in a sparse tensor under `coefficients.terms`, where each `index`
is the exponent tuple `[alpha_x, alpha_y, alpha_z]` for the monomial:

```text
c * x^alpha_x * y^alpha_y * z^alpha_z
```

The shape currently supports `mode: "implicit"` for `F(x,y,z)=0` and
`mode: "explicit"` for `z=P(x,y)` by default. The `center`, `scale`, and
optional `basis` fields transform world coordinates into local polynomial
coordinates at evaluation time. `basis` is an orthonormal list of local axis
directions in world space:

```text
local_i = dot(world - center, basis_i) / scale_i
```

If `basis` is omitted, it defaults to the identity basis. `bounds` remain a
world-space clipping box and are not transformed by `center`, `scale`, or
`basis`.

Example Barth sextic surface:

```json
{
  "id": "barth-sextic",
  "shape": "polynomial surface",
  "mode": "implicit",
  "input_dim": 3,
  "degree": 6,
  "center": [-0.9, 1.68, 0.38],
  "scale": 0.2,
  "basis": [
    [0.8660254037844386, 0, 0.5],
    [0, 1, 0],
    [-0.5, 0, 0.8660254037844386]
  ],
  "coefficients": {
    "format": "coo",
    "shape": [7, 7, 7],
    "degree_policy": "total",
    "terms": [
      { "index": [4, 2, 0], "value": -27.41640786499874 },
      { "index": [4, 0, 2], "value": 10.47213595499958 },
      { "index": [4, 0, 0], "value": -4.23606797749979 },
      { "index": [2, 4, 0], "value": 10.47213595499958 },
      { "index": [2, 2, 2], "value": 67.77708763999664 },
      { "index": [2, 2, 0], "value": -8.47213595499958 },
      { "index": [2, 0, 4], "value": -27.41640786499874 },
      { "index": [2, 0, 2], "value": -8.47213595499958 },
      { "index": [2, 0, 0], "value": 8.47213595499958 },
      { "index": [0, 4, 2], "value": -27.41640786499874 },
      { "index": [0, 4, 0], "value": -4.23606797749979 },
      { "index": [0, 2, 4], "value": 10.47213595499958 },
      { "index": [0, 2, 2], "value": -8.47213595499958 },
      { "index": [0, 2, 0], "value": 8.47213595499958 },
      { "index": [0, 0, 4], "value": -4.23606797749979 },
      { "index": [0, 0, 2], "value": 8.47213595499958 },
      { "index": [0, 0, 0], "value": -4.23606797749979 }
    ]
  },
  "bounds": {
    "position": [-0.9, 1.68, 0.38],
    "size": [0.4, 0.4, 0.4]
  },
  "material_id": "matte"
}
```

Non-polynomial implicit surfaces use `shape: "implicit equation"` with a
registered field type and field-specific parameters. The renderer owns the
field registry and numeric intersection logic; scene JSON owns the selected
field, parameters, placement, bounds, and tolerances:

```json
{
  "id": "gyroid-cell",
  "shape": "implicit equation",
  "field": {
    "type": "gyroid",
    "frequency": 3.2,
    "offset": 0.0
  },
  "bounds": {
    "position": [0, 0, 0],
    "size": [2, 2, 2]
  },
  "step": 0.01,
  "value_tol": 1e-7,
  "material_id": "matte"
}
```

Supported built-in implicit fields:

```text
torus: major_radius, minor_radius
gyroid: frequency, offset
```

The legacy top-level `"function": "torus"` / `"function": "gyroid"` form is
still accepted, but new scenes should prefer the `field.type` form.

### Rough Conductor

```json
{
  "id": "rough-metal",
  "surface": {
    "type": "rough_conductor",
    "eta": [0.17, 0.35, 1.5],
    "k": [3.1, 2.7, 1.9],
    "roughness": 0.2
  }
}
```

`roughness` is optional and defaults to `0.25`. Internally the parser converts it to microfacet alpha with `alpha = roughness^2`.

### Rough Dielectric Transmission

```json
{
  "id": "frosted-glass",
  "surface": {
    "type": "rough_dielectric_transmission",
    "transmittance": [0.9, 0.95, 1],
    "eta_outside": 1,
    "ior": {
      "type": "constant",
      "eta": 1.5
    },
    "roughness": 0.35
  }
}
```

`transmittance`, `eta_outside`, and `roughness` are optional. Like `rough_conductor`,
the parser converts user-facing roughness to microfacet alpha with
`alpha = roughness^2`. Use an object-level `medium_boundary` when this surface
should update the ray's medium stack while transmitting.

### Constant Emission

```json
{
  "id": "light",
  "emission": {
    "type": "constant",
    "color": [8, 8, 8]
  }
}
```

New scenes may also use `radiance`; `color` remains accepted as a legacy alias:

```json
{
  "id": "warm-light",
  "emission": {
    "type": "constant",
    "radiance": {
      "type": "blackbody",
      "temperature": 3000,
      "scale": 8
    }
  }
}
```

## Spectrum Parameters

Material and emission spectrum fields accept the legacy array form:

```json
"albedo": [0.8, 0.8, 0.8]
```

This is interpreted as scene-linear sRGB and is equivalent to:

```json
"albedo": {
  "type": "rgb",
  "space": "linear_srgb",
  "value": [0.8, 0.8, 0.8]
}
```

Supported object forms:

```json
{
  "type": "rgb",
  "space": "srgb",
  "value": [0.8, 0.6, 0.4]
}
```

```json
{
  "type": "constant",
  "value": 0.8
}
```

```json
{
  "type": "sampled",
  "wavelengths_nm": [400, 500, 600, 700],
  "values": [0.1, 0.4, 0.8, 0.7],
  "interpolation": "linear"
}
```

```json
{
  "type": "blackbody",
  "temperature": 6500,
  "scale": 1
}
```

## Render Output

```json
{
  "render": {
    "dimension": 3,
    "samples": 1000,
    "thread_num": 0,
    "camera_index": 0,
    "width": 800,
    "height": 800,
    "output_image": "../../outputs/image.png",
    "output_film": "../../outputs/image.bin",
    "exposure": 2.1,
    "tone_mapping": "reinhard",
    "gamma": 2.2,
    "spectrum_mode": "hero_wavelength",
    "wavelength_samples": 1,
    "color_space": "acescg"
  }
}
```

`dimension` selects the scene's runtime spatial dimension and defaults to `3`.
Set it to `4` for 4D experiments such as hypercube and hypersphere scenes. All
vectors in object and camera definitions must then have four components.

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

`hero_wavelength` is the default to preserve the renderer's current behavior. `rgb` disables camera wavelength sampling for fast compatibility renders. `sampled` traces multiple wavelength sub-samples per camera sample; set `wavelength_samples` to control the sub-sample count. If `sampled` is selected and `wavelength_samples` is omitted, the renderer defaults to 4 wavelength sub-samples.

Camera ray generation now only creates geometric rays. Wavelength sampling is owned by the renderer/integrator so that `rgb`, `hero_wavelength`, and `sampled` modes share the same camera code.

Film stores three display/work channels in an explicit color space and can also retain spectral bins. `rgb` mode accumulates directly in the selected film working space. `hero_wavelength` and `sampled` modes first accumulate scalar wavelength samples into spectral bins, then convert the bins once through the CIE 1931 XYZ matching functions and into the selected film working space before image output.

The code distinguishes authored RGB input spaces from film storage spaces. Material RGB parameters use `optics.RGBColorSpace` values such as `linear_srgb`, `srgb`, and `acescg`. Film accumulation uses `camera.FilmColorSpace`: `linear_srgb`, `acescg`, or `xyz`. `acescg` is the recommended wide-gamut working space for spectral renders that still need a three-channel film/output path.

`working_space` is accepted as a legacy alias for `color_space`; prefer `color_space` in new scene files.

`optics.Spectrum` preserves optional sampled channels in addition to its RGB compatibility fields. Lambert, specular reflection, specular dielectric, rough conductor, and constant emission evaluate their spectral parameters from `bxdf.ShadingContext` instead of collapsing all inputs to RGB at parse time. In `sampled` mode the renderer still traces wavelength sub-paths independently, which is required for dispersive paths where different wavelengths can refract in different directions. The sampled-channel `Spectrum` path is active in material evaluation and unit tests, and is the compatibility layer for a future packet-style spectral integrator.

CLI overrides:

```bash
go -C engine run . --script ../examples/scenes/feature-showcase.json --width 800 --height 800 --samples 2000 --exposure 2.1 --tone-mapping reinhard --gamma 2.2
```

## N-Dimensional Cameras and 4D Shapes

Use `type: "n_dim"` when the camera observes an N-dimensional scene. The camera
requires:

```text
position: N values
coordinates: width-count + 1 vectors, each with N values
widths: film resolution for each sampled camera axis
field_of_views: one value per width, or field_of_view as a shared fallback
ortho: optional orthographic mode
```

Example 4D camera with a 3D film:

```json
{
  "id": "outside-4d-orthographic-geometry-camera",
  "type": "n_dim",
  "position": [-3.97836, -0.716105, 0.477403, -1.67091],
  "coordinates": [
    [1.0, 0.18, -0.12, 0.42],
    [0.24, 1.0, 0.12, -0.58],
    [-0.32, 0.08, 1.0, 0.36],
    [0.42, -0.50, 0.34, 1.0]
  ],
  "widths": [100, 100, 100],
  "field_of_views": [120, 120, 120],
  "ortho": true
}
```

The current higher-dimensional shape aliases are:

```text
hypercube: equal-sided N-dimensional cuboid
hypercuboid: N-dimensional cuboid
hypersphere: N-dimensional sphere
```

Example:

```json
{
  "id": "main-hypercube",
  "shape": "hypercube",
  "center": [0, 0, 0, 0],
  "size": [1.65, 1.65, 1.65, 1.65],
  "material_id": "cell-palette-debug"
}
```

`cell_palette` emission is useful for hypercube geometry because it colors cells
by the dominant normal axis. In 4D, this gives separate colors to the eight
cubic cells of the hypercube boundary:

```json
{
  "id": "cell-palette-debug",
  "emission": {
    "type": "cell_palette",
    "shading": "solid",
    "intensity": 1.0
  }
}
```

## Legacy Material Fields

These old fields are not part of the current material schema and are intentionally not translated by the new parser:

```text
color
reflectivity
refractivity
radiate
diffuse_loss
```

Use `surface` and `emission` blocks instead.

## Current Showcase Scenes

```text
examples/scenes/feature-showcase.json
examples/scenes/material-benchmark-matrix.json
examples/scenes/geometry-benchmark-matrix.json
examples/scenes/dispersion-three-balls.json
examples/scenes/prism refraction.json
examples/scenes/true-spectral-prism-dispersion-200spp.json
examples/scenes/spectral-acescg-prism-showcase.json
examples/scenes/4d-hypercube/4d-hypercube-geometry-focus.json
examples/scenes/4d-hypercube/4d-hypercube-geometry-focus-direct.json
```

`feature-showcase.json` exercises Lambert color bleeding, constant emission, specular reflection, constant-IOR glass, Cauchy-dispersive glass, GGX rough conductor, spectral sampling, and tone-mapped PNG output.
`material-benchmark-matrix.json` is a controlled gray studio box with a stepped arc of material sample spheres for comparing diffuse, mirror, rough conductor, ideal dielectric, rough dielectric transmission, absorbing media, and emissive materials.
`geometry-benchmark-matrix.json` reuses the controlled gray studio layout with a 6x7 sample matrix covering sphere, cuboid, cylinder, circle, triangle, quadratic-equation, and four-order-equation geometry.
The `4d-hypercube` scenes exercise `render.dimension: 4`, `n_dim` cameras, hypercube/hypersphere geometry, Lambertian exterior observation, and cell-palette cell decomposition.
