# Integrator

## Real Integrator Category Tables

| Input value | Actual estimator | Runtime fallback or hard failure |
| --- | --- | --- |
| Path Tracing | Camera-path, BSDF-sampled path tracing | None |
| Light Tracing | Light-subpath tracing with a $t=1$ camera projection at eligible vertices | Fails if the camera does not implement `camera.ProjectiveCamera` |
| Bidirectional Path Tracing | Continuous bidirectional connections plus a separate delta-caustic camera-splat family | Falls back to the path estimator for unsupported scenes; the driver remains the BDPT splat driver |

Unknown values are rejected by `ParseIntegratorKind`.

### Runtime Types and Responsibilities

| Runtime type/interface | Role | Selected directly by input? |
| --- | --- | --- |
| `IntegratorKind` | Serialized canonical integrator name | Yes, through parsing |
| `SceneIntegrator` | Scene-level `Render(RenderContext)` contract | No |
| `configuredSceneIntegrator` | Common owner of handler, driver, session creation, and finalization | No; used for every kind |
| `RenderDriver` | Scheduling, effective sample count, and film-write concurrency contract | No |
| `pixelDriver` | Tiled per-pixel scheduling | Indirectly by `path` |
| `splatDriver` | Global work scheduling and normalized arbitrary-pixel splats | Indirectly by `bdpt` and `light_tracing` |
| `pixelKernel` | Per-pixel RGB/spectral sampling contract | No |
| `pathTracingKernel` | Adapter from the pixel driver to `TraceRGB` and `TraceRay` | Indirectly by `path` |
| `splatKernel` | `Prepare`, `WorkCount`, and `TraceSample` contract | No |
| `bdptKernel` | BDPT preparation, wavelength/pixel work mapping, fallback, and delta splats | Indirectly by `bdpt` |
| `lightTracingKernel` | Projective-camera validation, light distribution, global light paths, and splats | Indirectly by `light_tracing` |

Drivers and kernels are implementation components, not additional user-visible integrator categories.

### Capability Matrix

| Capability | `path` | Effective `bdpt` | `light_tracing` |
| --- | --- | --- | --- |
| Camera-path construction | Yes | Yes | No |
| Light-subpath construction | No | Yes | Yes |
| Explicit vertex connection | No | Light vertex to camera-subpath vertex | Light vertex to projective camera |
| Multiple importance sampling | No | Power heuristic for the continuous strategy family | No |
| Delta surface sampling | Yes | Yes in subpaths; excluded from continuous MIS; selected delta caustics use separate $t=1$ splats | Yes in the light walk, but delta vertices themselves cannot be projected |
| Emissive surface without `SurfaceSampler` | Can be hit by the camera path | Causes BDPT fallback if there is no other sampleable area light | Not selected as a light |
| Homogeneous absorption | Yes | Yes | Yes |
| Participating-medium scattering | No | No | No |
| Euclidean geometry | Yes | Yes | Practically required by the current projective/visibility math, but not explicitly gated |
| Klein geometry | Yes | Falls back to path | No Engine projective Klein camera exists |
| Spherical geometry | Yes, including wrap handling | Falls back to path | No Engine projective spherical camera exists |
| Non-reciprocal surface | Yes | Falls back to path | Not explicitly rejected |
| Arbitrary-pixel film splats | No | Yes for delta-caustic $t=1$ paths | Yes for every eligible light vertex |
| Unbiasedness scope | Conditional for the supported depth/arc-truncated camera-path model | Conditional for the enabled reciprocal continuous-MIS and separate supported delta-caustic families; fallback inherits path behavior | Conditional for the finite-light, finite-depth, Euclidean projective $t=1$ family |

## Model Background

The former shared-contract description was not a complete model background: it defined local throughput, absorption, and roulette, but did not first state the rendering equation, camera measurement, path-space measure, PDF conversions, delta events, spectral estimator, or the conditions required for unbiased Monte Carlo estimation. Those foundations are made explicit below.

### Surface Rendering Equation

For a surface point $x$ and outgoing direction $\omega_o$, the surface-only transport model is

$$
L_o(x,\omega_o,\lambda)
=
L_e(x,\omega_o,\lambda)
+
\int_{\mathcal{H}(x)}
f_s(x,\omega_i,\omega_o,\lambda)
L_i(x,\omega_i,\lambda)
|n_x\cdot\omega_i|\,d\omega_i.
$$

$\mathcal{H}(x)$ is the valid directional domain in the local shading frame. The engine evaluates finite emissive surfaces, BSDF scattering, visibility, homogeneous absorption, and medium-boundary IOR changes. It has no environment-emission term and no volume-scattering source term.

For a pixel $p$, an idealized camera measurement is

$$
I_p
=
\int_{\mathcal{C}_p}
W_p(r)\,L(r)\,d\mu_{\mathrm{camera}}(r),
$$

where $W_p$ contains the camera/raster filter and $\mathcal{C}_p$ is the camera-ray domain for that pixel. Path tracing samples this domain from the camera. Light tracing and the $t=1$ BDPT family instead use `ProjectPoint`, whose returned Jacobian maps a scene-point contribution to the box-filtered raster measurement.

### Path-Space Integral

A complete finite surface path is $\bar{x}=(x_0,\ldots,x_k)$, with a light endpoint, zero or more scattering vertices, and a camera endpoint. The measurement can be decomposed by path length:

$$
I_p
=
\sum\limits_{k=1}^{\infty}
\int_{\Omega_k}
F_p(\bar{x})\,d\mu_k(\bar{x}).
$$

For a non-delta Euclidean surface path, the contribution contains emission, BSDFs, segment transmittance, visibility, and geometry factors:

$$
F_p(\bar{x})
=
W_p(\bar{x})L_e(x_0)
\prod\limits_{i=1}^{k-1}
\left[f_i\,T_i\,G(x_{i-1},x_i)\right].
$$

Delta reflection and transmission are singular measures and cannot be treated as ordinary area- or solid-angle-density factors. The implementation therefore samples delta events discretely and excludes sampled-delta path views from continuous BDPT MIS.

### Monte Carlo Estimation and Unbiasedness Conditions

For samples $X_n\sim p$ over the same measure as $F$, the standard estimator is

$$
\widehat{I}_N
=
\frac{1}{N}
\sum\limits_{n=1}^{N}
\frac{F(X_n)}{p(X_n)}.
$$

It is unbiased when:

1. $p(x)>0$ everywhere $F(x)\ne0$ in the claimed path family;
2. every area, direction, light-selection, wavelength, roulette, and camera density is included with respect to the correct measure;
3. visibility, BSDF, emission, transmittance, and Jacobian evaluations match the sampled path;
4. rejected samples represent zero contribution rather than silently removing non-zero support;
5. MIS weights form a partition of unity over the strategies used for the same contribution;
6. termination is either part of the declared finite model or compensated probabilistically.

Under these conditions $\mathbb{E}[\widehat{I}_N]=I$. Unbiasedness does not imply low variance. Conversely, deterministic maximum depth, maximum arc length, omitted emitters, unsupported path measures, or missing transport phenomena make an estimator biased relative to the unrestricted physical rendering equation even when it is unbiased for the engine's narrower implemented model.

### Measure Conversion and Geometry

For two Euclidean surface vertices $x$ and $y$, a directional density at $x$ converts to area density at $y$ through

$$
p_A(y)
=
p_\omega(\omega_{xy})
\frac{|n_y\cdot\omega_{yx}|}{\|y-x\|^2}.
$$

The corresponding connection geometry term is

$$
G(x,y)
=
V(x,y)
\frac{|n_x\cdot\omega_{xy}|\,|n_y\cdot\omega_{yx}|}
{\|y-x\|^2},
$$

where $V(x,y)$ is binary visibility. These Euclidean area-measure formulas are used by effective BDPT and projective light tracing. Regular path tracing instead follows the active geometry's ray mapping and converts intersection parameters to geometry-specific arc length.

### Core Quantities and Engine Mapping

The implementations use the following quantities:

| Symbol | Meaning in the Engine |
| --- | --- |
| $\beta$ | Path throughput before evaluating the current connection/emission factor |
| $f(x,\omega_i,\omega_o)$ | Surface BSDF spectrum returned by `Surface.Eval` or `Surface.Sample` |
| $p_\omega(\omega_i)$ | Directional sampling PDF in solid angle |
| $p_A(x)$ | Density with respect to surface area |
| $T(d)$ | Homogeneous-medium segment transmittance |
| $G(x,y)$ | Surface-to-surface geometry term |
| $L_e(x,\omega)$ | Emitted radiance from an emissive material |
| $J_{\mathrm{camera}}(x)$ | `ProjectPoint` Jacobian from a scene point to the box-filtered film |

### BSDF Throughput Update

All three path families use the same basic sampled-surface update:

$$
\beta_{i+1}
=
\beta_i f(x_i,\omega_i,\omega_o)
\frac{|\cos\theta_i|}{p_\omega(\omega_i)}.
$$

The sample is rejected when its PDF is non-positive, its spectrum is non-finite, or its spectrum contains a negative component.

For a transmission event, the ray's medium stack and current index of refraction are updated after the BSDF weight is applied.

### Segment Transmittance

For homogeneous absorption coefficient $\sigma_a$ and traveled distance $d$, every traced or explicitly connected segment uses Beer-Lambert attenuation:

$$
T(d,\lambda)=\exp\!\left[-\sigma_a(\lambda)d\right].
$$

RGB mode evaluates this independently per RGB coefficient. Spectral mode evaluates the coefficient at the active wavelength. Zero absorption returns an identity transmittance.

The Engine does not sample a free-flight distance or phase function. A configured scattering coefficient is therefore not a volumetric path event in these integrators.

### Spectral Monte Carlo Estimation

In wavelength modes, a sampled wavelength $\lambda\sim p_\lambda$ contributes

$$
\widehat{C}(\lambda)
=
\frac{C(\lambda)}{p_\lambda(\lambda)}.
$$

Hero-wavelength path tracing and both splat kernels use one active wavelength per principal path. Sampled-mode path tracing and BDPT additionally stratify the wavelength domain across `wavelength_samples`. Dividing by $p_\lambda$ makes the wavelength estimator unbiased for the engine's sampled spectral integral, provided the wavelength PDF has support wherever the spectral contribution is non-zero. Film bins are converted to the configured color space only after accumulation.

RGB mode is not a wavelength Monte Carlo estimator. It evaluates the engine's RGB approximations directly, including RGB uplift behavior inside spectral parameters when a model explicitly requests wavelength evaluation.

### Russian Roulette

The internal handler default is to begin roulette at depth 3.

