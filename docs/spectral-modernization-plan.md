# Spectral Modernization Plan

This document defines a staged plan for moving Ray from the current RGB plus partial hero-wavelength model toward a modern spectral rendering pipeline. The goal is not to rewrite the renderer in one pass. The goal is to introduce clear color and spectrum boundaries while keeping existing scenes renderable.

## 1. Current State

The renderer has already moved through an important material architecture migration:

- Scene materials are parsed into `internal/material/core.Material`.
- Surface scattering is routed through BSDF/BxDF interfaces.
- `Ray` carries `WaveLength` and `WavelengthPDF`.
- Specular dielectric supports Cauchy IOR and wavelength-aware eta evaluation.
- Film output supports exposure, `linear`, `reinhard`, `aces` tone mapping, and gamma encoding.

The spectral pipeline is still incomplete:

- `core.Spectrum` is a fixed RGB triplet.
- Lambert, rough conductor, and constant emission parameters are still parsed as RGB spectra.
- Camera wavelength sampling is immediately reconstructed with `WaveLengthToRGB`.
- Film accumulates only three RGB channels.
- Scene JSON does not distinguish `srgb`, `linear_srgb`, `acescg`, or sampled spectra.
- The current `aces` output option is a fitted tone mapper, not a complete ACES/OCIO view transform.

The current model should be described as:

```text
RGB material parameters
+ renderer-level hero wavelength propagation
+ RGB reconstruction weight
+ RGB film/output
```

This is useful for dispersion demos, but it is not yet a full spectral material and color-management system.

## 2. Goals

The target system should:

1. Keep accepting RGB material and light inputs.
2. Avoid treating internal `Spectrum` as synonymous with RGB.
3. Give BSDF, emitter, IOR, integrator, film, and output code explicit spectrum-mode contracts.
4. Support three rendering modes:
   - `rgb`: fast compatibility mode.
   - `hero_wavelength`: one wavelength per camera path.
   - `sampled`: multiple wavelengths per path.
5. Use explicit output color management:
   - spectral/RGB result to CIE XYZ or a linear working RGB space
   - working space to tone mapping or view transform
   - display encoding, usually sRGB
6. Provide a schema that is validatable, migratable, and extensible.

## 3. Target Architecture

The pipeline should be split into five layers:

```text
Scene JSON / Editor
  -> parameter decoding and color-space conversion
  -> spectrum representation
  -> BSDF / Emitter / IOR evaluation
  -> film accumulation
  -> output transform
```

### 3.1 Input Layer

The input layer converts authored data into renderer parameters. It should not own scattering logic.

Recommended schema:

```json
{
  "surface": {
    "type": "lambert",
    "albedo": {
      "type": "rgb",
      "space": "linear_srgb",
      "value": [0.8, 0.72, 0.6]
    }
  }
}
```

Supported spectral parameter types:

```text
rgb
constant
sampled
blackbody
metal_ior
```

Emission example:

```json
{
  "emission": {
    "type": "constant",
    "radiance": {
      "type": "blackbody",
      "temperature": 6500,
      "scale": 5
    }
  }
}
```

Metal preset example:

```json
{
  "surface": {
    "type": "rough_conductor",
    "ior": {
      "type": "metal_ior",
      "material": "Cu"
    },
    "roughness": 0.2
  }
}
```

Compatibility rules:

- Legacy `[r, g, b]` arrays remain accepted.
- Legacy arrays default to `linear_srgb`.
- The parser may emit deprecation warnings, but old scenes should still render.
- New documentation and editor output should prefer object-form spectral parameters.

### 3.2 Spectrum Layer

Current representation:

```go
type Spectrum struct {
    R float64
    G float64
    B float64
}
```

The target design should separate spectral values from authored spectral parameters.

Avoid using Go interfaces in the hottest numeric path at first. A fixed small value type is a safer initial step:

```go
type SpectrumMode int

const (
    SpectrumModeRGB SpectrumMode = iota
    SpectrumModeHeroWavelength
    SpectrumModeSampled
)

type Spectrum struct {
    C [SpectrumSampleCount]float64
}
```

