# Spectra and Color Spaces

## Mathematical and Engine Summary

| Layer | Mathematical object | Engine representation | Direction of information flow |
| --- | --- | --- | --- |
| Physical spectrum | A scalar function of wavelength, such as spectral radiance $L_\lambda(\lambda)$ or reflectance $\rho(\lambda)$ | Active-wavelength transport and mandatory 64-bin Engine Film accumulation | Engine transport $\rightarrow$ Studio observer integration |
| CIE 1931 XYZ | Three linear observer responses $(X,Y,Z)$ | Studio analytic color-matching-function approximation for Film output | Spectrum $\rightarrow$ XYZ; XYZ $\leftrightarrow$ output RGB spaces |
| Linear sRGB | Three linear-light coefficients relative to sRGB primaries | Authored input space and Studio display-linear output | Authored RGB uplift; XYZ $\rightarrow$ display RGB |
| sRGB | Nonlinear display-oriented encoding of linear sRGB | Accepted only as an authored spectral-parameter space and decoded to linear values | sRGB input $\rightarrow$ linear sRGB |
| ACEScg | Three linear AP1-primary coefficients | Studio output transform; also accepted as an authored label | XYZ $\leftrightarrow$ ACEScg |
| Film encoding | Spectral planes over 380–750 nm | `camera.Film` v3 | Transport result $\rightarrow$ physical spectral persistence |
| Display mapping | Exposure, tone mapping, clipping, and power-law gamma | `studio/film.ToImage` | Spectral Film $\rightarrow$ XYZ $\rightarrow$ linear sRGB $\rightarrow$ 8-bit RGB |

The central pipeline is:

```text
physical or reconstructed spectrum S(lambda)
    -> CIE 1931 observer integration
    -> XYZ
    -> selected Studio output space: XYZ / linear sRGB / ACEScg
    -> conversion back to linear sRGB for image output
    -> exposure -> tone map -> clamp -> power-law gamma -> 8-bit RGB

authored sRGB -> sRGB decoding -> renderer linear RGB
authored linear RGB -> approximate spectral uplift when a wavelength is active
```

The forward spectral mapping is many-to-one. The backward RGB-to-spectrum arrow is therefore an approximation, not an inverse.

## Physical Spectrum

### Spectral Distributions

A spectrum is a quantity resolved by wavelength. For visible-light rendering, the most common functions are:

| Quantity | Symbol | Physical interpretation | Typical physical constraint |
| --- | --- | --- | --- |
| Spectral radiance | $L_\lambda(x,\omega,\lambda)$ | Power carried at point $x$ in direction $\omega$, per projected area, solid angle, and wavelength | $L_\lambda\geq0$ |
| Spectral power distribution | $S(\lambda)$ | Generic wavelength-dependent emission or signal | $S\geq0$ |
| Spectral reflectance | $\rho(\lambda)$ | Fraction of incident energy reflected at each wavelength | $0\leq\rho\leq1$ for a passive surface |
| Spectral transmittance | $\tau(\lambda)$ | Fraction transmitted at each wavelength | $0\leq\tau\leq1$ for a passive interface |
| Absorption coefficient | $\sigma_a(\lambda)$ | Extinction per unit distance | $\sigma_a\geq0$ |
| Refractive index | $\eta(\lambda)$ | Phase-velocity ratio controlling refraction and Fresnel terms | $\eta>0$ |
| Extinction coefficient | $k(\lambda)$ | Imaginary part of a conductor's complex IOR | $k\geq0$ |

In physical radiometry, spectral radiance may use units such as $\mathrm{W\,m^{-2}\,sr^{-1}\,nm^{-1}}$. The Engine does not attach units or perform radiometric calibration. Its spectral values are non-negative numerical weights, so all authored distances, absorption coefficients, emission scales, and exposure values must be mutually consistent.

The wavelength-resolved surface rendering equation is

$$
L_o(x,\omega_o,\lambda)
=L_e(x,\omega_o,\lambda)
+\int\limits_{\mathcal H(x)}
f_s(x,\omega_i,\omega_o,\lambda)
L_i(x,\omega_i,\lambda)
|n\cdot\omega_i|\,d\omega_i.
$$

For an ideal diffuse reflector under spectral irradiance $E(\lambda)$, this reduces to

$$
L_o(\lambda)=\frac{\rho(\lambda)E(\lambda)}{\pi}.
$$

