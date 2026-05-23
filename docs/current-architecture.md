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

`engine/model/camera/Film` stores three working/display channels plus a `camera.FilmColorSpace`.

Current film spaces:

```text
FilmColorSpaceLinearSRGB
FilmColorSpaceACEScg
FilmColorSpaceXYZ
```

Spectral render modes also allocate `Film.SpectralBins`. During rendering, each wavelength path returns a scalar radiance value tagged with its wavelength. The film accumulates those values into spectral bins and converts the bins once into the selected film working space after rendering. This keeps wavelength energy available through accumulation instead of converting every traced path to three channels immediately.

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
trace_scene.go        film preparation, workers, final spectral conversion
trace_tiles.go        tile iteration and pixel dispatch
trace_pixel.go        camera samples and wavelength samples
trace_ray.go          recursive surface path tracing
medium_transport.go   medium context, absorption, and boundary stack updates
throughput.go         RGB/spectral throughput handling
```

The current render flow is:

```text
TraceScene
-> prepareFilm
-> worker goroutines over tiles
-> TraceTile
-> TracePixel
-> TraceRGB or TraceSpectral
-> TraceRay
```

`TraceScene` initializes spectral bins for non-RGB modes, then launches tile workers. `TraceTile` only maps tile coordinates to pixels and delegates each pixel to `TracePixel`.

`TracePixel` owns the spectrum-mode branch:

- `rgb`: trace `samples` RGB camera paths with `TraceRGB`, average the returned `Color3`, and write the three film channels directly.
- `hero_wavelength`: trace one wavelength path per camera sample with `TraceSpectral`, record each scalar spectral sample into `Film.SpectralBins`.
- `sampled`: for each camera sample, trace `wavelength_samples` stratified wavelength subpaths with `TraceSpectral`, record each scalar spectral sample into `Film.SpectralBins`.

`TraceSpectral` returns wavelength-tagged scalar samples, not a final `Color3`. Each subpath sets the sampled wavelength and PDF on the ray, calls `TraceRay`, converts the traced spectral path to a scalar radiance estimate with `SpectralRayToScalar`, applies the wavelength PDF through `SpectralSampleRadiance`, and normalizes the batch before film recording.

After all workers finish, `TraceScene` calls `Film.ConvertSpectralBinsToFilmColorSpace` for spectral modes. That final step integrates the stored spectral bins through the CIE 1931 matching functions and converts XYZ into the film working color space.

`trace_ray.go` builds `bxdf.ShadingContext`, samples surfaces, applies throughput, and delegates medium operations to `medium_transport.go`.

## Math Utilities

`engine/utils/maths/frame.go` provides `maths.Frame` for world/local direction transforms around a surface normal. This is shared geometry math, not renderer-specific state.
