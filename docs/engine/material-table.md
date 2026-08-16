# Engine Material Categories, Mathematics, and Input Schemas

> Scope: this document describes only the current implementation under `engine/`. Studio schemas, example-only conventions, and older documentation are excluded.
>

## 1. What a Material Really Is

The Engine does not have a top-level material `type` hierarchy. Every JSON entry becomes the same runtime type, `*material.Material`, which is a composition of:

- an optional `bsdf.BSDF` in `Surface`;
- an optional `emission.Emitter` in `Emission`;
- internal `MaterialMetadata` initialized by the factory.

A material must contain at least a surface or an emission block. It may contain both. The real discriminator hierarchy therefore exists below the material level:

```text
material.Material
|- Surface: bsdf.BSDF
|  |- physical BxDF wrapped by bsdf.Single
|  |- bsdf.WeightedMixture
|  `- bsdf.CylindricalGridCutout
`- Emission: emission.Emitter
```

Supporting tagged inputs include spectral parameters and IOR models. Media are a related scene-level subsystem, not another material subtype.

## 2. Real Category Tables

### 2.1 Top-Level Material Forms

| JSON form | Runtime result | Accepted | Current path-tracing behavior |
| --- | --- | --- | --- |
| `surface` only | `Material{Surface: ...}` | Yes | Samples the BSDF at a hit |
| `emission` only | `Material{Emission: ...}` | Yes | Evaluates emission and terminates the camera path |
| both | `Material{Surface: ..., Emission: ...}` | Yes | The regular `TraceRay` path evaluates emission first and returns, so its surface is not sampled at that hit |
| neither | Empty `Material` | No | Factory error: material requires surface or emission |

Every material requires a unique, non-empty `id`. Objects refer to it through `material_id`.

### 2.2 Surface Discriminators

| JSON `surface.type` | Actual runtime result | Category | Delta/event flags |
| --- | --- | --- | --- |
| `weighted_mixture` | `bsdf.WeightedMixture` | Normalized probabilistic mixture | Union of component flags |
| `lambert` | `bsdf.Single{BxDF: bxdf.Lambert}` | Diffuse reflection | None |
| `specular_reflection` | `bsdf.Single{BxDF: bxdf.SpecularReflection}` | Perfect mirror reflection | `DeltaReflection` |
| `specular_dielectric` | `bsdf.Single{BxDF: bxdf.SpecularDielectric}` | Perfect Fresnel reflection and refraction | `DeltaReflection`, `DeltaTransmission`, `TransmissionEvent` |
| `rough_conductor` | `bsdf.Single{BxDF: bxdf.RoughConductor}` | GGX conductor reflection | None |
| `rough_dielectric_reflection` | `bsdf.Single{BxDF: bxdf.RoughDielectricReflection}` | GGX dielectric reflection lobe only | None |
| `rough_dielectric_transmission` | `bsdf.Single{BxDF: bxdf.RoughDielectricTransmission}` | GGX dielectric transmission lobe only | `TransmissionEvent`, `NonReciprocal` |
| `cylindrical_grid_cutout` | `bsdf.CylindricalGridCutout` | Procedural line BSDF plus transparent gaps | Always includes `DeltaTransmission`, plus line-surface flags |
| `wire_mesh` | `bsdf.CylindricalGridCutout` | Alias of `cylindrical_grid_cutout` | Same as above |

There are nine JSON surface values but only eight distinct runtime surface constructions because `wire_mesh` is an alias.

### 2.3 Emission Discriminators

| JSON `emission.type` | Runtime type | Spatial rule | `IsDelta()` |
| --- | --- | --- | --- |
| `constant` | `emission.Constant` | Direction-independent spectral radiance | False |
| `cell_palette` | `emission.CellPalette` | Color selected from the dominant normal axis and sign | False |
| `uv_klein` | `emission.UVKlein` | HSL visualization of Klein-bottle UV coordinates | False |

### 2.4 Supporting Tagged Types

