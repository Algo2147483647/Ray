# Optics and Materials

This document describes the current optics and material model after the BSDF/BxDF and controller refactor.

## 1. Ray State

A ray is stored in `engine/model/optics/ray.go`. It carries:

- origin,
- direction,
- RGB/XYZ color weight,
- sampled wavelength in nanometers,
- wavelength PDF,
- current scalar refractive index,
- medium stack.

The wavelength fields allow the renderer to operate in RGB, hero-wavelength, or sampled wavelength modes without changing camera code.

## 2. Spectrum Values

`engine/model/optics/spectrum.go` defines `optics.Spectrum`.

A spectrum can be:

- RGB, using scene-linear RGB compatibility channels,
- sampled, using channels aligned with the active wavelength samples.

Spectrum-valued material parameters implement `optics.SpectralParameter` in `engine/model/optics/spectrum_parameter/`:

```text
constant_parameter.go
rgb_parameter.go
sampled_parameter.go
blackbody_parameter.go
```

These parameters evaluate against a wavelength context, so the same material can produce RGB values in RGB mode and sampled values in spectral modes.

## 3. Material Package Layout

The material system lives under `engine/model/material/`:

```text
material.go       Material container and Emitter interface
validation.go     Physical validity checks for scattering functions
bsdf/             BSDF wrappers and mixtures
bxdf/             Individual scattering lobes
emission/         Emission models
medium/           IOR models, medium registry, boundary stack
microfacet/       Fresnel and GGX helpers
```

The intended boundary is:

- `bxdf`: individual scattering models such as Lambert, specular dielectric, and rough conductor.
- `bsdf`: composition and dispatch over one or more BxDFs.
- `medium`: medium identity, IOR evaluation, and nested-boundary stack logic.
- `microfacet`: low-level microfacet math shared by BxDFs.

## 4. Shading Context

`bxdf.ShadingContext` carries local information needed to evaluate or sample scattering:

- transport mode,
- spectrum mode,
- current IOR,
- active wavelength or wavelength packet,
- wavelength PDF,
- incident/transmitted medium IDs,
- incident/transmitted eta values,
- entering/leaving boundary flag.

This context is built in `engine/ray_tracing/trace_ray.go` from the ray, object boundary, and medium registry.

## 5. Surface Scattering

The current surface models are:

```text
lambert
specular_reflection
specular_dielectric
rough_conductor
```

Relevant code:

```text
engine/model/material/bxdf/lambert.go
engine/model/material/bxdf/specular.go
engine/model/material/bxdf/rough_conductor.go
engine/model/material/bsdf/single.go
engine/model/material/bsdf/mixture.go
```

Each BxDF supports:

- `Eval`,
- `Sample`,
- `PDF`,
- `AlbedoBound`,
- `RoughnessInfo`,
- `DeltaFlags`.

Delta materials such as perfect specular reflection and transmission sample deterministic directions. Rough conductor uses GGX microfacet distribution and conductor Fresnel.

## 6. Fresnel and Refraction

Dielectric Fresnel and refraction are implemented in `engine/model/material/bxdf/specular.go`.

For dielectric transmission:

1. resolve the active wavelength,
2. evaluate the inside IOR model,
3. resolve incident/transmitted eta from the medium context when available,
4. sample reflection or refraction by Fresnel probability,
5. update the transmitted medium on the returned sample.

Conductor Fresnel lives in `engine/model/material/microfacet/fresnel.go` and supports RGB or sampled spectra.

## 7. Media and Boundaries

Medium state lives in `engine/model/material/medium/`:

```text
ior.go       Constant and Cauchy IOR models
medium.go    Registry and homogeneous media
boundary.go  Object boundary description
stack.go     Nested medium stack and priority resolution
```

Every object may declare a `medium_boundary` in scene JSON. During tracing, the renderer compares the current ray medium stack with the object's boundary to compute:

- incident medium,
- transmitted medium,
- whether the ray is entering,
- eta on both sides of the interface.

This is more robust than the old single scalar IOR model and supports nested dielectric interfaces.

## 8. Emission

Constant emission lives in `engine/model/material/emission/constant.go`.

Emission uses spectral parameters just like surface reflectance and transmittance. This means a light can be RGB, constant scalar, sampled spectrum, or blackbody.

## 9. Wavelength Sampling

Wavelength sampling is owned by the renderer rather than the camera:

```text
engine/ray_tracing/wavelength_sampler.go
engine/ray_tracing/trace_pixel.go
```

Supported modes:

```text
rgb
hero_wavelength
sampled
```

In spectral modes, the ray carries a sampled wavelength and PDF. After tracing, spectral power is converted through CIE 1931 XYZ matching functions in `engine/model/optics/wavelength.go`.

## 10. Current Limits

Implemented:

- Lambert diffuse,
- perfect specular reflection,
- perfect specular dielectric transmission,
- Cauchy dispersion,
- GGX rough conductor,
- constant spectral emission,
- nested medium stack for boundary IOR,
- RGB, hero-wavelength, and sampled wavelength render modes.

Not yet implemented:

- rough dielectric,
- full volume absorption/scattering transport,
- direct-light sampling and MIS,
- caustic-focused integrators such as photon mapping or BDPT,
- production spectral material databases.