This product explains why a material does not possess one absolute displayed RGB color: its observed signal depends on both reflectance and illumination. An RGB value bakes together assumptions about the illuminant, observer, and reconstruction method that a measured reflectance spectrum keeps separate.

The Engine's spectral interval is

$$
\Lambda=[\lambda_{\min},\lambda_{\max}]
=[380,750]\ \mathrm{nm},
\qquad
\Delta\lambda=370\ \mathrm{nm}.
$$

Runtime sampling clamps wavelengths just inside this interval. The observer function itself returns zero at and outside the exact endpoints, so its effective support is $380<\lambda<750$ nm.

### Why Spectra Matter in Ray Tracing

RGB is sufficient only when transport can be represented by three fixed linear coefficients. A wavelength-resolved path is needed when the path geometry or throughput changes with wavelength. The current Engine contains three important examples:

- Cauchy dispersion changes $\eta(\lambda)$ and therefore changes refracted ray directions.
- Sampled absorption changes Beer--Lambert attenuation by wavelength.
- Blackbody emission provides a wavelength-dependent source distribution.

The homogeneous-medium transmittance over distance $d$ is

$$
T(\lambda,d)=\exp\!\left[-\sigma_a(\lambda)d\right].
$$

The scene distance unit must therefore be the reciprocal unit of $\sigma_a$. The parser does not enforce that relationship.

The Cauchy IOR model evaluates wavelength in micrometres:

$$
\eta(\lambda)
=A+\frac{B}{\lambda_{\mu\mathrm m}^{2}}
+\frac{C}{\lambda_{\mu\mathrm m}^{4}},
\qquad
\lambda_{\mu\mathrm m}=\frac{\lambda_{\mathrm{nm}}}{1000}.
$$

If no valid wavelength is active, the IOR implementation uses $550$ nm.

### Blackbody Radiation

Planck's wavelength law is proportional to

$$
B_\lambda(\lambda,T)
\propto
\frac{1}{\lambda^5\left(\exp\!\left(\frac{c_2}{\lambda T}\right)-1\right)},
\qquad
c_2=1.438776877\times10^7\ \mathrm{nm\,K}.
$$

The common factor cancels because the Engine normalizes the result at $560$ nm:

$$
S_{\mathrm{bb}}(\lambda;T,s)
=s\,
\frac{B_\lambda(\lambda,T)}{B_\lambda(560\ \mathrm{nm},T)}.
$$

This is a relative spectral shape, not absolutely calibrated blackbody radiance. In RGB mode, the Engine does not integrate Planck's law through the observer model; it instead uses an approximate color-temperature-to-sRGB algorithm and decodes that result to linear sRGB. RGB and spectral blackbody renders are therefore related but not mathematically identical.

### Runtime Spectral Value

`optics.Spectrum` has two real storage kinds:

| Kind | Stored data | Meaning |
| --- | --- | --- |
| `SpectrumKindRGB` | `RGB [3]float64` | Renderer-space scene-linear sRGB coefficients |
| `SpectrumKindSampled` | `Samples []float64` | Values aligned with the wavelengths in the current `ShadingContext` |

A sampled `Spectrum` does not store its own wavelength coordinates. Its samples are meaningful only beside the context that supplied `WavelengthsNM`. In the current renderer, a traced spectral ray activates one wavelength at a time, so even `sampled` render mode normally produces a one-element sampled `Spectrum` per path. That mode traces a batch of independent monochromatic paths; it is not a packet tracer carrying several wavelengths through one geometric path.

Mixed RGB/sampled arithmetic is deliberately restricted. Sampled-plus-sampled operations align by array index, missing entries become zero, and most nonzero RGB/sampled combinations return a zero `Spectrum` unless a model explicitly uplifts the RGB value. This makes a consistent spectral representation within one model important.

## Colorimetry and Color Spaces

### Tristimulus Color and Metamerism

A color space is not a physical spectrum. Human trichromatic color matching compresses a spectral function into three observer responses. For color-matching functions $\overline{x}(\lambda)$, $\overline{y}(\lambda)$, and $\overline{z}(\lambda)$, the ideal linear mapping is

$$
\begin{bmatrix}X\\Y\\Z\end{bmatrix}
=k\int\limits_{\Lambda}
S(\lambda)
\begin{bmatrix}
\overline{x}(\lambda)\\
\overline{y}(\lambda)\\
\overline{z}(\lambda)
\end{bmatrix}
\,d\lambda.
$$

