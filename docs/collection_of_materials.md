# Collection of Materials

> Scope: this document describes only the current implementation under `engine/`. Studio schemas, example-only conventions, and older documentation are excluded.
>

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

## Real Category Tables

### Surface Discriminators

The first column is the logical surface-model key. JSON aliases are grouped into the same row instead of being presented as separate material types.

| Surface material type | JSON `surface.type` | Description and mathematical model | Input parameters | Actual runtime result | Category | Delta/event flags |
| --- | --- | --- | --- | --- | --- | --- |
| Weighted Mixture | `weighted_mixture` | Normalized statistical mixture of recursively defined surfaces, with $p_i=w_i/\sum\limits_j w_j$ and $f=\sum\limits_i p_i f_i$. | Non-empty `components`; each component has finite $w_i>0$ and a recursive `surface`. | `bsdf.WeightedMixture` | Probabilistic mixture | Union of component flags |
| Lambert | `lambert` | Ideal reciprocal diffuse reflector, $f=\rho(\lambda)/I_D$; in 3D, $I_3=\pi$. | Required spectral albedo $\rho(\lambda)\ge 0$. | `bsdf.Single{BxDF: bxdf.Lambert}` | Diffuse reflection | None |
| Specular Reflection | `specular_reflection` | Colored ideal mirror with a single deterministic reflected direction and no Fresnel model. | Spectral reflectance $R(\lambda)\ge 0$; optional, default $R=1$. | `bsdf.Single{BxDF: bxdf.SpecularReflection}` | Perfect mirror reflection | `DeltaReflection` |
| Specular Dielectric | `specular_dielectric` | Smooth dielectric interface selecting perfect reflection with probability $F$ and refraction with probability $1-F$; supports dispersion and total internal reflection. | Spectral $R(\lambda),T(\lambda)\ge0$; optional, default 1. Outside IOR $\eta_o>0$, default 1. Inside `ior` is constant or Cauchy; legacy $\eta_i>0$, default 1.5. | `bsdf.Single{BxDF: bxdf.SpecularDielectric}` | Perfect Fresnel reflection and refraction | `DeltaReflection`, `DeltaTransmission`, `TransmissionEvent` |
| Rough Conductor | `rough_conductor` | Reciprocal GGX microfacet reflection using complex spectral IOR $\eta(\lambda)+ik(\lambda)$ and $\alpha=\max(r^2,10^{-4})$. | Required spectral $\eta(\lambda),k(\lambda)\ge0$; roughness $r\in[0,1]$, default 0.25; spectral weight $W(\lambda)\ge0$, default 1. | `bsdf.Single{BxDF: bxdf.RoughConductor}` | GGX conductor reflection | None |
| Rough Dielectric Reflection | `rough_dielectric_reflection` | Reciprocal GGX dielectric reflection lobe with Fresnel modulation; it contains no transmission lobe. | Spectral $R(\lambda)\ge0$, default 1; $\eta_o>0$, default 1; constant or Cauchy inside `ior`; $r\in[0,1]$, default 0.25. | `bsdf.Single{BxDF: bxdf.RoughDielectricReflection}` | GGX dielectric reflection only | None |
| Rough Dielectric Transmission | `rough_dielectric_transmission` | Walter-style GGX dielectric transmission lobe for opposite hemispheres; it contains no reflection fallback. | Spectral $T(\lambda)\ge0$, default 1; $\eta_o>0$, default 1; constant or Cauchy inside `ior`; $r\in[0,1]$, default 0.25. | `bsdf.Single{BxDF: bxdf.RoughDielectricTransmission}` | GGX dielectric transmission only | `TransmissionEvent`, `NonReciprocal` |
| Cylindrical Grid Cutout / Wire Mesh | `cylindrical_grid_cutout`, `wire_mesh` | Procedural cylindrical-coordinate mask: grid lines delegate to `line_surface`, while gaps are deterministic straight-through delta transmission. | Recursive `line_surface`; 3-vectors $o$, axis $a\ne0$, and reference axis; widths $w_l,w_g,h_g\ge0$; reference radius $r_{ref}>0$. All are optional and have documented defaults. | `bsdf.CylindricalGridCutout` | Spatial line BSDF plus transparent gaps | Always `DeltaTransmission`, plus line-surface flags |

There are nine JSON surface values but only eight distinct runtime surface constructions because `wire_mesh` is an alias.