| Input layer | JSON values | Runtime implementations |
| --- | --- | --- |
| Spectral parameter | `rgb`, `constant`, `sampled`, `blackbody` | `RGBParameter`, `ConstantParameter`, `SampledParameter`, `BlackbodyParameter` |
| IOR model | `constant`, `cauchy` | `medium.Constant`, `medium.Cauchy` |
| Scene medium | `homogeneous` only | `medium.Homogeneous` |

## 3. Common Material Schema and Runtime Contract

```jsonc
{
  "materials": [{
    "id": "required unique non-empty string",
    "surface": { "type": "surface discriminator" },   // optional
    "emission": { "type": "emission discriminator" } // optional
  }]
}
```

- At least one of `surface` and `emission` is required.
- Unknown fields are generally ignored because the factories read selected values from untyped maps.
- The factory sets `Metadata.Name=id` and `Metadata.SpectrumMode=RGB`. Other metadata fields are not populated from JSON.
- Runtime spectral mode comes from the render handler and `ShadingContext`, not from the material metadata value.

The shared scattering contract is:

```text
Eval(ctx, wi, wo) -> f
Sample(ctx, wo, random2D) -> {wi, f, pdf, flags, eta, wavelength, medium}
PDF(ctx, wi, wo) -> solid-angle density for non-delta events
AlbedoBound(ctx) -> conservative throughput estimate
RoughnessInfo(ctx) -> delta/alpha metadata
DeltaFlags() -> model-level event capabilities
```

Directions are expressed in the local shading frame and the final component is the normal axis. After sampling, the regular path tracer multiplies throughput by:

$$
f(\omega_i,\omega_o)
\frac{|\cos\theta_i|}{p_\omega(\omega_i)}.
$$

Delta models return zero from `Eval` and `PDF`; their non-zero probability and value exist only in `Sample`.

## 4. Shared Spectral Parameter Schema

Every documented spectral field, including albedo, reflectance, transmittance, conductor eta/k, weight, radiance, and medium coefficients, uses the same parser.

Legacy RGB form:

```jsonc
[0.8, 0.6, 0.2] // exactly three non-negative linear-sRGB values
```

Tagged forms:

```jsonc
{ "type": "rgb", "value": [0.8, 0.6, 0.2],
  "space": "linear_srgb | srgb | acescg" } // default linear_srgb

{ "type": "constant", "value": "non-negative number" }

{ "type": "sampled",
  "wavelengths_nm": [/* at least two strictly increasing numbers */],
  "values": [/* same length, non-negative */],
  "interpolation": "linear" } // optional; linear is the only accepted value

{ "type": "blackbody",
  "temperature": "positive kelvin",
  "scale": "non-negative number" } // default 1
```

Processing details:

- `srgb` values are decoded channel-by-channel to linear values. `acescg` values are currently stored as authored linear values; no ACEScg-to-render-space transform is implemented.
- In wavelength modes, RGB parameters are uplifted through the Engine's RGB reflectance approximation.
- A sampled parameter uses linear interpolation and clamps outside its wavelength range to the nearest endpoint value.
- Without an active wavelength context, sampled data is converted to linear sRGB.
- Spectral blackbody evaluation uses Planck power relative to its value at 560 nm. RGB mode uses a color-temperature approximation instead of integrating the spectrum.
- The parser enforces non-negativity but does not cap reflectance, transmittance, albedo, or weights at 1. Values above 1 can violate energy conservation.
- Any spectral form is syntactically accepted for any spectral field, even combinations that are not physically meaningful, such as blackbody conductor eta.

## 5. Shared IOR Schema

Used by `specular_dielectric`, `rough_dielectric_reflection`, and `rough_dielectric_transmission`:

```jsonc
{
  "eta_outside": "positive number", // optional, default 1
  "ior": {
    "type": "constant",
    "eta": "positive number"
  }
}
```

or:

```jsonc
{
  "eta_outside": 1,
  "ior": {
    "type": "cauchy",
    "a": "number",
    "b": "number",
    "c": "number" // optional, default 0
  }
}
```

Legacy fallback when `ior` is absent:

```jsonc
{ "eta_inside": "positive number" } // optional, default 1.5
```

The Cauchy model evaluates, with wavelength $\lambda$ in micrometers:

$$
\eta(\lambda)=A+\frac{B}{\lambda^2}+\frac{C}{\lambda^4}.
$$

The parser checks positive finite eta at 380, 550, and 750 nm. If `ior` is present, it takes precedence over `eta_inside`.

For transmissive models, an active `medium_boundary` supplies incident and transmitted IOR values through `ShadingContext`; these override the surface's fallback eta pair. The rough reflection-only model is an exception: it always evaluates Fresnel with `eta_outside` and its inside IOR, rather than the boundary-resolved pair.

## 6. Surface Models

### 6.1 Weighted Mixture

Schema:

```jsonc
{
  "type": "weighted_mixture",
  "components": [{
    "weight": "finite number > 0",
    "surface": { "type": "any surface type" }
  }]
}
```

The component list must be non-empty. Parsing is recursive, so mixtures and procedural cutouts may themselves appear as components.

Let $w_i$ be a component weight and define $p_i=w_i/\sum_jw_j$. This model is a normalized blend, not an additive layer stack:

$$
f=\sum_i p_i f_i,
\qquad
p_\omega=\sum_i p_i p_{\omega,i}.
$$

Sampling selects component $i$ with probability $p_i$ and remaps the random number into that component's interval. For a non-delta sample, the final value and PDF are recomputed from the complete mixture. For a delta sample, only the selected component contributes and both its value and discrete PDF are multiplied by $p_i$.

Flags are bitwise-unioned. Roughness reports the largest component alpha and is delta only when every active component is delta. This is a statistical mixture; it does not implement coating order, multiple internal reflections, or Fresnel-aware lobe-selection probabilities.

### 6.2 Lambert

```jsonc
{
  "type": "lambert",
  "albedo": "spectral parameter; required"
}
```

For local direction dimension $D$, define:

$$
I_D
=
\frac{\pi^{(D-1)/2}}
{\Gamma\!\left((D-1)/2+1\right)}.
$$

Then, for upper-hemisphere directions $\omega_i$ and $\omega_o$:

$$
f(\omega_i,\omega_o)=\frac{\rho}{I_D},
\qquad
p_\omega(\omega_i)=\frac{\cos\theta_i}{I_D},
$$

where $\rho$ is the albedo.

In 3D, $I_3=\pi$, giving the standard $\rho/\pi$. Sampling is cosine-weighted and has a dedicated N-dimensional implementation, so Lambert is one of the surfaces intentionally supporting local dimensions above three.

### 6.3 Specular Reflection

```jsonc
{
  "type": "specular_reflection",
  "reflectance": "spectral parameter" // optional, default constant 1
}
```

This is a colored perfect mirror without a Fresnel term. It deterministically negates every tangential component of $\omega_o$ while preserving the final normal component. The discrete sample is:

$$
\omega_i=\operatorname{reflect}(\omega_o),
\qquad
f_{\mathrm{sample}}=\frac{R}{|\cos\theta_i|},
\qquad
p_{\mathrm{sample}}=1,
$$

where $R$ is the spectral reflectance.

The path-throughput cosine factor cancels the division. `Eval` and solid-angle `PDF` return zero because the distribution is a delta. The reflection logic supports arbitrary local direction dimension.

### 6.4 Specular Dielectric

```jsonc
{
  "type": "specular_dielectric",
  "reflectance": "spectral parameter",   // optional, default 1
  "transmittance": "spectral parameter", // optional, default 1
  "eta_outside": 1,                       // optional
  "ior": { /* constant or cauchy */ }      // optional; eta_inside fallback supported
}
```

For incident index $\eta_i$, transmitted index $\eta_t$, and incident cosine $c$, unpolarized dielectric Fresnel is:

$$
F=\frac{r_{\parallel}^2+r_{\perp}^2}{2}.
$$

Snell refraction satisfies