The scalar $k$ determines normalization. Because an entire function is reduced to three numbers, infinitely many spectra can produce the same XYZ value. Such spectra are metamers under that observer and viewing condition. Consequently:

- spectrum-to-XYZ conversion loses information;
- XYZ or RGB cannot uniquely recover the source spectrum;
- two RGB-matched materials can behave differently under a different illuminant or after wavelength-dependent transport.

```
Physical world
     │
     │ spectrum S(λ)
     ▼
┌───────────────────┐
│ Human color vision│
└───────────────────┘
     │
     │ tristimulus compression
     ▼
     XYZ
      │
      │ coordinate / basis transform
      ├───────────────┐
      ▼               ▼
 linear sRGB        ACEScg
      │
      ▼
 encoded sRGB
      │
      ▼
   Display
```

### Engine CIE 1931 Approximation

The Engine does not use a tabulated CIE dataset. It evaluates an analytic approximation. Define the asymmetric Gaussian

$$
G(\lambda;\mu,a_l,a_r)
=\exp\!\left[-\frac{1}{2}
\left((\lambda-\mu)
\begin{cases}
a_l,&\lambda<\mu,\\
a_r,&\lambda\geq\mu
\end{cases}
\right)^2\right].
$$

Before non-negative clamping, the implemented functions are

$$
\begin{aligned}
\overline{x}(\lambda)
&=0.362G(\lambda;442.0,0.0624,0.0374)
+1.056G(\lambda;599.8,0.0264,0.0323)\\
&\quad-0.065G(\lambda;501.1,0.0490,0.0382),\\
\overline{y}(\lambda)
&=0.821G(\lambda;568.8,0.0213,0.0247)
+0.286G(\lambda;530.9,0.0613,0.0322),\\
\overline{z}(\lambda)
&=1.217G(\lambda;437.0,0.0845,0.0278)
+0.681G(\lambda;459.0,0.0385,0.0725).
\end{aligned}
$$

Each component is clamped to at least zero. XYZ in this renderer is therefore a numerical CIE-like observer space, not a claim of exact conformance to a particular tabulated standard-observer dataset.

$Y$ is the luminance-like channel of XYZ. Chromaticity may be formed when $X+Y+Z>0$:

$$
x=\frac{X}{X+Y+Z},
\qquad
y=\frac{Y}{X+Y+Z},
\qquad
z=1-x-y.
$$

The Engine does not expose chromaticity as a runtime color type; this relation is useful for analysis only.

### Linear sRGB and sRGB

An RGB color space is defined by at least:

- three chromatic primaries, which determine the basis and gamut;
- a reference white, which determines the neutral axis;
- a transfer function, which distinguishes linear-light values from encoded values.

Changing only the transfer function does not change the primaries: linear sRGB and encoded sRGB share primaries and D65 white, but their channel numbers have different meanings. XYZ is different: it is an observer-derived tristimulus basis rather than a display-primary RGB space. A chromatic adaptation transform is additionally required when values are moved between spaces whose reference whites differ while preserving perceived neutrality.

Linear sRGB is a linear-light RGB space with sRGB primaries and a D65 white point. Engine accepts it for authored RGB parameters and approximate spectral uplift; Studio uses it as the default display-linear output. The Film itself has no RGB space.

The authored `srgb` spectral-parameter form is decoded channel by channel:

$$
C_{\mathrm{linear}}=
\begin{cases}
\dfrac{C_{\mathrm{srgb}}}{12.92},
&C_{\mathrm{srgb}}\leq0.04045,\\[6pt]
\left(\dfrac{C_{\mathrm{srgb}}+0.055}{1.055}\right)^{2.4},
&C_{\mathrm{srgb}}>0.04045.
\end{cases}
$$

The output `gamma` option is not the inverse sRGB transfer function. It is the generic power law $C^{1/\gamma}$. Thus `gamma: 2.2` is only an approximation to an sRGB display encoding.

### CIE XYZ and Linear sRGB Matrices

The Engine uses the following direct transforms:

$$
\begin{bmatrix}X\\Y\\Z\end{bmatrix}
=
\begin{bmatrix}
0.4124564&0.3575761&0.1804375\\
0.2126729&0.7151522&0.0721750\\
0.0193339&0.1191920&0.9503041
\end{bmatrix}
\begin{bmatrix}R_s\\G_s\\B_s\end{bmatrix},
$$