Initial count:

```go
const SpectrumSampleCount = 4
```

For a lower-risk first implementation, keep the current RGB value type and add a parameter abstraction:

```go
type SpectralParameter interface {
    Eval(ctx ShadingContext) Spectrum
    Bounds() SpectrumBounds
}
```

Then BSDFs store parameters instead of already-collapsed RGB values:

```go
type Lambert struct {
    Albedo SpectralParameter
}
```

Each `Eval` or `Sample` call evaluates parameters under the current `ShadingContext`.

### 3.3 Wavelength Sampling Layer

Move wavelength sampling out of `Ray.SetSpectralWavelength` and into a renderer-level sampler.

Recommended API:

```go
type WavelengthSample struct {
    LambdaNM []float64
    PDF      []float64
    Weight   Spectrum
}

type WavelengthSampler interface {
    Sample(u float64) WavelengthSample
}
```

Mode behavior:

```text
rgb:
  no wavelength sampling
  RGB parameters evaluate directly

hero_wavelength:
  sample one lambda per camera path
  BSDFs evaluate spectral parameters at lambda
  film reconstructs XYZ/RGB with wavelength matching functions

sampled:
  sample N wavelengths per path
  BSDFs return N-channel spectra
  film integrates N channels to XYZ/RGB
```

Important rules:

- BxDFs do not sample wavelengths.
- The camera path integrator samples wavelengths.
- BxDFs only evaluate parameters for the wavelengths in `ShadingContext`.
- Film reconstructs path contribution to the output color space.

### 3.4 Material Layer

Material implementations should replace raw RGB constants with spectral parameters.

Recommended priority:

1. Lambert
   - `albedo: SpectralParameter`
   - RGB albedo may be evaluated directly in RGB mode or upsampled in spectral modes.

2. Constant emission
   - `radiance: SpectralParameter`
   - supports RGB, sampled, and blackbody inputs.

3. Specular dielectric
   - `reflectance/transmittance: SpectralParameter`
   - keep Cauchy IOR and evaluate it through `ctx.Wavelengths`.

4. Rough conductor
   - support sampled `eta/k` or `metal_ior` presets.
   - keep RGB eta/k as a compatibility form only.

5. Rough dielectric
   - implement after the spectral framework is stable.
   - evaluate Fresnel using wavelength-aware eta.

### 3.5 Film And Output Layer

Short term, RGB Film can remain, but its working space must be explicit:

```text
rgb:
  BSDF returns linear working RGB
  Film accumulates linear working RGB

hero_wavelength:
  path contribution is narrow-band
  Film converts lambda contribution to XYZ/RGB before accumulation

sampled:
  Film accumulates XYZ or working RGB after spectral integration
```

Longer-term Film should use either XYZ internally:

```go
type Film struct {
    XYZ     [3]Tensor[float64]
    Samples int64
}
```

or explicit working-space RGB:

```go
type Film struct {
    WorkingSpace ColorSpace
    Data         [3]Tensor[float64]
    Samples      int64
}
```

Output transform:

```text
linear_srgb / acescg / xyz
  -> tone mapping or view transform
  -> display_srgb
```

Minimum viable implementation:

- Keep existing `linear`, `reinhard`, and `aces` options.
- Add `working_space`, default `linear_srgb`.
- Add `display_space`, default `srgb`.
- Document that current `aces` is an ACES-fitted tone mapper, not full OCIO.

## 4. JSON Schema

### 4.1 Spectral Parameters

RGB:

```json
{
  "type": "rgb",
  "space": "linear_srgb",
  "value": [1, 0.8, 0.6]
}
```

Sampled spectrum:

```json
{
  "type": "sampled",
  "wavelengths_nm": [400, 500, 600, 700],
  "values": [0.1, 0.4, 0.8, 0.7],
  "interpolation": "linear"
}
```

Blackbody:

```json
{
  "type": "blackbody",
  "temperature": 6500,
  "scale": 1
}
```