$$
\sin\theta_t=\frac{\eta_i}{\eta_t}\sin\theta_i.
$$

Total internal reflection gives $F=1$.

Sampling chooses reflection with probability $F$ and transmission with probability $1-F$:

- reflection: $f_{\mathrm{sample}}=RF/|\cos\theta_i|$ and $p=F$;
- transmission: $f_{\mathrm{sample}}=T(1-F)/|\cos\theta_i|$ and $p=1-F$.

If refraction fails, sampling falls back to reflection with probability 1. A transmission sample carries `TransmissionEvent`, the transmitted-side eta, and `ctx.TransmitMedium`. With a dispersive Cauchy IOR and an active wavelength, the sampled wavelength is propagated in the event.

The implementation does not apply an additional eta-squared radiance factor in this delta model.

### 6.5 Rough Conductor

```jsonc
{
  "type": "rough_conductor",
  "eta": "spectral parameter; required",
  "k": "spectral parameter; required",
  "roughness": "number in [0,1]", // optional, default 0.25
  "weight": "spectral parameter"  // optional, default 1
}
```

The factory maps perceptual roughness $r$ to $\alpha=r^2$; GGX then clamps $\alpha$ to $[10^{-4},1]$. Roughness zero is therefore still a very sharp non-delta GGX lobe, not a perfect mirror.

For the half-vector

$$
\omega_h=\frac{\omega_i+\omega_o}{\|\omega_i+\omega_o\|},
$$

the evaluated lobe is:

$$
f(\omega_i,\omega_o)
=
W\,
F_{\mathrm{conductor}}(\omega_i\cdot\omega_h,\eta,k)
\frac{D_{\mathrm{GGX}}(\omega_h)G(\omega_i,\omega_o)}
{4\cos\theta_i\cos\theta_o}.
$$

The conductor Fresnel calculation is performed per RGB channel or sampled wavelength from the complex IOR $\eta+ik$. GGX uses:

$$
D(\omega_h)
=
\frac{\alpha^2}
{\pi\left[\cos^2\theta_h(\alpha^2-1)+1\right]^2},
\qquad
G(\omega_i,\omega_o)
=
\frac{1}{1+\Lambda(\omega_i)+\Lambda(\omega_o)}.
$$

Sampling uses GGX visible-normal sampling, reflects $\omega_o$ about the sampled microfacet normal, and transforms visible-normal density with $1/(4|\omega_o\cdot\omega_h|)$.

### 6.6 Rough Dielectric Reflection

```jsonc
{
  "type": "rough_dielectric_reflection",
  "reflectance": "spectral parameter", // optional, default 1
  "eta_outside": 1,
  "ior": { /* constant or cauchy */ },
  "roughness": "number in [0,1]" // optional, default 0.25
}
```

This is the reflection lobe only. It uses the same GGX distribution and visible-normal sampling as the rough conductor, but with scalar dielectric Fresnel:

$$
f(\omega_i,\omega_o)
=
R\,F_{\mathrm{dielectric}}
\frac{D(\omega_h)G(\omega_i,\omega_o)}
{4\cos\theta_i\cos\theta_o}.
$$

It never produces transmission and is useful as a clearcoat/glaze reflection component in a mixture. Its effective roughness parameter is $\alpha=\max(r^2,10^{-4})$.

Current limitation: Fresnel always uses the configured outside and inside IOR values. It does not use `ctx.EtaIncident` and `ctx.EtaTransmit`, so exiting-interface or nested-medium reflection can use the wrong eta orientation.

### 6.7 Rough Dielectric Transmission

```jsonc
{
  "type": "rough_dielectric_transmission",
  "transmittance": "spectral parameter", // optional, default 1
  "eta_outside": 1,
  "ior": { /* constant or cauchy */ },
  "roughness": "number in [0,1]" // optional, default 0.25
}
```

This is the transmission lobe only. Directions $\omega_i$ and $\omega_o$ must be in opposite hemispheres. With $\eta=\eta_t/\eta_i$, the Walter-style transmission half-vector is:

