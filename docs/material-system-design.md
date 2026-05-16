# Material System Design

## Goals

The material system should move from a small set of optical parameters to a composable, physically grounded framework.

Current-style material data:

```text
color + reflectivity + refractivity + refractive_index
```

Target model:

```text
Material =
  Parameters
  + Surface BSDF
  + Subsurface BSSRDF
  + Volume
  + Emission
  + Textures / Fields
  + Sampling
  + Metadata / Validation
```

The central design principle is that a `Material` is a container of orthogonal light-interaction components. The lowest-level surface primitive is `BxDF`.

## Scope

The complete long-term material model includes:

```text
Material
  Surface
    BSDF / BxDF
    Normal / Tangent Frame
    Coating / Layering
    Displacement / Micro-displacement
  Subsurface
    BSSRDF
    Random-walk / Diffusion Profile
  Volume
    Phase Function
    Absorption / Scattering / Emission
    Heterogeneous Density Field
  Emission
    Area emission
    Spectral emission
    Temperature / blackbody
  Textures / Fields
    2D / 3D / Procedural
    UDIM / OSL-like nodes
    Sparse volume / neural field
  Metadata
    Units
    Color space
    Valid parameter ranges
    Energy conservation checks
    Differentiability support
```

The first implementation phase should focus on `Surface` and `BxDF`. Subsurface, volume, advanced texture graphs, and differentiability should be designed for but not implemented first.

## Core Concepts

### Material

`Material` owns high-level composition. It should not directly encode every scattering rule.

Conceptual shape:

```go
type Material struct {
    Surface    BSDF
    Subsurface BSSRDF
    Volume     VolumeMaterial
    Emission   Emitter
    Metadata   MaterialMetadata
}
```

Only `Surface` is required in the first phase.

### BxDF

`BxDF` is the normalized primitive for local surface scattering.

Every BxDF must provide:

```text
eval()
sample()
pdf()
albedo_bound()
roughness_info()
delta_flags()
```

Optional for research and inverse rendering:

```text
parameter_derivatives()
```

Candidate Go interface:

```go
type BxDF interface {
    Eval(ctx ShadingContext, wi, wo Direction) Spectrum
    Sample(ctx ShadingContext, wo Direction, u Sample2D) BxDFSample
    PDF(ctx ShadingContext, wi, wo Direction) float64

    AlbedoBound(ctx ShadingContext) Spectrum
    RoughnessInfo(ctx ShadingContext) RoughnessInfo
    DeltaFlags() DeltaFlags
}
```

Optional extension:

```go
type DifferentiableBxDF interface {
    ParameterDerivatives(ctx ShadingContext, wi, wo Direction) ParameterGradients
}
```

### BSDF

`BSDF` composes one or more BxDFs and owns mixture, layering, and coating behavior.

Examples:

```text
SingleBxDF
WeightedMixture
LayeredBSDF
CoatedBSDF
```

`BSDF` should expose the same core operations as `BxDF`, but may combine multiple component PDFs and sampling strategies.

## Interface Conventions

These conventions must be fixed before implementation.

### Direction Convention

Use local shading coordinates for BxDF calls:

```text
normal = +Z
wi = incident direction, pointing away from the surface toward the previous path vertex
wo = outgoing direction, pointing away from the surface toward the next path vertex / camera
```

Both directions are expressed in the local tangent frame unless an API explicitly states otherwise.

World-space conversion belongs in `ShadingContext` / `Frame`, not inside individual BxDFs.

### Hemisphere Convention

For reflection BxDFs:

```text
wi.z > 0
wo.z > 0
```

For transmission BxDFs:

```text
wi.z and wo.z are on opposite sides
```

Invalid direction pairs should return zero value and zero PDF unless the BxDF explicitly supports that configuration.

### Eval Convention

`Eval` returns the BSDF value:

```text
f(wi, wo)
```

It does not include the cosine term.

Path throughput updates should use:

```text
throughput *= f(wi, wo) * abs_cos_theta(wi) / pdf(wi)
```

### PDF Convention

`PDF` is measured with respect to solid angle.

It does not include projected solid angle unless the API explicitly introduces a projected-measure variant.

For delta distributions, `PDF` cannot be represented as a normal finite density. Delta behavior must be exposed through `DeltaFlags` and the sample result.

### Sample Convention

`Sample` returns a direction, PDF, evaluated value, and flags.

Conceptual shape:

```go
type BxDFSample struct {
    Wi           Direction
    F            Spectrum
    PDF          float64
    Flags        DeltaFlags
    Eta          float64
    WavelengthNM float64
}
```