For regular path tracing, survival is based on the current ray throughput and is clamped to $[0.05,1]$ when throughput is positive. Spectral paths use spectral power, multiplied by the maximum RGB-compatibility channel when that compatibility path is active.

For BDPT camera/light subpaths:

$$
q=
\begin{cases}
1, & \text{before the roulette depth},\\
\min\!\left(0.95,\max\!\left(0.05,\max_c \beta_c\right)\right),
& \text{at and after the roulette depth}.
\end{cases}
$$

A surviving path divides throughput by $q$. BDPT also folds $q$ into the pending forward density used for subsequent area-PDF calculations.

For a continuation contribution $Y$, roulette produces

$$
\widehat{Y}
=
\begin{cases}
Y/q, & \text{with probability }q,\\
0, & \text{with probability }1-q,
\end{cases}
$$

and therefore $\mathbb{E}[\widehat{Y}]=Y$. Roulette is unbiased when $q>0$ for every non-zero continuation and the division by $q$ is applied exactly once. It changes variance and work, not the expectation.

### Model Completeness Boundary

The background above is complete for interpreting the estimators implemented in this file: it covers the surface rendering equation, camera measurement, path-space decomposition, continuous and delta measures, area/solid-angle conversion, BSDF throughput, homogeneous attenuation, spectral sampling, MIS conditions, and roulette.

It is not a model of every physically possible transport process. The current engine omits environment lighting, participating-medium scattering, phase functions, free-flight sampling, polarization, fluorescence, diffraction, and transient transport. Its hard `MaxRayLevel`, optional geometry arc budget, finite supported light set, and integrator-specific strategy restrictions must therefore be included when stating any unbiasedness claim.

## Path Tracing

### Actual Category

This is a unidirectional camera-path tracer. It samples only the BSDF at each surface. It has no explicit next-event estimation, no light selection, and no MIS.

A path contributes only when it directly reaches an emissive surface. A miss contributes zero; there is no environment-light evaluation.

### Estimator

For a camera path with sampled surface vertices $x_1,\ldots,x_k$ ending on an emitter, the implemented contribution has the form:

$$
\widehat{L}
=
\left[
\prod\limits_{i=1}^{k-1}
T_i f_i\frac{|\cos\theta_i|}{p_i}
\right]
T_k L_e(x_k,\omega_{o,k}).
$$

Russian-roulette compensation is included in the product for surviving paths. There is no direct-light term at non-emissive vertices.

If a material contains both emission and a surface, `traceEmission` evaluates the emission and returns immediately. Its surface is not sampled by this integrator at that hit.

### Unbiasedness Analysis

For the engine's finite surface-transport model, one camera-path sample has the standard importance-sampling form $F(\bar{x})/p(\bar{x})$. The BSDF factors, directional PDFs, segment transmittance, wavelength PDF, and roulette survival probabilities are compensated in throughput. Therefore, for every path family with non-zero sampling support,

$$
\mathbb{E}[\widehat{L}_{\mathrm{path}}]
=L_{\mathrm{implemented}}.
$$

The absence of next-event estimation or MIS does not by itself introduce bias. It can produce extremely high variance because an emissive surface must be reached by the BSDF random walk.

This conclusion has explicit boundaries:

- the deterministic `MaxRayLevel = 64` truncates longer paths and is biased relative to the infinite-bounce rendering equation;
- `MaxArc` can additionally truncate curved-geometry paths;
- a black miss, no environment term, no participating-medium scattering, and emission-first material behavior define a narrower engine model rather than an estimator of those omitted phenomena;
- numerical intersection failures, unsupported Shape/geometry combinations, or a BSDF PDF with missing support can remove non-zero contributions.

**Conclusion:** path tracing is conditionally unbiased for the engine's supported, depth/arc-truncated surface model. It is not strictly unbiased for the unrestricted infinite-path physical rendering equation.

### Per-Sample Flow

1. The camera generates a ray for the current film coordinate.
2. RGB mode disables spectral sampling. Spectral modes attach one sampled wavelength and its PDF.
3. Before a bounce, the path checks `MaxRayLevel` and Russian roulette.
4. The Engine intersects the ray in the active geometry.
5. On a spherical miss at the end of a half-great-circle, it may wrap at the antipode and continue while the arc budget allows.
6. The hit parameter is converted to geodesic arc length.
7. Beer-Lambert absorption for the current medium is applied to the segment.
8. Total traveled arc is advanced and checked against `MaxArc`.
9. The hit creates a geometry-aware shading frame, local outgoing direction, medium transition context, hit point, UV, and object AABB context.
10. If the surface emits, emission multiplies the current throughput and the path ends.
11. Otherwise the surface BSDF is sampled and throughput is multiplied by $f|\cos\theta|/p_\omega$.
12. Transmission updates the medium stack/IOR.
13. The sampled local direction is transformed to world space, projected into the geometry tangent space, normalized with the geometry metric, and traced recursively.

### Geometry Behavior

Path tracing is the only current integrator with complete Engine routing for all three geometry kinds:

- Euclidean rays use affine embedded-space intersections.
- Klein rays use the geometry's embedded-ray mapping and geodesic arc conversion.
- Spherical rays use explicit great-circle intersection and antipodal wrapping.

Actual shape support still depends on the shape's geometry-specific intersection implementation. See `geometry-shape.md` for the Engine-only shape matrix.

