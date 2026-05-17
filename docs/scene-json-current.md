# Current Scene JSON

This document records the current scene JSON fields used by the Go renderer after the BSDF/BxDF material migration.

## Top-Level Shape

```json
{
  "materials": [],
  "objects": [],
  "cameras": [],
  "render": {}
}
```

`camera` is still accepted as a legacy alias for `cameras` when `cameras` is absent.

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
    "gamma": 2.2
  }
}
```

Supported tone mappers:

```text
linear
reinhard
aces
```

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