`F` should be the same value returned by `Eval(ctx, wi, wo)` for non-delta samples. It should not pre-divide by PDF and should not include the cosine term.

### Spectral Convention

Each BxDF must explicitly declare its spectrum support:

```text
RGB only
Spectral only
RGB and spectral
```

Implicit conversion should be avoided in the BxDF itself. Conversion policy belongs in material loading, texture evaluation, or renderer configuration.

## Shading Context

BxDFs require more than directions.

Conceptual context:

```go
type ShadingContext struct {
    Position        Point
    GeometricNormal Direction
    ShadingNormal   Direction
    Frame           TangentFrame
    UV              Vec2
    Wavelengths     WavelengthSet
    TransportMode   TransportMode
    MediumIn        Medium
    MediumOut       Medium
    Textures        TextureEvaluator
}
```

Normal mapping, tangent frames, and displacement should be handled before BxDF evaluation. BxDFs should see a coherent local frame.

## BxDF Library

Initial and long-term primitives:

```text
Lambert / Oren-Nayar
GGX / Beckmann microfacet
Disney diffuse
Rough dielectric
Rough conductor
Thin dielectric
Sheen
Clearcoat
Retroreflection
Diffuse transmission
Hair BSDF
Cloth / fiber BSDF
Flakes / glints
Custom research BSDF
```

Recommended first phase:

```text
Lambert
Oren-Nayar
RoughDielectric GGX
RoughConductor GGX
ThinDielectric
Clearcoat
```

## Microfacet Defaults

Microfacet materials should default to:

```text
Normal Distribution: GGX first, Beckmann second
Masking-Shadowing: Smith
Fresnel: exact dielectric / conductor Fresnel
Sampling: visible normal distribution function, VNDF
```

This gives a stable industrial baseline and a research-friendly connection to standard references.

## Validation Requirements

Every production BxDF should pass the same validation harness.

### 1. Non-Negativity

For sampled valid direction pairs:

```text
f(wi, wo) >= 0
pdf(wi, wo) >= 0
```

### 2. Reciprocity

For reciprocal materials:

```text
f(wi, wo) ~= f(wo, wi)
```

Non-reciprocal materials must explicitly declare that property in metadata or flags.

### 3. Energy Conservation

For fixed `wo`, integrate:

```text
integral f(wi, wo) * abs_cos_theta(wi) dwi <= 1
```

Use Monte Carlo integration with tolerances per BxDF family.

### 4. PDF Consistency

`Sample` and `PDF` must describe the same distribution.

Validation options:

```text
histogram test
chi-square test
sample mean test against known integrals
```

### 5. Spectral Consistency

Each BxDF must make RGB and spectral behavior explicit.

Tests should cover:

```text
RGB path
Spectral path
Unsupported mode errors
Conversion boundaries
```

### 6. Numerical Stability

Stress cases:

```text
grazing angles
roughness near 0
roughness near 1
IOR near 1
high IOR
conductor eta/k extremes
wi or wo near normal
wi or wo below the expected hemisphere
```

Results should not produce NaN, Inf, negative PDFs, or unstable energy spikes.

## Renderer Integration

The renderer should move from material-owned ray propagation:

```text
hit surface -> material mutates / creates next ray
```

to sampling-based path tracing:

```text
hit surface
build ShadingContext
sample material surface
throughput *= f * abs_cos_theta / pdf
spawn next ray
```

This change is required for BSDF sampling, multiple importance sampling, physically meaningful PDFs, and validation.

Legacy material behavior should be preserved through an adapter during migration.

## Suggested Package Layout

```text
engine/go/internal/material/
  core/
    spectrum.go
    direction.go
    frame.go
    sampling.go
    flags.go
  bxdf/
    lambert.go
    oren_nayar.go
    microfacet.go
    rough_dielectric.go
    rough_conductor.go
  bsdf/
    single.go
    mixture.go
    layered.go
  emission/
    area.go
    blackbody.go
  volume/
    phase.go
    medium.go
  texture/
    constant.go
    image.go
    procedural.go
  validation/
    non_negative.go
    reciprocity.go
    energy.go
    pdf_consistency.go
```

The existing `model/optics/material.go` can remain during migration and later become a compatibility layer.

## Migration Plan

### Phase 1: Interfaces And Lambert

Delivered:

```text
Spectrum
Direction / Frame helpers
BxDF interface
BxDFSample
Lambert BxDF
basic validation harness
```

The initial implementation lives under `engine/go/internal/material` and is intentionally isolated from the current renderer.

### Phase 2: BSDF Containers