### Emission Discriminators

The first column is the logical emission-model key. An emission model may be the only component of a material or may coexist with a surface model.

| Emission material type | JSON `emission.type` | Description and mathematical model | Input parameters | Runtime type | Spatial rule | `IsDelta()` |
| --- | --- | --- | --- | --- | --- | --- |
| Constant Emission | `constant` | Spatially and directionally invariant spectral radiance, $L_e(x,\omega_o,\lambda)=L(\lambda)$. | Required spectral radiance $L(\lambda)\ge0$ in `radiance`; `color` is a fallback alias. | `emission.Constant` | Constant over position and outgoing direction | False |
| Cell Palette Emission | `cell_palette` | Diagnostic emission indexed by the dominant signed normal axis, $i=2\operatorname*{arg\,max}_j\lvert n_j\rvert+\mathbf{1}_{n_i>0}$, with optional boundary-grid replacement. | Non-empty RGB `palette` $C_i\in\mathbb{R}_{\ge0}^3$; intensity $s\ge0$; `shading`; RGB `grid_color`; grid thickness $t\ge0$. All are optional with documented defaults. | `emission.CellPalette` | Dominant normal axis/sign, optionally modified near AABB boundaries | False |
| UV Klein Emission | `uv_klein` | Diagnostic Klein-bottle parameter visualization: hue follows wrapped $u$, and $2N$ alternating lightness bands follow wrapped $v$. | Saturation $S\in[0,1]$, default 1; lightness $L\in[0,1]$, default 0.55; positive integer $N=$ `v_stripes`; intensity $s\ge0$, default 1. `v_stripes` is effectively required by the current parser bug. | `emission.UVKlein` | HSL mapping of Klein-bottle UV coordinates | False |

### Medium Discriminators

Media are scene-level optical models rather than subtypes of `material.Material`, but they participate directly in dielectric boundary resolution and path throughput. The first column is the logical medium-model key.

| Medium type | JSON `media.<name>.type` | Description and mathematical model | Input parameters | Runtime type | Current transport support |
| --- | --- | --- | --- | --- | --- |
| Homogeneous Medium | `homogeneous` or omitted | Spatially invariant IOR and extinction coefficients. Absorption over arc length $d$ is $T(\lambda,d)=\exp[-\sigma_a(\lambda)d]$; extinction data also includes $\sigma_s(\lambda)$. | Optional `ior`: constant $\eta>0$ or Cauchy $\eta(\lambda)=A+B/\lambda^2+C/\lambda^4$, default $\eta=1$. Optional spectral $\sigma_a(\lambda),\sigma_s(\lambda)\ge0$, both default 0. | `medium.Homogeneous` | IOR boundary transitions and Beer-Lambert absorption are active. $\sigma_s$ is parsed and stored, but volumetric scattering events are not implemented. |

### Supporting Tagged Types

| Input layer | JSON values | Runtime implementations |
| --- | --- | --- |
| Spectral parameter | `rgb`, `constant`, `sampled`, `blackbody` | `RGBParameter`, `ConstantParameter`, `SampledParameter`, `BlackbodyParameter` |
| IOR model | `constant`, `cauchy` | `medium.Constant`, `medium.Cauchy` |

### Common Material Schema and Runtime Contract

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

### Shared Spectral Parameter Schema

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

### Shared IOR Schema

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

## Surface Models

### Weighted Mixture

#### Definition, Properties, and Model

A weighted mixture is a statistical BSDF mixture over one or more component surfaces. Let $w_i>0$ be the authored component weights and define

$$
p_i=\frac{w_i}{\sum\limits_jw_j}.
$$

The normalized model is

$$
f=\sum\limits_i p_i f_i,
\qquad
p_\omega=\sum\limits_i p_i p_{\omega,i}.
$$

It is a probabilistic blend, not an additive layer stack. It does not model coating order, multiple internal reflections, or Fresnel-aware lobe-selection probabilities. Parsing is recursive, so a component may itself be a mixture or procedural cutout.

#### Implementation Logic and Mathematical Process

Sampling selects component $i$ with probability $p_i$ and remaps the random number into that component's interval. For a non-delta sample, the final value and PDF are recomputed from the complete mixture. For a delta sample, only the selected component contributes and both its value and discrete PDF are multiplied by $p_i$.