$$
\omega_h
=
\frac{\omega_o+\eta\omega_i}
{\|\omega_o+\eta\omega_i\|}.
$$

The evaluated lobe is proportional to the ratio

$$
f(\omega_i,\omega_o)
\propto
\frac{
T(1-F)DG\,\eta^2
|\omega_i\cdot\omega_h|
|\omega_o\cdot\omega_h|
}{
\cos\theta_i\cos\theta_o
\left(\omega_o\cdot\omega_h
+\eta\,\omega_i\cdot\omega_h\right)^2
}.
$$

Radiance transport additionally applies the squared $(\eta_i/\eta_t)^2$ adjoint factor. The PDF multiplies visible-normal density by the half-vector-to-direction Jacobian.

Sampling draws a GGX visible normal and refracts through it. Refraction failure or total internal reflection returns an invalid sample; this model does not fall back to reflection. Use `specular_dielectric` for a combined perfect interface, or combine rough reflection and transmission explicitly while accounting for the fact that mixture selection is not Fresnel-aware.

The event is marked `TransmissionEvent|NonReciprocal`. Consequently, any scene containing this surface, directly or inside a mixture, is rejected by the current BDPT support check.

### 6.8 Cylindrical Grid Cutout / Wire Mesh

```jsonc
{
  "type": "cylindrical_grid_cutout | wire_mesh",
  "line_surface": { "type": "any surface type" }, // optional
  "origin": [0, 0, 0],                              // optional 3-vector
  "axis": [0, 0, 1],                                // optional non-zero 3-vector
  "reference_axis": [1, 0, 0],                      // optional 3-vector
  "line_width": 0.006,                              // optional, >= 0
  "gap_width": 0.03,                                // optional, >= 0
  "gap_height": 0.03,                               // optional, >= 0
  "reference_radius": 1                             // optional, > 0
}
```

The default line surface is a silver-like rough conductor with hard-coded eta, k, roughness, and weight values.

```jsonc
{
  "type": "rough_conductor",
  "eta": [0.15, 0.14, 0.13],
  "k": [4.1, 3.5, 2.7],
  "roughness": 0.22,
  "weight": [0.88, 0.90, 0.92]
}
```

For hit point $p$, the model projects $p-o$ onto the cylinder axis. Let $h$ be axial height, $\theta$ the angle around the orthonormal reference/bitangent frame, and $s=\theta r_{\mathrm{reference}}$. A point lies on a grid line when either:

$$
\operatorname{periodicDistance}
\!\left(s,w_{\mathrm{gap}}+w_{\mathrm{line}}\right)
\le \frac{w_{\mathrm{line}}}{2}
$$

or:

$$
\operatorname{periodicDistance}
\!\left(h,h_{\mathrm{gap}}+w_{\mathrm{line}}\right)
\le \frac{w_{\mathrm{line}}}{2}.
$$

On a line, Eval/Sample/PDF delegate to `line_surface`. In a gap, Sample returns deterministic straight-through transmission $\omega_i=-\omega_o$, value $1/|\cos\theta_i|$, and discrete PDF one. Gap transmission has `DeltaTransmission` but deliberately lacks `TransmissionEvent`, so it does not modify the medium stack.

The pattern uses the first three world-space hit coordinates. If fewer than three are available, every point is treated as a line. This is a BSDF mask, not geometric displacement: gaps remain intersection surfaces for visibility and shadow rays unless the transport path samples through them.

## 7. Emission Models

### 7.1 Constant Emission

```jsonc
{
  "type": "constant",
  "radiance": "spectral parameter; required"
  // color is accepted as a fallback alias when radiance is absent
}
```

If both `radiance` and `color` exist, `radiance` wins. `Emit` ignores outgoing direction and returns the evaluated spectral parameter, so emission is direction-independent and not one-sided at the emitter level.

### 7.2 Cell Palette Emission