Delivered:

```text
core.Scattering
core.BSDF
core.Material container
bsdf.Single
bsdf.WeightedMixture
validation over Scattering instead of only BxDF
```

This phase does not include a compatibility adapter for the old `model/optics.Material`. The new material system is allowed to move independently of the existing material representation.

### Phase 3: Renderer Sampling Path

Delivered:

```text
hit surface
build ShadingContext
sample Material.Surface
throughput *= f * abs_cos_theta / pdf
spawn next ray
```

The tracing loop now uses `Material.Surface.Sample/Eval/PDF` for non-emissive surface interaction. Emissive materials terminate the path and multiply the current throughput by emitted radiance.

### Phase 4: Material Schema

Delivered initial schema:

```json
{
  "id": "matte-red",
  "surface": {
    "type": "lambert",
    "albedo": [0.8, 0.1, 0.1]
  }
}
```

Emissive material:

```json
{
  "id": "light",
  "emission": {
    "type": "constant",
    "color": [7, 7, 7]
  }
}
```

Ideal mirror:

```json
{
  "id": "mirror",
  "surface": {
    "type": "specular_reflection",
    "reflectance": [1, 1, 1]
  }
}
```

Ideal dielectric:

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

Dispersive dielectric:

```json
{
  "id": "crown-glass",
  "surface": {
    "type": "specular_dielectric",
    "reflectance": [1, 1, 1],
    "transmittance": [1, 1, 1],
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

`cauchy` coefficients use wavelengths in micrometers:

```text
eta(lambda_nm) = A + B / lambda_um^2 + C / lambda_um^4
```

`eta_inside` remains accepted as a shorthand for constant IOR while scenes move to the explicit `ior` block.

Rough conductor:

```json
{
  "id": "brushed-metal",
  "surface": {
    "type": "rough_conductor",
    "eta": [0.2, 0.9, 1.5],
    "k": [3.9, 2.5, 1.9],
    "roughness": 0.25
  }
}
```

Example render command:

```bash
go -C engine/go run ./cmd/ray --script ../../examples/scenes/rough-conductor.json --samples 64 --width 256 --height 256 --output-image ../../outputs/rough-conductor.png
```

Old fields such as `color`, `reflectivity`, `refractivity`, `radiate`, and `diffuse_loss` are not part of the new material schema. They are intentionally not translated in the new parser.

### Phase 5: Microfacet

Implement:

```text
GGX distribution
Smith masking-shadowing
dielectric Fresnel
conductor Fresnel
VNDF sampling
RoughDielectric
RoughConductor
```

Add stress tests for roughness, IOR, and grazing angles.

### Phase 6: Textures, Emission, Volume

Add these after the surface model is stable:

```text
texture inputs for BxDF parameters
area and spectral emission
phase functions
homogeneous volume
heterogeneous volume
```

### Phase 6.0: IOR And Dispersion Wiring

Delivered:

```text
material/ior model interface
constant IOR model
Cauchy dispersion model
specular dielectric wavelength-aware eta evaluation
renderer-level wavelength sampling on camera rays
lambda PDF propagation through ShadingContext
white-point normalized spectral-to-RGB reconstruction
ray wavelength propagation through ShadingContext and BxDFSample
```

The camera samples one wavelength per path at ray generation time. The ray records both `WaveLength` and `WavelengthPDF`, and the initial throughput is multiplied by a white-point normalized reconstruction weight:

```text
throughput_rgb *= wavelength_to_rgb(lambda) / mean_visible_wavelength_to_rgb
```

BxDFs do not sample wavelengths. Dispersive BxDFs only evaluate their IOR model at the wavelength already carried by the renderer context. If no wavelength is present, they fall back to a deterministic 550nm value for non-renderer tests and compatibility paths.

## Open Design Questions

1. Should `Spectrum` start as RGB, sampled wavelengths, or a generic interface?
2. Should BxDFs own parameter validation, or should constructors enforce all valid ranges?
3. Should delta materials return `PDF = 0` with delta flags, or use a separate sample kind?
4. How should old `reflectivity` / `refractivity` blend into the new BSDF model?
5. Should renderer transport mode distinguish radiance and importance from the first implementation?
6. How strict should validation tolerances be for Monte Carlo energy checks?

## Non-Goals For The First Implementation

The first implementation should not attempt:

```text
BSSRDF random walk
heterogeneous participating media
UDIM or OSL-like node graphs
neural fields
differentiable rendering
hair and cloth BSDFs
full MIS rewrite
```

These should be supported by the architecture but implemented after the base BSDF path is correct.