Constant:

```json
{
  "type": "constant",
  "value": 0.8
}
```

### 4.2 Render Config

```json
{
  "render": {
    "spectrum_mode": "hero_wavelength",
    "wavelength_min_nm": 380,
    "wavelength_max_nm": 780,
    "wavelength_samples": 1,
    "working_space": "linear_srgb",
    "display_space": "srgb",
    "tone_mapping": "aces_fitted",
    "gamma": 2.2
  }
}
```

Allowed values:

```text
spectrum_mode:
  rgb
  hero_wavelength
  sampled

working_space:
  linear_srgb
  acescg
  xyz

display_space:
  srgb
```

### 4.3 Backward Compatibility

Legacy:

```json
"albedo": [0.8, 0.8, 0.8]
```

Equivalent new form:

```json
"albedo": {
  "type": "rgb",
  "space": "linear_srgb",
  "value": [0.8, 0.8, 0.8]
}
```

`emission.color` may remain as a legacy alias for one version cycle. New scenes should use `emission.radiance`.

## 5. Implementation Phases

### Phase 0: Documentation And Boundaries

Goals:

- Record the current RGB/hero-wavelength hybrid status.
- Define the target schema.
- Define compatibility and test baselines.

Tasks:

- Add this plan.
- Update `scene-json-current.md` with legacy array semantics.
- Establish a baseline smoke test for `feature-showcase.json`.

Acceptance:

- Existing tests pass.
- Existing scene JSON files still render without modification.

### Phase 1: Spectral Parameter Parsing

Goals:

- Introduce `SpectralParameter`.
- Support both legacy arrays and new object-form spectrum parameters.
- Keep the BSDF numeric path RGB-compatible.

Tasks:

- Add `internal/material/spectrum` or `internal/material/core/spectral_parameter.go`.
- Implement:
  - `RGBParameter`
  - `ConstantParameter`
  - `SampledParameter`
  - `BlackbodyParameter`
- Replace `requiredSpectrumField` and `optionalSpectrumField` with spectral-parameter variants.
- Wire Lambert and constant emission first.

Acceptance:

- Legacy `[r, g, b]` input renders the same in RGB mode.
- New object-form input matches legacy arrays in RGB mode.
- Invalid color spaces, negative reflectance, and non-monotonic wavelengths are rejected.

### Phase 2: Wavelength Sampler Extraction

Goals:

- Move wavelength sampling out of `Ray`.
- Make `ShadingContext` carry the active wavelength sample.

Tasks:

- Add `WavelengthSample` and `WavelengthSampler`.
- Make the camera path integrator select the spectrum mode from render config.
- Keep `Ray.WaveLength` for compatibility and debug during the transition.
- Set `ctx.SpectrumMode` from render config in `TraceRay`.

Acceptance:

- RGB mode does not sample wavelengths.
- Hero-wavelength mode gives each camera path one lambda and pdf.
- Cauchy glass still shows dispersion in hero-wavelength mode.

### Phase 3: CIE Reconstruction And Film

Goals:

- Stop using `WaveLengthToRGB` as the physical reconstruction core.
- Reconstruct wavelengths through CIE matching functions.

Tasks:

- Add CIE 1931 2-degree color matching tables.
- Implement `WavelengthToXYZ(lambda)`.
- Implement white-point normalization.
- Add `WorkingSpace` or internal XYZ storage to Film.
- Centralize output transforms in a color transform package.

Acceptance:

- Equal-energy white reconstructs near the configured white point.
- Single-wavelength tests form a reasonable spectral locus after RGB output.
- Old RGB mode output remains stable.

### Phase 4: Spectral Materials

Goals:

- Main BSDFs and emitters no longer store raw RGB constants.

Tasks:

- Lambert uses spectral reflectance.
- Constant emission supports RGB, sampled, and blackbody radiance.
- Specular dielectric reflectance and transmittance use spectral parameters.
- Rough conductor supports sampled eta/k.
- Add a minimal metal dataset: Al, Cu, Au, Ag.

