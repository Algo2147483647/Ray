# Current Scene JSON

This document records the current scene JSON fields used by the Go renderer after the BSDF/BxDF material migration.

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

`camera` is still accepted as a legacy alias for `cameras` when `cameras` is absent.

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
sigma_a: optional spectral absorption placeholder
sigma_s: optional spectral scattering placeholder
```

`sigma_a` and `sigma_s` are parsed now but homogeneous volume attenuation/scattering is not yet applied during transport.

## Materials

Each material requires an `id` and at least one of `surface` or `emission`.

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

`outside` defaults to `air`. `inside` is required when `medium_boundary` is present. `priority` is parsed for future overlap resolution. `thin: true` marks a non-container boundary and disables medium-stack mutation.

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
    "samples": 1000,
    "thread_num": 0,
    "camera_index": 0,
    "width": 800,
    "height": 800,
    "output_image": "../../outputs/image.png",
    "output_film": "../../outputs/image.bin",
    "debug_output": "../../outputs/debug.json",
    "exposure": 2.1,
    "tone_mapping": "reinhard",
    "gamma": 2.2,
    "spectrum_mode": "hero_wavelength",
    "wavelength_samples": 1,
    "working_space": "linear_srgb"
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

`hero_wavelength` is the default to preserve the renderer's current behavior. `rgb` disables camera wavelength sampling for fast compatibility renders. `sampled` traces multiple wavelength sub-samples per camera sample; set `wavelength_samples` to control the sub-sample count. If `sampled` is selected and `wavelength_samples` is omitted, the renderer defaults to 4 wavelength sub-samples.

Camera ray generation now only creates geometric rays. Wavelength sampling is owned by the renderer/integrator so that `rgb`, `hero_wavelength`, and `sampled` modes share the same camera code.

Film currently stores three channels in `linear_srgb` by default. Hero-wavelength reconstruction uses a CIE 1931 XYZ approximation converted to linear sRGB and white-point normalized before the contribution reaches the Film. The Film also has an `xyz` working-space path for future spectral accumulation, but the active renderer keeps `linear_srgb` until BSDF throughput is fully spectral.

`core.Spectrum` now preserves optional sampled channels in addition to its RGB compatibility fields. Lambert, specular reflection, specular dielectric, rough conductor, and constant emission evaluate their spectral parameters from `ShadingContext` instead of collapsing all inputs to RGB at parse time. In `sampled` mode the renderer still traces wavelength sub-paths independently, which is required for dispersive paths where different wavelengths can refract in different directions. The sampled-channel `Spectrum` path is active in material evaluation and unit tests, and is the compatibility layer for a future packet-style spectral integrator.

CLI overrides:

```bash
go -C engine/go run ./cmd/ray --script ../../examples/scenes/feature-showcase.json --width 800 --height 800 --samples 2000 --exposure 2.1 --tone-mapping reinhard --gamma 2.2
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
examples/scenes/neutral-dispersion-slit-test.json
examples/scenes/feature-showcase.json
```

`feature-showcase.json` exercises Lambert color bleeding, constant emission, specular reflection, constant-IOR glass, Cauchy-dispersive glass, GGX rough conductor, spectral sampling, and tone-mapped PNG output.