```jsonc
{
  "type": "cell_palette",
  "palette": [[r,g,b], /* one or more non-negative RGB triples */], // optional
  "intensity": "non-negative number",                              // optional
  "shading": "solid | boundary_grid",                              // optional, default solid
  "grid_color": [r,g,b],                                            // optional, default white
  "grid_thickness": "non-negative world-space number"              // optional, default 0.02
}
```

The default palette contains eight colors for `-X,+X,-Y,+Y,-Z,+Z,-W,+W`. The emitter finds the geometric-normal component with largest absolute magnitude and chooses

$$
i=2a+\mathbf{1}_{\mathrm{positive}},
$$

wrapping the index modulo the palette length. Missing or zero normal data falls back to palette entry zero.

```text
-X [1.00, 0.20, 0.20]   +X [0.20, 1.00, 0.20]
-Y [0.20, 0.40, 1.00]   +Y [1.00, 0.85, 0.20]
-Z [1.00, 0.30, 0.90]   +Z [0.20, 0.95, 0.95]
-W [1.00, 0.55, 0.10]   +W [0.92, 0.92, 0.92]
```

In `boundary_grid` mode, non-dominant coordinates are compared with the object's AABB faces. A point within `grid_thickness` of any such boundary emits `grid_color`; otherwise it emits the cell color. Grid evaluation therefore depends on valid hit position, geometric normal, and Shape AABB data in `ShadingContext`.

Intensity scales palette colors but does not scale `grid_color`. `grid_color` and `grid_thickness` are parsed only in boundary-grid mode.

### 7.3 UV Klein Emission

```jsonc
{
  "type": "uv_klein",
  "saturation": "number in [0,1]", // optional, default 1
  "lightness": "number in [0,1]",  // optional, default 0.55
  "v_stripes": "positive integer",
  "intensity": "non-negative number" // optional, default 1
}
```

The emitter assumes UV values are angular coordinates in radians. It maps wrapped $u$ to HSL hue and alternates $v$ bands between full lightness and $0.45$ times the configured lightness, with $2\,\mathtt{v\_stripes}$ bands per $v$ cycle. The HSL color is converted to RGB and multiplied by intensity.

Current parser bug: `v_stripes` is intended to default to 1, but when the field is absent the subsequent integer check compares the default against the original zero value and rejects the material. In current code, `v_stripes` is effectively required.

## 8. Medium Integration

Media are scene-level definitions but are essential to dielectric material behavior.

```jsonc
{
  "media": {
    "glass": {
      "type": "homogeneous", // optional, default homogeneous
      "ior": { /* constant or cauchy; optional, default eta 1 */ },
      "sigma_a": "spectral parameter", // optional, default zero
      "sigma_s": "spectral parameter"  // optional, default zero
    }
  }
}
```

An object may define:

```jsonc
{
  "material_id": "glass-surface",
  "medium_boundary": {
    "outside": "medium name", // optional, default air
    "inside": "required medium name",
    "priority": "integer",    // optional, default 0
    "thin": "boolean"         // optional, default false
  }
}
```

Before BSDF evaluation, the medium stack resolves incident/transmitted media and fills eta values in `ShadingContext`. A sampled `TransmissionEvent` updates the stack unless the boundary is thin. Overlapping media use the highest priority; ties favor the most recently encountered entry.

Absorption follows Beer-Lambert attenuation for traveled arc length:

$$
T(\lambda,d)=\exp\!\left[-\sigma_a(\lambda)d\right].
$$

`sigma_s` is parsed, stored, and queryable, but no ray-tracing code currently consumes it. The Engine therefore implements homogeneous absorption but not volumetric scattering events.

## 9. Spectral, Dimensional, and Integrator Behavior

### 9.1 Spectral Processing

The render selects RGB, hero-wavelength, or sampled-wavelength mode. `ShadingContext` carries the active wavelength data to every spectral parameter and IOR model. Cauchy dielectric samples can propagate wavelength metadata on transmission. RGB authored data is uplifted when a wavelength representation is requested.

The generic `Spectrum` type does not freely combine RGB and sampled values. Most mixed-kind operations return zero unless a model explicitly performs a compatible uplift. Authors should use consistent spectral parameter forms within one model.