### Spectral and Sample Normalization

| Spectrum mode | Work per active pixel | Wavelength sampling | Pixel normalization | Recorded `Film.Samples` |
| --- | --- | --- | --- | --- |
| `rgb` | `samples` camera paths | None | Arithmetic mean of camera paths | `samples` |
| `hero_wavelength` | `samples` camera paths | One uniform random wavelength per path | Each value is divided by wavelength PDF, then all samples by `samples` | `samples` |
| `sampled` | $\mathtt{samples}\times\mathtt{wavelength\_samples}$ paths | Stratified wavelength batch per camera sample | Each value is divided by wavelength PDF, then all wavelength paths by their total count | $\mathtt{samples}\times\mathtt{wavelength\_samples}$ |

Spectral contributions are accumulated into 64 default film bins and converted to the film color space during finalization.

### Strengths and Structural Limits

- It supports curved geometry and non-reciprocal surfaces that BDPT rejects.
- It can see any intersectable emissive surface, including an emitter whose shape cannot be area-sampled.
- It is inefficient for small lights and caustics because it never explicitly samples a light or connects vertices.
- It stops at the first emissive hit and does not continue through a surface component on the same material.

## Light Tracing

### Actual Category and Camera Requirement

This is a light-subpath tracer with a projective-camera $t=1$ estimator. It starts from a finite sampleable emissive surface, walks through the scene, and tries to project every eligible light-path vertex onto the film.

`Prepare` requires the selected camera to implement:

```go
type ProjectiveCamera interface {
    Camera
    ProjectPoint(point *mat.VecDense) (FilmProjection, bool)
}
```

In the current Engine, `Camera3D` is the only implementation of `ProjectPoint`. Other camera types cause a render error rather than a fallback.

### Area-Light Distribution

Only objects satisfying all of these conditions enter the distribution:

- non-nil object and material;
- material has emission;
- non-nil shape;
- shape implements `shape.SurfaceSampler`;
- finite positive `SurfaceArea()`.

For light $j$, the preparation weight is:

$$
\begin{aligned}
P_j
&=\max_c\!\left[
\operatorname{Emit}_j(\text{RGB context},(0,0,1))_c
\right],\\
w_j&=A_jP_j,\\
p_{\mathrm{select}}(j)&=\frac{w_j}{\sum\limits_k w_k}.
\end{aligned}
$$

If $P_j$ is non-positive or non-finite, the code substitutes $P_j=1$. The sampled endpoint area density is:

$$
p_A(x_0)=p_{\mathrm{select}}(j)\,p_{\mathrm{surface}}(x_0\mid j).
$$

This is a power proxy, not an integrated spectral power calculation.

### Light Endpoint and Direction Sampling

The endpoint is initialized with:

$$
\beta_0=\frac{1}{p_A(x_0)}.
$$

The code chooses either orientation of the geometric normal with probability $1/2$, then cosine-samples a hemisphere. The combined directional density is:

$$
p_\omega(\omega_0)=\frac{|\cos\theta_0|}{2\pi}.
$$

The launched light-path throughput is:

$$
\beta_1
=
L_e(x_0,\omega_0)
\frac{|\cos\theta_0|}
{p_A(x_0)p_\omega(\omega_0)}.
$$

Subsequent vertices apply segment transmittance, the shared BSDF throughput update, medium transitions, and Russian roulette.

### Camera Projection Estimator

For an eligible surface vertex $x$ projected to a camera pixel:

$$
C
=
T(x\rightarrow\mathrm{camera})\,
\beta_x f_x(\omega_{\mathrm{camera}},\omega_{o,x})
J_{\mathrm{camera}}(x)
|n_x\cdot\omega_{\mathrm{camera}}|.
$$

For the sampled light endpoint, the BSDF factor is replaced by its emitted radiance:

$$
C_{\mathrm{endpoint}}
=
T(x_0\rightarrow\mathrm{camera})\,
\beta_0 L_e(x_0,\omega_{\mathrm{camera}})
J_{\mathrm{camera}}(x_0)
|n_0\cdot\omega_{\mathrm{camera}}|.
$$

`ProjectPoint` supplies the target pixel, direction to the camera, distance, and Jacobian. The projection is rejected when it is outside the film, behind the camera, occluded, has a non-positive surface cosine, or produces an invalid spectrum.

A non-endpoint vertex is also rejected when:

- it has no surface;
- its sampled event is delta; or
- the complete surface advertises any delta flag.

Thus delta events can transport a light path to a later continuous surface, but a delta vertex is not directly evaluated as a continuous camera connection.

### Unbiasedness Analysis

Let $X_n$ be one sampled light subpath and let $\mathcal{V}(X_n)$ be its eligible projected vertices, including the sampled light endpoint. The implemented splat estimator can be written as

$$
\widehat{I}_{\mathrm{LT}}
=
\frac{1}{N}
\sum\limits_{n=1}^{N}
\sum\limits_{v\in\mathcal{V}(X_n)}
C(v;X_n),
$$

where each $C$ already contains the inverse light-selection, endpoint-area, emission-direction, BSDF, wavelength, and roulette densities carried by $\beta$, plus the camera projection Jacobian. Multiple splats from one path are not independent, but independence within a path is unnecessary for unbiasedness of their sum.