Flags are bitwise-unioned. Roughness reports the largest component alpha and is delta only when every active component is delta. Consequently, a `NonReciprocal` or transmission flag in any component propagates to the complete mixture and can affect integrator compatibility.

#### Parameters and Schema

- `components` must be non-empty.
- Every $w_i$ must be finite and strictly positive.
- `surface` accepts any recursively parseable surface model.

```jsonc
{
  "type": "weighted_mixture",
  "components": [{
    "weight": "finite number > 0",
    "surface": { "type": "any surface type" }
  }]
}
```

### Lambert

#### Definition, Properties, and Model

A Lambert surface is an ideal diffuse reflector with direction-independent BRDF over the upper hemisphere. It is reciprocal, non-delta, and parameterized by spectral albedo $\rho$. For local direction dimension $D$, define the projected-hemisphere normalization

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

In 3D, $I_3=\pi$, giving the standard $\rho/\pi$ law.

#### Implementation Logic and Mathematical Process

`Eval` returns $\rho/I_D$ only when both directions lie in the upper local hemisphere. `PDF` returns $\cos\theta_i/I_D$ on the same domain. Sampling uses a cosine-weighted N-dimensional hemisphere construction, so the throughput factor simplifies to the albedo:

$$
\frac{\rho}{I_D}
\frac{\cos\theta_i}{\cos\theta_i/I_D}
=\rho.
$$

This dedicated N-dimensional sampler makes Lambert one of the surfaces intentionally supporting local dimensions above three.

#### Parameters and Schema

- `albedo` is a required non-negative spectral parameter $\rho(\lambda)$.
- Values above one are syntactically accepted but can violate energy conservation.

```jsonc
{
  "type": "lambert",
  "albedo": "spectral parameter; required"
}
```

### Specular Reflection

#### Definition, Properties, and Model

This model is a colored ideal mirror without a Fresnel term. It is a reciprocal delta-reflection distribution: all energy lies at one reflected direction, so its ordinary solid-angle function is zero almost everywhere. Reflection preserves the normal component and negates every tangential component of $\omega_o$.

#### Implementation Logic and Mathematical Process

The deterministic discrete sample is

$$
\omega_i=\operatorname{reflect}(\omega_o),
\qquad
f_{\mathrm{sample}}=\frac{R}{|\cos\theta_i|},
\qquad
p_{\mathrm{sample}}=1,
$$

where $R$ is the spectral reflectance.

The path-throughput cosine factor cancels the division. `Eval` and solid-angle `PDF` return zero because the distribution is a delta. The reflection logic supports arbitrary local direction dimension.

#### Parameters and Schema

- `reflectance` is a non-negative spectral parameter $R(\lambda)$.
- Its default is the constant value one.
- No IOR or Fresnel model is evaluated.

```jsonc
{
  "type": "specular_reflection",
  "reflectance": "spectral parameter" // optional, default constant 1
}
```

### Specular Dielectric

#### Definition, Properties, and Model

This model is an ideal smooth dielectric interface with mutually exclusive delta reflection and delta transmission. It supports constant or dispersive IOR, medium-boundary transitions, and total internal reflection. For incident index $\eta_i$, transmitted index $\eta_t$, and incident cosine $c$, unpolarized dielectric Fresnel is

$$
F=\frac{r_{\parallel}^2+r_{\perp}^2}{2}.
$$

Snell refraction satisfies

$$
\sin\theta_t=\frac{\eta_i}{\eta_t}\sin\theta_i.
$$

Total internal reflection gives $F=1$.

#### Implementation Logic and Mathematical Process

Sampling chooses reflection with probability $F$ and transmission with probability $1-F$:

- reflection: $f_{\mathrm{sample}}=RF/|\cos\theta_i|$ and $p=F$;
- transmission: $f_{\mathrm{sample}}=T(1-F)/|\cos\theta_i|$ and $p=1-F$.

If refraction fails, sampling falls back to reflection with probability 1. A transmission sample carries `TransmissionEvent`, the transmitted-side eta, and `ctx.TransmitMedium`. With a dispersive Cauchy IOR and an active wavelength, the sampled wavelength is propagated in the event.

The implementation does not apply an additional eta-squared radiance factor in this delta model.

#### Parameters and Schema