### 9.2 Dimension Support

- Lambert and perfect specular reflection intentionally support N-dimensional local directions.
- Specular dielectric reflection/refraction helpers preserve the input direction length.
- GGX sampling and evaluation use a 3-component local microfacet parameterization. They are correct for ordinary 3D surfaces and for Spherical surfaces whose intrinsic local frame has two tangents plus one normal. Euclidean render dimensions above three do not have a fully N-dimensional GGX model.
- Cylindrical grid placement is explicitly based on three world-space coordinates.
- Cell palette selection is dimension-generic because it scans every normal component.

### 9.3 Emission and Integrators

- The regular recursive path tracer stops at every emissive hit, even if the same material also has a surface.
- BDPT rejects non-Euclidean scenes and any surface whose flags include `NonReciprocal`; this excludes rough dielectric transmission and mixtures containing it.
- Area-light sampling requires both an emitter and a Shape implementing `SurfaceSampler` with positive area. Material emission alone does not make a Shape sampleable.
- Light-to-camera projection skips non-endpoint vertices whose surface advertises any delta flag. A mixed or cutout surface may therefore be excluded even when it also contains a continuous lobe.

## 10. Validation, Physical Limits, and Known Risks

The package contains numerical checks for non-negativity, reciprocity, energy conservation, and Sample/Eval/PDF consistency. These are utilities used by tests; the JSON factory does not run them on every loaded material.

| Topic | Current fact | Consequence |
| --- | --- | --- |
| Unknown fields | Generally ignored | Misspelled optional fields may silently use defaults |
| Spectral magnitudes | Non-negative but usually not capped at 1 | User input can create energy gain |
| Roughness zero | Squared, then alpha clamps to `1e-4` | Rough models never become true delta models |
| Rough reflection eta | Ignores boundary-resolved eta pair | Incorrect Fresnel is possible when exiting or nesting media |
| Rough transmission TIR | Returns an invalid sample instead of reflection | Paths can terminate when only the transmission lobe is present or selected |
| Rough transmission reciprocity | Marked `NonReciprocal` | BDPT rejects the entire scene |
| Combined surface and emission | Accepted, but emission wins in regular tracing | The surface is not sampled after an emissive hit |
| Cutout gaps | BSDF straight-through, not geometric holes | Shadow/visibility intersection behavior differs from alpha-tested geometry |
| `uv_klein.v_stripes` | Intended optional default is broken | Field is effectively required |
| Cell palette intensity | Scales palette but not grid color | Grid brightness may not track the requested intensity |
| ACEScg input | Stored without color-space conversion | Values are not transformed into the renderer's linear-sRGB working space |
| `sigma_s` | Parsed and stored but unused | No volume scattering despite a scattering coefficient in the schema |
| Material metadata | Mostly zero/default and not JSON-configurable | Do not treat it as a complete capability registry |

## 11. Maintenance Sources of Truth

When changing the material system, inspect at least:

1. `engine/controller/factory/materials.go` for all JSON discriminators, defaults, schemas, aliases, and recursive composition.
2. `engine/model/material/bxdf/` for physical evaluation, sampling, PDFs, eta resolution, and event flags.
3. `engine/model/material/bsdf/` for wrappers, mixtures, and procedural cutouts.
4. `engine/model/material/microfacet/` for GGX and Fresnel implementation details.
5. `engine/model/material/emission/` for emitter behavior and required shading context.
6. `engine/model/optics/spectrum_parameter/` for authored spectral input semantics.
7. `engine/model/material/medium/` and `engine/controller/factory/media.go` for IOR, absorption, boundaries, and stack priority.
8. `engine/ray_tracing/trace_ray.go`, `medium_transport.go`, `bdpt.go`, and `light_trace.go` for the behavior integrators actually apply.

Adding a BxDF or Emitter Go type does not make it JSON-loadable. It also requires a factory discriminator and schema. Event flags are part of the transport contract: they control delta handling, medium transitions, BDPT support, and light-tracing visibility behavior.