Acceptance:

- Copper, gold, silver, and aluminum render with distinct spectral behavior.
- Blackbody lights at 3000K, 6500K, and 10000K show stable color-temperature differences.
- RGB fallback remains compatible.

### Phase 5: Editor Upgrade

Goals:

- The scene editor stops treating all material color as three unlabeled numbers.

Tasks:

- Add spectral parameter editing modes:
  - RGB
  - sampled table
  - blackbody
  - metal preset
- Add a color-space selector for RGB input.
- Add render controls for spectrum mode, working space, and display space.

Acceptance:

- The editor can create the new schema.
- Existing scenes can be loaded and saved in the new schema.
- Common materials remain easy to author.

## 6. Code Boundaries

Likely modules:

```text
engine/go/internal/material/core/spectrum.go
engine/go/internal/material/core/bxdf.go
engine/go/internal/material/spectrum/
engine/go/internal/material/bxdf/
engine/go/internal/material/emission/
engine/go/internal/material/ior/
engine/go/internal/controller/parse_materials.go
engine/go/internal/app/render_config.go
engine/go/internal/model/optics/ray.go
engine/go/internal/model/camera/film.go
engine/go/internal/ray_tracing/
apps/scene-editor/src/types/scene.ts
apps/scene-editor/src/components/InspectorPanel.tsx
```

Do not include in the first implementation pass:

- Full OSL-style node graphs.
- Volume rendering.
- Mandatory scene rewrites.
- Large OCIO dependency integration.
- Wavelength sampling inside BxDFs.

## 7. Test Strategy

Unit tests:

- Spectrum arithmetic.
- RGB, sampled, blackbody parameter parsing.
- Sampled spectrum interpolation.
- CIE wavelength-to-XYZ conversion.
- Cauchy IOR wavelength evaluation.
- Film output transform.

Integration tests:

- Legacy RGB scenes render without crashes.
- `feature-showcase.json` remains renderable.
- Dispersion scenes show non-gray separation in hero-wavelength mode.
- Blackbody lights change color with temperature.
- Metal presets do not produce NaN, Inf, or negative energy.

Numeric invariants:

- Passive reflectance is in `[0, 1]`.
- Radiance and emission may exceed 1 but cannot be negative.
- Wavelength pdf must be greater than 0.
- Finite checks remain near path throughput updates.
- BSDF `Eval`, `Sample`, and `PDF` agree on spectrum-mode behavior.

## 8. Risks

### 8.1 Higher Noise

Hero-wavelength rendering samples less color information per path than RGB.

Mitigation:

- Keep RGB mode as the default fast path.
- Add sampled spectrum mode.
- Add wavelength importance sampling later.

### 8.2 RGB To Spectrum Is Underdetermined

Many spectra can produce the same RGB value.

Mitigation:

- Treat RGB as an artistic input, not measured material data.
- Use a stable RGB upsampling strategy.
- Recommend sampled spectra or presets for measured materials.

### 8.3 ACES Naming

The current `aces` option is a fitted tone mapper, not a full ACES color pipeline.

Mitigation:

- Rename the new option to `aces_fitted`.
- Keep `aces` as a legacy alias.
- Add full OCIO or built-in ACES transforms later.

### 8.4 Performance

Multi-sample spectra increase inner-loop cost.

Mitigation:

- Keep RGB mode.
- Use fixed-size value types for hot-path spectra.
- Start sampled mode with small N.

## 9. Recommended Landing Order

Recommended sequence:

```text
Phase 1 spectral parameter parser
  -> Phase 2 wavelength sampler extraction
  -> Phase 3 CIE reconstruction and film color management
  -> Phase 4 spectral material data
  -> Phase 5 editor support
```

Smallest useful first PR:

1. Add spectral parameter types.
2. Make the parser accept object-form spectrum parameters.
3. Wire Lambert and constant emission.
4. Keep RGB output fully compatible.
5. Add parser and Lambert/emission tests.

After that first PR, the project has a real extension point for spectral rendering without breaking the current renderer.