$$
\begin{bmatrix}R_s\\G_s\\B_s\end{bmatrix}
=
\begin{bmatrix}
3.2404542&-1.5371385&-0.4985314\\
-0.9692660&1.8760108&0.0415560\\
0.0556434&-0.2040259&1.0572252
\end{bmatrix}
\begin{bmatrix}X\\Y\\Z\end{bmatrix}.
$$

Matrix conversion can produce negative RGB components for colors outside the destination gamut. Core matrix functions preserve those negatives. Some helper conversions and final 8-bit output clamp them, so conversion and display are not globally invertible once clamping occurs.

At the Go type level, `RGB`, `XYZ`, and `ACEScg` are aliases of the same `Color3` array type. The type system does not prevent one space from being passed as another; semantic correctness depends on using the explicit conversion functions at each boundary.

### ACEScg

ACEScg is a linear working space based on AP1 primaries. The Engine uses:

$$
\begin{bmatrix}R_a\\G_a\\B_a\end{bmatrix}
=
\begin{bmatrix}
1.6410233797&-0.3248032942&-0.2364246952\\
-0.6636628587&1.6153315917&0.0167563477\\
0.0117218943&-0.0082844420&0.9883948585
\end{bmatrix}
\begin{bmatrix}X\\Y\\Z\end{bmatrix},
$$

$$
\begin{bmatrix}X\\Y\\Z\end{bmatrix}
=
\begin{bmatrix}
0.6624541811&0.1340042065&0.1561876870\\
0.2722287168&0.6740817658&0.0536895174\\
-0.0055746495&0.0040607335&1.0103391003
\end{bmatrix}
\begin{bmatrix}R_a\\G_a\\B_a\end{bmatrix}.
$$

Two limitations are essential:

1. Linear-sRGB conversion is chained directly through XYZ. There is no explicit D65-to-D60 chromatic adaptation between the sRGB and ACEScg conventions.
2. A material parameter tagged with `"space": "acescg"` is stored unchanged. It is not transformed from ACEScg into the renderer's scene-linear sRGB representation before shading.

ACEScg film storage therefore works as the implemented matrix transform, but the project does not yet provide a complete color-management pipeline. The `aces` tone-map option is also not ACEScg and is not an ACES RRT/ODT; it is only a per-channel fitted curve described below.

## Spectrum-to-Color Mapping

### White-Point-Normalized Observer Model

Let

$$
\mathbf{c}(\lambda)
=\begin{bmatrix}\overline{x}(\lambda)&\overline{y}(\lambda)&\overline{z}(\lambda)\end{bmatrix}^{\mathsf T}
$$

and let the mean observer response over the Engine interval be

$$
\overline{\mathbf{c}}
=\frac{1}{\Delta\lambda}
\int\limits_{\lambda_{\min}}^{\lambda_{\max}}
\mathbf{c}(\lambda)\,d\lambda.
$$

The code approximates this mean with 2,048 midpoint samples. Its target white is the D65 XYZ vector

$$
\mathbf{d}_{65}=
\begin{bmatrix}0.95047&1&1.08883\end{bmatrix}^{\mathsf T}.
$$

Using element-wise multiplication $\odot$ and division $\oslash$, define

$$
\mathbf{q}(\lambda)
=\mathbf{c}(\lambda)
\odot\mathbf{d}_{65}
\oslash\overline{\mathbf{c}}.
$$

The Engine's normalized target for spectral signal $S$ is

$$
\mathbf{C}_{XYZ}
=\frac{1}{\Delta\lambda}
\int\limits_{\lambda_{\min}}^{\lambda_{\max}}
S(\lambda)\mathbf{q}(\lambda)\,d\lambda.
$$

This normalization maps a flat unit spectrum to the chosen D65 XYZ white:

$$
S(\lambda)=1
\quad\Longrightarrow\quad
\mathbf{C}_{XYZ}=\mathbf{d}_{65}.
$$

It is an Engine normalization convention, not an absolute radiometric scale.

### Monte Carlo Wavelength Estimator

For independent wavelength samples $\lambda_i\sim p(\lambda)$, the direct estimator is

$$
\widehat{\mathbf{C}}_{XYZ}
=\frac{1}{N}
\sum\limits_{i=1}^{N}
\frac{S(\lambda_i)\mathbf{q}(\lambda_i)}
{p(\lambda_i)\Delta\lambda}.
$$