The preparation weight $w_j=A_jP_j$ affects variance, not expectation, because the selected endpoint is divided by its actual $p_A$. Global division by $N=SP$ then averages the launched paths. Subject to an accurate `ProjectPoint` Jacobian, visibility test, and positive sampling support, the estimator is unbiased for the implemented projective-camera $t=1$ path family.

It is not a complete unbiased estimator of the unrestricted camera measurement:

- emissive objects without a positive `SurfaceSampler` area have zero launch probability;
- delta vertices cannot be projected directly as continuous camera connections;
- only the current Euclidean 3D projective-camera interpretation has an established Jacobian;
- environment and participating-medium paths are absent;
- `MaxRayLevel` deterministically truncates long light paths.

The lack of MIS is not itself bias within the supported family; it affects variance. **Conclusion:** light tracing is conditionally unbiased for its sampled finite-light, finite-depth, Euclidean projective $t=1$ family, but not for all paths of the full rendering equation.

### Work Scheduling and Normalization

Let $P$ be the number of active pixels and $S$ the configured sample count:

$$
N_{\mathrm{light\ paths}}=SP,
\qquad
C_{\mathrm{film}}=\frac{C}{SP}.
$$

Every global path may produce zero, one, or many splats and may write any active pixel. Per-pixel locks protect concurrent additions.

Pixel windows restrict accepted splats and also reduce $P$, so the number of launched paths scales with the active region. If no sampleable lights, no positive total light weight, or no active pixels exist, the kernel performs zero work and finalizes a blank film.

In RGB mode, splat values are converted from linear sRGB to the selected film color space. In either spectral mode, each light path samples one random wavelength and stores $L(\lambda)/p(\lambda)$. `wavelength_samples` does not create extra light-tracing paths and sampled mode is not wavelength-stratified here.

### Current Limits

- Only finite emissive shapes implementing `SurfaceSampler` can launch paths.
- There is no MIS with camera-path strategies.
- There is no environment-light or infinite-light sampling.
- There is no participating-medium scattering.
- The kernel has no BDPT-style geometry or reciprocity capability gate. Its projection, visibility segment, squared-distance term, and current projective camera are Euclidean/3D mechanisms; using it outside that setting is not established by the code.

## Bidirectional Path Tracing

### Actual Category

The implemented BDPT combines three behaviors:

1. A camera subpath and a light subpath are built independently.
2. Continuous $t\ge 2$ connection strategies are combined with a power-heuristic MIS weight.
3. A separate projective-camera $t=1$ estimator handles a selected delta-caustic path family.

It is not a fully symmetric all-strategy BDPT. The camera endpoint is fixed by the selected pixel, continuous $s=0$ and $t=1$ strategies are outside the MIS family, and paths containing sampled delta events are rejected from continuous MIS.

### Preparation and Fallback

Before sampling, BDPT prepares its area-light distribution and records a fallback reason. Effective BDPT requires:

- `geometry.Get(sceneGeometry).Kind()` equals `EuclideanKind`;
- no material metadata marked `NonReciprocal`;
- no surface whose delta flags contain `NonReciprocal`;
- at least one sampleable finite area light with positive total weight.

Otherwise every work item calls the regular path estimator and writes the result through the BDPT splat driver. A message of this form is written to standard error once during preparation:

```text
BDPT fallback: requested=bdpt effective=path reason=...
```

The configured kind, driver, work count, film synchronization, and `Film.Samples` semantics remain BDPT even during fallback.

The fallback reasons are currently:

| Condition | Reason text |
| --- | --- |
| Geometry kind is not Euclidean | `BDPT currently requires three-dimensional Euclidean geometry` |
| Any surface is non-reciprocal | `scene contains a non-reciprocal surface` |
| No usable prepared light | `scene has no sampleable finite area light` |

The first gate checks only the geometry kind. The reason text says three-dimensional, while the explicit three-component assumptions appear later in MIS direction/frame helpers. Non-3D Euclidean input is therefore not rejected by this gate, but its continuous BDPT densities cannot be evaluated normally.

### Camera Subpath

The camera generates a ray for the selected pixel. The subpath begins with $\beta=1$ and uses radiance transport mode.

At each hit:

1. Apply segment transmittance.
2. Convert the pending solid-angle density to area measure at the destination:

   $$
   p_A(x_i)
   =p_\omega(\omega_{i-1})
   \frac{|n_i\cdot(-\omega_{i-1})|}
   {\|x_i-x_{i-1}\|^2}.
   $$

3. Store position, geometric normal, frame, local outgoing direction, context, object, throughput, area PDF, and cloned medium stack.
4. Add directly hit emission only when the hit is the first camera vertex, the previous event was delta, or the emitter is not sampleable as an area light.
5. Sample the BSDF and update throughput.
6. Store directional PDF, delta status, and roulette survival probability.
7. Apply medium transmission and continue.

The selective emitted term avoids adding the normal non-delta hit of a sampleable area light on top of the explicit continuous connection family. It preserves camera-visible, specular-visible, and non-sampleable emission.

### Light Subpath

The light endpoint distribution and initial direction estimator are the same as light tracing. The remainder of the subpath uses importance transport mode.

Each stored light vertex contains the same data as a camera vertex. Its pending directional PDF, including roulette survival, is converted into a forward area PDF at the next vertex.

Homogeneous absorption is applied before recording each surface vertex. Delta surfaces are allowed in subpath sampling even though they are excluded from continuous connection MIS.

### Continuous Vertex Connection