- `reflectance` and `transmittance` are non-negative spectral multipliers with default one.
- `eta_outside` is positive and defaults to one.
- `ior` accepts the shared constant or Cauchy schema; legacy `eta_inside` is used only when `ior` is absent.
- An active `medium_boundary` can override the fallback incident/transmitted IOR pair.

```jsonc
{
  "type": "specular_dielectric",
  "reflectance": "spectral parameter",   // optional, default 1
  "transmittance": "spectral parameter", // optional, default 1
  "eta_outside": 1,                       // optional
  "ior": { /* constant or cauchy */ }      // optional; eta_inside fallback supported
}
```

### Rough Conductor

#### Definition, Properties, and Model

This is a reciprocal, non-delta microfacet reflection model for a conductor with complex spectral IOR $\eta+ik$. The GGX distribution models unresolved surface normals, conductor Fresnel supplies wavelength-dependent reflection, and the scalar/spectral weight $W$ scales the lobe.

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

#### Implementation Logic and Mathematical Process

The factory maps perceptual roughness $r$ to $\alpha=r^2$; GGX then clamps $\alpha$ to $[10^{-4},1]$. Roughness zero is therefore still a very sharp non-delta GGX lobe, not a perfect mirror.

Sampling uses GGX visible-normal sampling, reflects $\omega_o$ about the sampled microfacet normal, and transforms visible-normal density with $1/(4|\omega_o\cdot\omega_h|)$.

`Eval` and `PDF` reject directions outside the reflection hemispheres, and `AlbedoBound` uses the spectral weight as its conservative throughput control.

#### Parameters and Schema

- `eta` is the required non-negative real part $\eta(\lambda)$ of the conductor IOR.
- `k` is the required non-negative extinction coefficient $k(\lambda)$.
- `roughness` is $r\in[0,1]$, defaulting to $0.25$.
- `weight` is a non-negative spectral factor $W(\lambda)$, defaulting to one.

```jsonc
{
  "type": "rough_conductor",
  "eta": "spectral parameter; required",
  "k": "spectral parameter; required",
  "roughness": "number in [0,1]", // optional, default 0.25
  "weight": "spectral parameter"  // optional, default 1
}
```

### Rough Dielectric Reflection

#### Definition, Properties, and Model

This model is the reciprocal, non-delta reflection lobe of a rough dielectric interface. It contains no transmission lobe. GGX describes the microfacet normals and scalar dielectric Fresnel modulates reflection:

$$
f(\omega_i,\omega_o)
=
R\,F_{\mathrm{dielectric}}
\frac{D(\omega_h)G(\omega_i,\omega_o)}
{4\cos\theta_i\cos\theta_o}.
$$

It never produces transmission and is useful as a clearcoat/glaze reflection component in a mixture. Its effective roughness parameter is $\alpha=\max(r^2,10^{-4})$.

#### Implementation Logic and Mathematical Process

The implementation samples a GGX visible normal, reflects $\omega_o$ about it, evaluates the dielectric Fresnel term, and applies the reflection half-vector Jacobian. `Eval`, `Sample`, and `PDF` use the same GGX $D$ and Smith $G$ functions as `rough_conductor`.

Current limitation: Fresnel always uses the configured outside and inside IOR values. It does not use `ctx.EtaIncident` and `ctx.EtaTransmit`, so exiting-interface or nested-medium reflection can use the wrong eta orientation.

#### Parameters and Schema

- `reflectance` is a non-negative spectral factor $R(\lambda)$, default one.
- `eta_outside` is positive and defaults to one.
- `ior` accepts the shared constant or Cauchy model.
- `roughness` is $r\in[0,1]$, default $0.25$, with effective $\alpha=\max(r^2,10^{-4})$.

```jsonc
{
  "type": "rough_dielectric_reflection",
  "reflectance": "spectral parameter", // optional, default 1
  "eta_outside": 1,
  "ior": { /* constant or cauchy */ },
  "roughness": "number in [0,1]" // optional, default 0.25
}
```

### Rough Dielectric Transmission

#### Definition, Properties, and Model

This model is the rough dielectric transmission lobe only. It is a non-delta, non-reciprocal radiance-transport model and never produces reflection. Directions $\omega_i$ and $\omega_o$ must lie in opposite hemispheres. With $\eta=\eta_t/\eta_i$, the Walter-style transmission half-vector is

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

#### Implementation Logic and Mathematical Process