It is unbiased for the normalized continuous observer integral when:

- $p(\lambda)>0$ wherever the integrand is nonzero;
- the transported spectral contribution is itself an unbiased estimate;
- the actual sampled wavelength is used in $\mathbf{q}$;
- all finite-depth, visibility, roulette, and model-support assumptions of the selected integrator hold.

The default wavelength density is uniform:

$$
p_u(\lambda)=\frac{1}{370\ \mathrm{nm}}.
$$

For this density, $p_u\Delta\lambda=1$. The code also implements piecewise-constant weighted samplers built from non-negative bin weights, including CIE-$Y$, RGB-importance, and composite weighting. They retain the $1/p$ compensation and can therefore preserve the estimator's expectation. They are runtime APIs only; the public render JSON currently provides no sampler selector, and the default handler always uses uniform sampling.

### Spectral Film Binning

Spectral renders initialize 64 film bins over $[380,750)$ nm, giving width

$$
\delta\lambda_{\mathrm{bin}}
=\frac{370}{64}\ \mathrm{nm}
=5.78125\ \mathrm{nm}.
$$

For bin $B_j$, the accumulated scalar is conceptually

$$
V_j
=\frac{1}{N}
\sum\limits_{i=1}^{N}
\mathbf{1}_{B_j}(\lambda_i)
\frac{S(\lambda_i)}{p(\lambda_i)\Delta\lambda}.
$$

Finalization evaluates the observer at each bin center $\lambda_j^*$:

$$
\widehat{\mathbf{C}}_{XYZ}^{\mathrm{bin}}
=\sum\limits_{j=1}^{64}
V_j\mathbf{q}(\lambda_j^*).
$$

This is unbiased for the bin-center-discretized observer model, not exactly for the continuous observer integral: replacing $\mathbf{q}(\lambda_i)$ with $\mathbf{q}(\lambda_j^*)$ introduces spectral discretization error. The raw bins are normalized Monte Carlo band contributions, not calibrated point samples of spectral radiance. They should not be plotted as an SPD without accounting for sampling density, normalization, and bin width.

### Authored Sampled Data Without a Wavelength Context

When a `sampled` parameter is evaluated in RGB mode, the Engine converts its authored pairs directly to linear sRGB. It computes an equal-weight discrete mean of normalized XYZ samples and then applies the XYZ-to-linear-sRGB matrix:

$$
\mathbf{C}_{XYZ}
=\frac{1}{n}
\sum\limits_{i=1}^{n}
S_i\mathbf{q}(\lambda_i).
$$

Negative output RGB channels are clamped to zero. This is not trapezoidal integration and does not weight irregular wavelength spacing. Authors who need a physically meaningful RGB conversion should provide sufficiently regular samples or perform controlled spectral integration before authoring the RGB value.

## Color-to-Spectrum Mapping

### Underdetermined Reconstruction

Given RGB value $\mathbf{r}$ and forward operator $\mathcal{M}$, spectral reconstruction asks for some $S$ satisfying

$$
\mathcal{M}[S]=\mathbf{r}.
$$

The null space of $\mathcal{M}$ is infinite-dimensional: if $N(\lambda)$ maps to zero, then $S+N$ has the same tristimulus color whenever it remains admissible. A reconstruction therefore requires additional assumptions such as smoothness, non-negativity, bounded reflectance, a basis, or a reference illuminant.

### Engine RGB Uplift

The Engine first converts a wavelength's approximate XYZ response to linear sRGB and clamps negative channels:

$$
\mathbf{b}(\lambda)
=\max\!\left(\mathbf{0},M_{XYZ\rightarrow sRGB}\mathbf{c}(\lambda)\right).
$$

Each basis channel is normalized by its mean over 2,048 wavelengths, producing $\mathbf{w}(\lambda)$. An RGB spectrum is then evaluated as

$$
S_{RGB}(\lambda)
=\max\!\left(0,
R\,w_R(\lambda)+G\,w_G(\lambda)+B\,w_B(\lambda)
\right).
$$

For reflectance uplift, the result is additionally capped by the largest RGB component:

$$
S_{\rho}(\lambda)
=\min\!\left(
\max(R,G,B),
S_{RGB}(\lambda)
\right).
$$