For a light vertex $x_l$ and a camera vertex $x_c$, the Engine first checks the hard depth condition:

$$
l_i+c_i+1\le \mathtt{MaxRayLevel}.
$$

It then requires an unobstructed segment, a valid camera-vertex surface BSDF, valid endpoint cosines, and a valid light factor.

The geometry term is:

$$
G(x_l,x_c)
=
\frac{|n_l\cdot\omega_{lc}|\,|n_c\cdot\omega_{cl}|}
{\|x_l-x_c\|^2}.
$$

For a non-endpoint light vertex:

$$
C_{s,t}
=
T_{lc}\,\beta_l f_l\,G(x_l,x_c)\,f_c\beta_c.
$$

For the sampled light endpoint:

$$
C_{1,t}
=
T_{lc}\,\beta_l L_e(x_l,\omega_{lc})
G(x_l,x_c)\,f_c\beta_c.
$$

Here the light-endpoint $\beta_l$ contains $1/p_A$. The connection transmittance is evaluated using the light vertex's current medium for the complete shadow segment.

### Continuous MIS

For one complete non-delta path of length $n$, enabled split strategies satisfy $1\le s\le n-1$. The fixed camera endpoint is not represented as a stored BDPT vertex, so $s=0$ and the $t=1$ camera-splat strategy are excluded.

Each strategy density begins with the sampled light endpoint area density and multiplies area-measure edge densities on both sides of the split:

$$
p_A(x_j\rightarrow x_k)
=
p_\omega(\omega_{jk})
\frac{|n_k\cdot\omega_{jk}|}{\|x_j-x_k\|^2}.
$$

The light side evaluates PDFs in importance mode; the camera side evaluates them in radiance mode. The current strategy is weighted by the power heuristic:

$$
w_s=\frac{p_s^2}{\sum\limits_k p_k^2}.
$$

The implementation evaluates adjacent-strategy density ratios in log space, clamps exponential overflow/underflow, and returns zero for invalid densities.

If any selected light or camera vertex reports `SampledDelta`, the continuous MIS weight is zero. Reverse discrete densities and refractive eta corrections are not present, so discrete and continuous measures are intentionally not mixed.

### Separate Delta-Caustic $t=1$ Family

After the continuous local estimator is computed, the kernel scans the light subpath. Once an earlier vertex sampled a delta event, each later vertex is considered for direct projective-camera splatting through the same `projectLightVertex` logic used by light tracing.

This targets paths such as:

```text
light -> specular chain -> continuous receiving surface -> camera
```

These splats are not included in the continuous MIS denominator. They are emitted only when the selected camera implements `ProjectiveCamera`; otherwise continuous BDPT still runs without this family.

### Unbiasedness Analysis

For a continuous path contribution $F(\bar{x})$ sampled by enabled strategies $s\in\mathcal{S}$, the multiple-strategy estimator is unbiased when every strategy uses the correct density and the weights satisfy

$$
\sum\limits_{s\in\mathcal{S}}w_s(\bar{x})=1.
$$

The power heuristic used by the engine has this partition property for finite positive continuous densities:

$$
w_s(\bar{x})
=
\frac{p_s(\bar{x})^2}
{\sum\limits_{k\in\mathcal{S}}p_k(\bar{x})^2}.
$$

The subpath throughputs include endpoint, BSDF, wavelength, transmittance, and roulette factors, while adjacent-strategy ratios convert directional PDFs to area measure. Under those conditions, the continuous MIS estimate is unbiased for the reciprocal, non-delta strategy family actually enabled by the implementation.

The separate delta-caustic $t=1$ splats use a light-tracing-style estimator and are excluded from the continuous denominator. This avoids mixing discrete delta measures with continuous PDFs. They are conditionally unbiased for that selected projective path family when their light-path PDFs and camera Jacobian are correct.

The complete result still has important qualifications:

- deterministic `MaxRayLevel` truncation is biased relative to infinite path space;
- continuous MIS omits $s=0$, ordinary $t=1$, camera-endpoint density, lens strategies, and every sampled-delta path view;
- direct camera emission and the separate delta family recover selected omitted contributions, but the implementation does not constitute a general completeness proof for every physical path class;
- unsupported geometry, non-reciprocal surfaces, or missing sampleable lights switch to the path estimator, whose own conditional-unbiasedness statement applies;
- the continuous density implementation is established only for reciprocal 3D Euclidean surface transport.

**Conclusion:** effective BDPT is conditionally unbiased for the union of its implemented continuous reciprocal MIS family and separate supported delta-caustic family, subject to correct disjoint accounting. It is not a strictly unbiased estimator of unrestricted infinite-depth transport. Fallback mode is the path estimator under BDPT scheduling and normalization.

### Work Scheduling and Spectral Modes

Let $P$ be the number of active pixels, $S$ the configured sample count, and $W$ equal `wavelength_samples` only in sampled mode and one otherwise:

$$
N_{\mathrm{work}}=SPW.
$$

Work indices cycle over active pixels. Sampled spectral mode also cycles over wavelength strata. Each work item builds one camera/light pair or one fallback camera path.

The local camera-pixel contribution is multiplied by $P$ before the splat driver divides every contribution by total work. This cancels uniform pixel selection and produces per-pixel averaging:

$$
C_{\mathrm{stored}}
=
C_{\mathrm{local}}\frac{P}{SPW}
=
\frac{C_{\mathrm{local}}}{SW}.
$$