Sampling draws a GGX visible normal and refracts through it. Refraction failure or total internal reflection returns an invalid sample; this model does not fall back to reflection. Use `specular_dielectric` for a combined perfect interface, or combine rough reflection and transmission explicitly while accounting for the fact that mixture selection is not Fresnel-aware.

The event is marked `TransmissionEvent|NonReciprocal`. Consequently, any scene containing this surface, directly or inside a mixture, is rejected by the current BDPT support check.

#### Parameters and Schema

- `transmittance` is a non-negative spectral factor $T(\lambda)$, default one.
- `eta_outside` is positive and defaults to one.
- `ior` accepts the shared constant or Cauchy model.
- `roughness` is $r\in[0,1]$, default $0.25$, with effective $\alpha=\max(r^2,10^{-4})$.
- Active medium-boundary data supplies $\eta_i$, $\eta_t$, and the transmitted medium.

```jsonc
{
  "type": "rough_dielectric_transmission",
  "transmittance": "spectral parameter", // optional, default 1
  "eta_outside": 1,
  "ior": { /* constant or cauchy */ },
  "roughness": "number in [0,1]" // optional, default 0.25
}
```

### Cylindrical Grid Cutout / Wire Mesh

#### Definition, Properties, and Model

This is a procedural BSDF mask on cylindrical coordinates, not geometric displacement. It alternates a recursively defined `line_surface` with perfectly transparent gaps. `wire_mesh` is an exact parser alias of `cylindrical_grid_cutout`.

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

The model is spatially discontinuous at line/gap boundaries. The line region inherits all properties and flags of `line_surface`; the gap is a delta straight-through event.

#### Implementation Logic and Mathematical Process

On a line, Eval/Sample/PDF delegate to `line_surface`. In a gap, Sample returns deterministic straight-through transmission $\omega_i=-\omega_o$, value $1/|\cos\theta_i|$, and discrete PDF one. Gap transmission has `DeltaTransmission` but deliberately lacks `TransmissionEvent`, so it does not modify the medium stack.

The pattern uses the first three world-space hit coordinates. If fewer than three are available, every point is treated as a line. This is a BSDF mask, not geometric displacement: gaps remain intersection surfaces for visibility and shadow rays unless the transport path samples through them.

The factory normalizes `axis`, projects `reference_axis` into its orthogonal plane, normalizes the result, and derives the bitangent by a cross product. The default line surface is a silver-like rough conductor with hard-coded optical constants.

#### Parameters and Schema

- `origin`, `axis`, and `reference_axis` define the cylindrical coordinate frame in 3D.
- `line_width`, `gap_width`, and `gap_height` are non-negative.
- `reference_radius` is strictly positive and converts angle to arc coordinate $s$.
- `line_surface` is optional and recursively accepts any surface model.

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

Default `line_surface`:

```jsonc
{
  "type": "rough_conductor",
  "eta": [0.15, 0.14, 0.13],
  "k": [4.1, 3.5, 2.7],
  "roughness": 0.22,
  "weight": [0.88, 0.90, 0.92]
}
```

## Emission Models

### Constant Emission

#### Definition, Properties, and Model

Constant emission is a spatially and directionally invariant radiance model:

$$
L_e(x,\omega_o,\lambda)=L(\lambda).
$$

It is non-delta and does not impose one-sided emission at the emitter level.

#### Implementation Logic and Mathematical Process

`Emit` ignores outgoing direction and interaction coordinates, evaluates the shared spectral parameter in the active spectral context, and returns it directly. Any visibility, surface-area sampling, or sidedness behavior belongs to the Shape and integrator rather than this emitter.

#### Parameters and Schema

- `radiance` is the required non-negative spectral function $L(\lambda)$.
- `color` is accepted only as a fallback alias when `radiance` is absent.
- If both fields exist, `radiance` takes precedence.

```jsonc
{
  "type": "constant",
  "radiance": "spectral parameter; required"
  // color is accepted as a fallback alias when radiance is absent
}
```

### Cell Palette Emission

#### Definition, Properties, and Model

Cell-palette emission is a diagnostic, direction-independent emitter that assigns color from the dominant signed axis of the geometric normal. The default palette contains eight colors for `-X,+X,-Y,+Y,-Z,+Z,-W,+W`. For

$$
a=\operatorname*{arg\,max}_j|n_j|,
$$