This is a renderer compatibility approximation. It does not guarantee exact round-trip color, spectral smoothness, energy preservation, illuminant invariance, or a physically measured reflectance. The cap does not enforce the physical upper bound of one when an authored RGB component exceeds one.

`RGBParameter.Eval` uses the reflectance uplift whenever a wavelength context is active, regardless of whether the parameter semantically represents reflectance, transmittance, eta, $k$, or emission. The shared schema is convenient, but the author remains responsible for choosing a physically appropriate spectral form for each field.

## Spectral Render Modes

| Public `spectrum_mode` | Path-tracing paths per camera sample | Wavelength selection | Transport representation | Film path |
| --- | ---: | --- | --- | --- |
| `rgb` | 1 | None | Three scene-linear sRGB coefficients | RGB is transformed directly into the selected film space |
| `hero_wavelength` | 1 | One uniform random wavelength | One monochromatic scalar path, with explicit RGB compatibility handling | Spectral bin $\rightarrow$ XYZ $\rightarrow$ film space |
| `sampled` | `wavelength_samples` | Stratified samples across the wavelength sampler's unit interval | Multiple independent monochromatic paths | All contributions enter spectral bins, then XYZ and the film space |

If sampled mode resolves to one or fewer wavelength samples, render configuration promotes it to four. At the lower-level handler API, a non-positive sampled count also defaults to four.

For $N_c$ camera samples and $m$ wavelength samples, pixel tracing normalizes all spectral contributions by

$$
\frac{1}{N_c m}.
$$

Hero mode uses $m=1$. In sampled pixel tracing, stratum $j$ uses

$$
u_j=\frac{j+\xi_j}{m},
\qquad
j=0,\ldots,m-1,
\qquad
\xi_j\sim U[0,1).
$$

The sampler maps $u_j$ to a wavelength and PDF. This reduces wavelength-stratification variance compared with fully independent uniform draws. Integrator-specific scheduling can differ: for example, the current splat-based light-tracing path samples one wavelength per light path rather than creating `wavelength_samples` paths.

### Hybrid RGB Compatibility

A spectral ray carries both `SpectralPower` and an optional `RGBCompatibility` product. A sampled material contribution multiplies the scalar spectral path directly. An RGB-only contribution encountered by a spectral ray is accumulated separately and evaluated at the ray wavelength before film accumulation. This prevents every RGB-only procedural component from immediately terminating a spectral render, but it remains an approximate RGB uplift path rather than measured spectral transport.

## Film Space and Image Output

Engine Film v3 contains only spectral planes. It has no RGB/XYZ channels, color
space, exposure, tone curve, gamma, or image encoder. Film merges require equal
dimensions, spectral-bin counts, and wavelength bounds.

Studio owns the observer and display boundary. It integrates Film bins against
the CIE 1931 approximation with one shared Y normalization for X, Y, and Z,
converts XYZ to the requested `color_space`, and finally encodes linear sRGB for
PNG output. `linear_srgb`, `xyz`, and `acescg` are the supported Studio color
spaces. Since XYZ and ACEScg are linear transforms, selecting either preserves
the same output tristimulus color rather than changing the physical Film.

### Output Transform

Before producing an 8-bit image, XYZ and ACEScg films are converted back to linear sRGB. For each channel $v$, non-finite or non-positive values become zero. The remaining operation order is:

$$
v_0=E v,
\qquad
v_1=T(v_0),
\qquad
v_2=\operatorname{clamp}(v_1,0,1),
\qquad
v_3=
\begin{cases}
v_2^{1/\gamma},&\gamma>0\ \text{and}\ \gamma\ne1,\\
v_2,&\text{otherwise}.
\end{cases}
$$

The available per-channel tone curves are

$$
T_{\mathrm{linear}}(v)=v,
$$

$$
T_{\mathrm{Reinhard}}(v)=\frac{v}{1+v},
$$

$$
T_{\mathrm{ACES\ fit}}(v)
=\frac{v(2.51v+0.03)}{v(2.43v+0.59)+0.14}.
$$

The final byte is $\operatorname{round}(255v_3)$. Tone mapping is channel-wise, so it can change hue and saturation. Exposure, tone mapping, and gamma are display operations; they must not be confused with a color space, a chromatic adaptation, or physical spectral transport.

The standalone Engine executable renders and saves the binary Film. When Studio drives Engine, the controller transfers each completed `Film` in memory; Studio persists it and calls its own image conversion directly, so image creation does not reread the file that was just written. Display controls are never applied to the binary Film itself.

