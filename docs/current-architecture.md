# Current Architecture

This document is the short current-state map for code boundaries. Planning documents may describe target designs; this file should reflect what the repository does now.

## Optics

`engine/model/optics/` owns ray optical state, spectrum values, wavelength sampling, color conversion, and renderer spectrum modes.

Key types:

```text
Ray
Spectrum
SpectralParameter
RGBColorSpace
SpectrumMode
```

`optics.SpectrumMode` is a renderer sampling policy:

```text
SpectrumModeRGB
SpectrumModeHeroWavelength
SpectrumModeSampledWavelengths
```

## Film And Color Spaces

`engine/model/camera/Film` stores three accumulation channels plus a `camera.FilmColorSpace`.

Current film spaces:

```text
FilmColorSpaceLinearSRGB
FilmColorSpaceXYZ
```

Material RGB input spaces are intentionally separate and live in `optics.RGBColorSpace`:

```text
RGBColorSpaceLinearSRGB
RGBColorSpaceSRGB
RGBColorSpaceACEScg
```

This keeps authored RGB interpretation separate from film accumulation/storage.

## Materials

`engine/model/material/` owns high-level material composition.

```text
material.go   Material container and metadata
bsdf/         BSDF composition
bxdf/         Individual scattering lobes
emission/     Emitters
medium/       Medium definitions and boundary state
microfacet/   Fresnel and GGX helpers
```

`bxdf.ShadingContext` carries local path state into BxDF evaluation. It references `optics.SpectrumMode`; BxDFs do not own renderer wavelength sampling policy.

## Media

`engine/model/material/medium/` owns medium identity and boundary state:

```text
Medium
Registry
Boundary
Stack
Model
Coefficient
```

Implemented behavior:

- constant and Cauchy IOR,
- named homogeneous media,
- object-level `medium_boundary`,
- priority-aware nested medium stack,
- Beer-Lambert absorption from `sigma_a`.

Parsed for future participating-media scattering:

- `sigma_s`.

## Ray Tracing

`engine/ray_tracing/` owns integration:

```text
trace_pixel.go        camera samples and wavelength samples
trace_ray.go          recursive surface path tracing
medium_transport.go   medium context, absorption, and boundary stack updates
throughput.go         RGB/spectral throughput handling
```

`trace_pixel.go` samples wavelengths according to `optics.SpectrumMode`. `trace_ray.go` builds `bxdf.ShadingContext`, samples surfaces, applies throughput, and delegates medium operations to `medium_transport.go`.

## Math Utilities

`engine/utils/maths/frame.go` provides `maths.Frame` for world/local direction transforms around a surface normal. This is shared geometry math, not renderer-specific state.