Delta-caustic splats are global contributions and are divided directly by total work.

The splat driver records `Film.Samples` as $S$, even when sampled mode performs $SW$ work items per active pixel. This differs from the path driver's effective sample count.

### Current Limits

- Effective BDPT is limited to Euclidean geometry and reciprocal surfaces.
- Light discovery supports only finite emissive `SurfaceSampler` shapes.
- Continuous MIS excludes every path view containing a sampled delta event.
- The separate $t=1$ family covers delta-caustic projections, not all ordinary $t=1$ strategies.
- No camera endpoint density, lens sampling, $s=0$ continuous strategy, environment-light strategy, or participating-medium scattering is implemented.
- The continuous MIS helpers explicitly require three-component points and frames even though the capability gate itself checks only Euclidean geometry kind.

## Public Input, Execution, and Film Semantics

### Canonical Scene JSON

Integrator selection is part of `render`, or of each entry in `renders` for multiple render jobs:

```jsonc
{
  "render": {
    "integrator": "path",
    "samples": 20,
    "thread_num": 8,
    "camera_index": 0,
    "widths": [400, 400],
    "spectrum_mode": "hero_wavelength",
    "wavelength_samples": 1,
    "pixel_windows": [
      { "min": [0, 0], "max": [400, 400] }
    ]
  },
  "geometry": {
    "type": "euclidean",
    "max_arc": 0
  }
}
```

The same `RenderScript` schema is accepted in:

```jsonc
{
  "renders": [
    { "integrator": "path", "samples": 20 },
    { "integrator": "bdpt", "samples": 100 }
  ]
}
```

### Integrator-Relevant Fields

| JSON field           | Accepted/current meaning                                     | Controller default                                           | Important behavior                                           |
| -------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `integrator`         | `path`, `bdpt`, `light_tracing`, or alias `light_trace`      | `path`                                                       | Parsed to a canonical kind immediately before rendering      |
| `samples`            | Positive integer in normal controller use                    | `20`                                                         | Per-active-pixel target; global splat work derives from it   |
| `thread_num`         | Positive worker count                                        | `runtime.NumCPU()`                                           | Non-positive values from script do not override the default  |
| `spectrum_mode`      | `hero_wavelength`, `sampled`                                 | `hero_wavelength`                                            | Changes wavelength sampling; Film accumulation is always spectral |
| `wavelength_samples` | Positive integer                                             | `1`; promoted to `4` when resolved sampled mode is at most one | Used by path and BDPT sampled mode; not used to multiply light-tracing work |
| `pixel_windows`      | Array of half-open `{min,max}` coordinate boxes              | Entire film                                                  | Restricts active pixels; overlapping windows are de-duplicated |
| `camera_index`       | Camera selected for the render                               | `0`                                                          | Light tracing additionally requires the selected camera to be projective |
| `widths`             | Film dimensions as an integer array                          | Selected camera dimensions                                   | Affect active-pixel count and splat normalization            |
| `geometry.type`      | `euclidean`, `klein`, or `spherical`                         | Engine scene default                                         | Determines path-geodesic behavior and BDPT fallback          |
| `geometry.max_arc`   | Non-negative geodesic distance budget                        | `0` except spherical defaults to $2\pi$                      | Enforced by regular path tracing; BDPT only runs in Euclidean geometry |

The controller resolves defaults, then scene fields, then command-line overrides. The `renders` array creates independent jobs using the same schema.

### CLI Inputs

The Engine CLI exposes these relevant flags:

```text
--integrator path|bdpt|light_tracing
--samples N
--threads N
--spectrum-mode hero_wavelength|sampled
--wavelength-samples N
--pixel-window min:max,min:max
--camera-index N
--widths N,N
```

`--pixel-window` may be repeated. Both `light_tracing` and `light_trace` pass integrator validation even though only the canonical name is listed in the help string.

### Internal Controls That Are Not Scene Inputs

`ray_tracing.Handler` contains JSON tags for several fields, but the canonical controller does not deserialize a handler from the scene. It constructs `NewHandler()` and only copies selected render settings. Consequently these are fixed internal defaults in normal Engine scene rendering:

| Internal field         | Default         | Runtime role                      | Public scene/CLI override? |
| ---------------------- | --------------- | --------------------------------- | -------------------------- |
| `MaxRayLevel`          | `64`            | Hard subpath/bounce cap           | No                         |
| `RussianRouletteDepth` | `3`             | First depth eligible for roulette | No                         |
| `BlockCols`            | `8`             | Pixel-driver tile width           | No                         |
| `BlockRows`            | `8`             | Pixel-driver tile height          | No                         |
| `WavelengthSampler`    | Uniform sampler | Wavelength distribution           | No                         |

`MaxArc`, thread count, spectrum mode, wavelength count, and film color space are copied or resolved by the controller.


### Render Lifecycle

Every configured integrator follows the same scene-level lifecycle:

1. Validate non-nil camera, object tree, and film; reject negative sample count.
2. Prepare film color space and allocate 64 spectral bins when needed.
3. Run the selected driver.
4. Convert spectral bins to the film color space when spectral rendering is active.
5. Store the driver's effective sample count in `Film.Samples`.

Zero samples are accepted by the lower-level `RenderContext`. The pixel driver performs no work. Splat kernels may prepare first, then report zero work.

### Scheduling Comparison