### Film Binary Format v3

Film files are strict little-endian streams. The header is `RAYFILM\0`, version `uint32(3)`, sample count `int64`, rank `uint32`, `rank` dimensions as `uint64`, spectral-bin count `uint32`, and two `float64` spectral bounds. The payload contains only contiguous `float64` spectral planes. Implementations encode and decode planes in reusable 1 MiB blocks.

The decoder validates the exact version, rank, dimensions, spectral metadata, and payload byte count before allocation. Every older Film version is intentionally unsupported.

## Public Input Schema

### Render Configuration

```jsonc
{
  "render": {
    "spectrum_mode": "hero_wavelength", // hero_wavelength | sampled
    "wavelength_samples": 1,             // positive integer; sampled promotes <= 1 to 4
    "color_space": "linear_srgb",       // linear_srgb | xyz | acescg

    "exposure": 1,                       // positive output multiplier
    "tone_mapping": "linear",           // linear | reinhard | aces
    "gamma": 1                           // positive power-law exponent
  }
}
```

Engine defaults are `hero_wavelength` and one wavelength sample. `wavelength_samples` only controls sampled wavelength mode. `color_space`, exposure, tone mapping, and gamma are Studio-only output settings.

Equivalent command-line controls are:

```text
--spectrum-mode hero_wavelength|sampled
--wavelength-samples N
--color-space linear_srgb|xyz|acescg
--exposure E
--tone-mapping linear|reinhard|aces
--gamma G
```

### Shared Spectral Parameter

Legacy RGB is exactly three non-negative linear-sRGB values:

```jsonc
[0.8, 0.6, 0.2]
```

Tagged RGB:

```jsonc
{
  "type": "rgb",
  "value": [0.8, 0.6, 0.2],
  "space": "linear_srgb" // optional: linear_srgb | srgb | acescg
}
```

Constant spectrum:

```jsonc
{
  "type": "constant",
  "value": 0.8 // non-negative
}
```

Piecewise-linear sampled spectrum:

```jsonc
{
  "type": "sampled",
  "wavelengths_nm": [400, 500, 600, 700],
  "values": [0.1, 0.4, 0.8, 0.2],
  "interpolation": "linear"
}
```

The wavelength and value arrays must have equal lengths of at least two; wavelengths must be strictly increasing; values must be non-negative. `linear` is the only accepted interpolation name. Evaluation uses

$$
S(\lambda)
=(1-t)S_i+tS_{i+1},
\qquad
t=\frac{\lambda-\lambda_i}{\lambda_{i+1}-\lambda_i},
$$

inside an interval and clamps to the nearest endpoint value outside the authored range. Because the Engine can sample from 380 to 750 nm, endpoint extension should be chosen deliberately. Add explicit 380 nm and 750 nm samples when constant extrapolation is not intended.

Relative blackbody spectrum:

```jsonc
{
  "type": "blackbody",
  "temperature": 6500,
  "scale": 1
}
```

Temperature must be positive and is interpreted in kelvin. Scale is optional, defaults to one, and must be non-negative.

### Dispersive and Absorbing Medium Example

```jsonc
{
  "media": {
    "glass": {
      "type": "homogeneous",
      "ior": {
        "type": "cauchy",
        "a": 1.50,
        "b": 0.004,
        "c": 0
      },
      "sigma_a": {
        "type": "sampled",
        "wavelengths_nm": [380, 500, 620, 750],
        "values": [0.8, 0.2, 0.05, 0.02]
      }
    }
  }
}
```

This combines wavelength-dependent refraction with wavelength-dependent Beer--Lambert absorption. Participating-medium scattering is not implemented, so the Engine rejects `sigma_s` instead of retaining an ineffective physical parameter.

## Correctness, Invertibility, and Gamut

| Operation | Linear? | One-to-one in the implemented domain? | Main loss or approximation |
| --- | --- | --- | --- |
| Spectrum $\rightarrow$ XYZ | Yes | No | Metameric collapse from a function to three values |
| XYZ $\leftrightarrow$ linear sRGB matrix | Yes | Algebraically yes before clipping | Destination gamut and later negative clipping |
| XYZ $\leftrightarrow$ ACEScg matrix | Yes | Algebraically yes before clipping | No explicit D65/D60 adaptation in cross-space chains |
| sRGB $\rightarrow$ linear sRGB | Nonlinear | Yes over the intended scalar domain | None from the formula itself |
| Linear RGB $\rightarrow$ spectrum | Yes before max/cap | No | Heuristic basis uplift; not an inverse colorimetric reconstruction |
| Tone mapping | Nonlinear | Generally no | Highlight compression and channel-wise hue changes |
| Clamp/8-bit quantization | Nonlinear | No | Negative/out-of-range removal and quantization |