the palette index is

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

The model is intended for cell/face orientation visualization rather than physical illumination.

#### Implementation Logic and Mathematical Process

In `boundary_grid` mode, non-dominant coordinates are compared with the object's AABB faces. A point within `grid_thickness` of any such boundary emits `grid_color`; otherwise it emits the cell color. Grid evaluation therefore depends on valid hit position, geometric normal, and Shape AABB data in `ShadingContext`.

Intensity scales palette colors but does not scale `grid_color`. `grid_color` and `grid_thickness` are parsed only in boundary-grid mode.

#### Parameters and Schema

- `palette` is an optional non-empty list of non-negative RGB triples.
- `intensity` is non-negative.
- `shading` is `solid` or `boundary_grid`, defaulting to `solid`.
- `grid_color` defaults to white and is not multiplied by `intensity`.
- `grid_thickness` is a non-negative world-space distance with default $0.02$.

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

### UV Klein Emission

#### Definition, Properties, and Model

UV Klein emission is a diagnostic parameter-space visualization for Klein-bottle geometry. It assumes $(u,v)$ are angular coordinates in radians. Wrapped $u$ controls cyclic hue, while $v$ controls alternating light and dark stripes. It is direction-independent, non-delta, and intentionally non-physical.

#### Implementation Logic and Mathematical Process

The wrapped angular coordinates are

$$
\hat{u}=\operatorname{frac}\!\left(\frac{u}{2\pi}\right),
\qquad
\hat{v}=\operatorname{frac}\!\left(\frac{v}{2\pi}\right).
$$

The hue is $H=360\hat{u}$ degrees. The stripe index alternates across $2N$ bands per $v$ cycle, where $N=\mathtt{v\_stripes}$. Even and odd bands use lightness $L$ and $0.45L$, respectively. The HSL result is converted to RGB and multiplied by `intensity`.

Current parser bug: `v_stripes` is intended to default to 1, but when the field is absent the subsequent integer check compares the default against the original zero value and rejects the material. In current code, `v_stripes` is effectively required.

#### Parameters and Schema

- `saturation` lies in $[0,1]$ and defaults to one.
- `lightness` lies in $[0,1]$ and defaults to $0.55$.
- `v_stripes` is a positive integer and is effectively required by the current parser bug.
- `intensity` is non-negative and defaults to one.

```jsonc
{
  "type": "uv_klein",
  "saturation": "number in [0,1]", // optional, default 1
  "lightness": "number in [0,1]",  // optional, default 0.55
  "v_stripes": "positive integer",
  "intensity": "non-negative number" // optional, default 1
}
```

## Medium Integration

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

## Spectral, Dimensional, and Integrator Behavior

### Spectral Processing

The render selects RGB, hero-wavelength, or sampled-wavelength mode. `ShadingContext` carries the active wavelength data to every spectral parameter and IOR model. Cauchy dielectric samples can propagate wavelength metadata on transmission. RGB authored data is uplifted when a wavelength representation is requested.

The generic `Spectrum` type does not freely combine RGB and sampled values. Most mixed-kind operations return zero unless a model explicitly performs a compatible uplift. Authors should use consistent spectral parameter forms within one model.

### Dimension Support

- Lambert and perfect specular reflection intentionally support N-dimensional local directions.
- Specular dielectric reflection/refraction helpers preserve the input direction length.
- GGX sampling and evaluation use a 3-component local microfacet parameterization. They are correct for ordinary 3D surfaces and for Spherical surfaces whose intrinsic local frame has two tangents plus one normal. Euclidean render dimensions above three do not have a fully N-dimensional GGX model.
- Cylindrical grid placement is explicitly based on three world-space coordinates.
- Cell palette selection is dimension-generic because it scans every normal component.

### Emission and Integrators

- The regular recursive path tracer stops at every emissive hit, even if the same material also has a surface.
- BDPT rejects non-Euclidean scenes and any surface whose flags include `NonReciprocal`; this excludes rough dielectric transmission and mixtures containing it.
- Area-light sampling requires both an emitter and a Shape implementing `SurfaceSampler` with positive area. Material emission alone does not make a Shape sampleable.
- Light-to-camera projection skips non-endpoint vertices whose surface advertises any delta flag. A mixed or cutout surface may therefore be excluded even when it also contains a continuous lobe.