| Property              | Pixel driver (`path`)                             | Splat driver (`bdpt`, `light_tracing`)                |
| --------------------- | ------------------------------------------------- | ----------------------------------------------------- |
| Work unit             | Tile/pixel                                        | Global path sample                                    |
| Worker allocation     | Atomic next-tile counter                          | Atomic next-work counter                              |
| Default spatial block | `8 x 8` for 2D films; 64-element chunks otherwise | Not tiled                                             |
| Film write ownership  | One worker owns the pixel while tracing it        | Any worker may add to any pixel                       |
| Synchronization       | No per-pixel locks                                | One mutex per film element                            |
| RGB write             | Path sets the final pixel mean                    | Splats add normalized contributions                   |
| Spectral write        | Path adds already normalized spectral samples     | Splats add $L(\lambda)/[p(\lambda)N_{\mathrm{work}}]$ |

If `ThreadNum` is non-positive at the driver layer, it is treated as one worker. Normal controller rendering resolves a positive CPU-count default.

### Pixel Windows

Pixel windows are half-open boxes `[min,max)` in every film dimension.

- The path driver builds work only for active pixels.
- BDPT selects camera pixels only from the active list and accepts delta splats only inside the mask.
- Light tracing launches $\mathtt{samples}\times\mathtt{activePixelCount}$ paths and accepts projections only inside the mask.
- Overlap does not duplicate a pixel.

For non-2D films, path work is linearized. Splat projection currently depends on the projective 3D camera and its 2D raster mapping.

### Recorded Sample Count

| Integrator/mode         | Actual principal work       | `Film.Samples` |
| ----------------------- | --------------------------- | -------------- |
| Path RGB/hero           | `S` paths per active pixel  | `S`            |
| Path sampled            | $SW$ paths per active pixel | $SW$           |
| BDPT RGB/hero           | $SP$ global work items      | $S$            |
| BDPT sampled            | $SPW$ global work items     | $S$            |
| Light tracing, any mode | $SP$ light paths            | $S$            |

This field is therefore driver-defined metadata, not a uniform count of every traced path across all integrators.

### Core Implementation Facts and Caveats

1. **Configured kind and effective algorithm are different concepts.** In particular, `bdpt` can execute the regular path estimator while retaining the splat schedule and BDPT film metadata.
2. **Only `Camera3D` is currently projective.** Light tracing rejects all other camera types; BDPT merely omits its delta-caustic splats when projection is unavailable.
3. **Light-origin algorithms need `SurfaceSampler`.** An object can be emissive and visible to path tracing without being eligible for light selection.
4. **BDPT's fallback warning is diagnostic, not an error.** Rendering continues with path samples.
5. **The controller does not preflight the BDPT scene capability gate.** Capability analysis occurs in `bdptKernel.Prepare` during rendering.
6. **The BDPT geometry reason text is stricter than its actual test.** It says three-dimensional Euclidean geometry, but the gate checks Euclidean kind only; later MIS helpers require exactly three components.
7. **Light tracing has no equivalent capability gate.** Its correctness domain is constrained indirectly by `ProjectiveCamera`, Euclidean visibility, and three-dimensional camera projection.
8. **Spectral work differs by integrator.** Path and BDPT use wavelength strata in sampled mode; light tracing always samples one wavelength per global path.
9. **No integrator performs volumetric scattering.** Media only attenuate segments and update IOR/boundary state.
10. **No integrator samples an environment light.** A miss is black. Emission comes from intersected or area-sampled objects.
11. **Regular path tracing has no next-event estimation.** Small or difficult-to-hit lights can have high variance.
12. **Continuous BDPT deliberately rejects delta measures.** Delta-caustic splats are a separate, non-MIS path family.
13. **Random samples use `math/rand/v2` package-level generation.** The integrator input schema exposes no seed or sampler-selection control.
14. **Unknown JSON fields are normally ignored by `encoding/json`.** Integrator value validation occurs later through `ParseIntegratorKind`; CLI values are validated during flag parsing.

| Concern | Engine source |
| --- | --- |
| Names, aliases, runtime selection | `engine/ray_tracing/integrator.go` |
| Pixel/splat scheduling and normalization | `engine/ray_tracing/render_driver.go` |
| Session validation, accumulation, finalization | `engine/ray_tracing/render_session.go` |
| Film preparation and integrator entry point | `engine/ray_tracing/trace_scene.go` |
| Path kernel and spectral sampling | `engine/ray_tracing/trace_pixel.go` |
| Recursive camera-path transport | `engine/ray_tracing/trace_ray.go` |
| Throughput and roulette helpers | `engine/ray_tracing/throughput.go` |
| Shared homogeneous transmittance | `engine/ray_tracing/medium_transport.go` |
| Light tracing and camera projection | `engine/ray_tracing/light_trace.go` |
| BDPT paths, connection, densities, MIS, fallback gates | `engine/ray_tracing/bdpt.go` |
| BDPT work mapping and delta-caustic splats | `engine/ray_tracing/bdpt_kernel.go` |
| Camera projection contract | `engine/model/camera/camera.go`, `camera_3d.go` |
| Handler defaults | `engine/ray_tracing/handler.go` |
| Scene JSON schema | `engine/controller/parser/schema.go` |
| Defaults, CLI flags, override resolution | `engine/controller/render_context.go` |
| Controller-to-render-handler wiring | `engine/controller/handler.go` |