Three distinct failure modes should not be conflated:

- **Metamerism** is unavoidable information loss in spectral-to-tristimulus mapping.
- **Out-of-gamut conversion** occurs when a valid XYZ color needs negative coefficients in a chosen RGB basis.
- **Display mapping loss** is introduced intentionally by tone mapping, clipping, gamma, and quantization.

## Authoring Guidance

- Use `rgb` mode for speed when wavelength does not affect path direction or attenuation.
- Use `hero_wavelength` for spectral effects at the lowest per-camera-sample path count. Expect higher wavelength noise.
- Use `sampled` with several wavelength samples when dispersion, narrow spectra, or strongly varying absorption needs lower chromatic variance.
- Author measured or deliberately sampled spectra for dispersion-sensitive work. RGB uplift is a compatibility mechanism, not a material-measurement substitute.
- Use `srgb` only for values read from an sRGB-authored source. Use `linear_srgb` for already linear numerical coefficients.
- Treat authored `acescg` material parameters cautiously until a real ACEScg-to-render-space input transform and chromatic-adaptation policy exist.
- Keep passive reflectance and transmittance at or below one even though the parser enforces only non-negativity.
- Include the Engine interval endpoints in sampled data when endpoint clamping would otherwise create unintended tails.
- Keep film data linear for merging and analysis. Apply exposure, tone mapping, gamma, and quantization only when creating a display image.
- Compare film values only in the same working space. Convert to a common linear space before numerical error metrics.

### Current Model Boundaries

The Engine currently does not provide:

- absolute radiometric units or camera exposure calibration;
- a tabulated standard observer, selectable observer, or camera sensor response;
- ultraviolet or infrared transport outside 380--750 nm;
- wavelength-changing fluorescence, phosphorescence, or Raman scattering;
- polarization or Stokes/Mueller transport;
- participating-medium scattering (`sigma_s` is rejected);
- a physically constrained spectral reconstruction from RGB;
- a full ICC/OCIO/ACES color-management stack;
- explicit chromatic adaptation between D65 and D60 conventions;
- a public JSON selector for weighted wavelength samplers;
- exact sRGB output encoding;
- a calibrated spectral-file interchange format.

These are boundaries, not additional hidden categories. Within them, the renderer supports wavelength-dependent surface parameters, dispersion, absorption, blackbody emission, Monte Carlo wavelength sampling, XYZ observer conversion, three film spaces, and deterministic display transforms.

| Concern | Engine source |
| --- | --- |
| Wavelength interval, CIE approximation, D65 normalization, spectral-to-XYZ conversion | `engine/model/optics/wavelength.go` |
| RGB/XYZ/ACEScg matrices and spectral-ray conversion | `engine/model/optics/color.go` |
| Color triples | `engine/model/optics/color3.go` |
| Spectrum kinds, arithmetic, and RGB uplift | `engine/model/optics/spectrum.go` |
| Spectral parameter interface and sRGB decoding | `engine/model/optics/spectral_parameter.go` |
| RGB, sampled, constant, and blackbody evaluation | `engine/model/optics/spectrum_parameter/` |
| Uniform and weighted wavelength samplers | `engine/model/optics/wavelength_sampler.go` |
| Spectral ray state | `engine/model/optics/ray.go` |
| Spectral/RGB throughput compatibility | `engine/ray_tracing/throughput.go` |
| Pixel wavelength scheduling and normalization | `engine/ray_tracing/trace_pixel.go` |
| Film bins, film-space transforms, tone mapping, and gamma | `engine/model/camera/film.go` |
| Spectral film preparation and finalization | `engine/ray_tracing/trace_scene.go`, `engine/ray_tracing/render_session.go` |
| Public render schema and defaults | `engine/controller/parser/schema.go`, `engine/controller/render_context.go` |
| Public spectral-parameter parser | `engine/controller/factory/materials.go` |
| Cauchy dispersion and homogeneous media | `engine/model/material/medium/` |
