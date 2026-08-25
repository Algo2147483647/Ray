# Integrators

This document describes the Integrator implementation that exists in the
current Engine. The serialized values are `path`, `bdpt`, and `light_tracing`;
`light_trace` remains a parser alias for `light_tracing`.

The selected Integrator is never changed at runtime. Preparation or capability
failure returns an error before samples are committed.

## Runtime Contracts

`SceneIntegrator` exposes:

```go
Run(*RenderContext) error
EffectiveSampleCount(*RenderContext) int64
ConcurrentFilmWrites() bool
```

`NewSceneIntegrator` maps the requested kind directly:

| Kind | Implementation | Work and writes |
| --- | --- | --- |
| `path` | `pixelSceneIntegrator` + `pathTracingKernel` | Atomic tile claims; each pixel has one owner |
| `bdpt` | `splatSceneIntegrator` + `bdptKernel` | Global work indices; contributions may target arbitrary pixels |
| `light_tracing` | `splatSceneIntegrator` + `lightTracingKernel` | Global light paths; eligible vertices splat to projected pixels |

The shared `RenderContext` contains the selected Camera, ObjectTree, sample
count, Handler, and FilmAccumulator. It does not contain Scene dimension or an
alternate Integrator state.

## Path

Path tracing generates camera Rays, samples wavelengths, intersects in the
Scene Geometry, evaluates emission and scattering events, applies homogeneous
Beer–Lambert absorption, updates medium boundaries after transmission, and uses
Russian roulette after the configured depth. It supports Euclidean, Klein, and
spherical propagation subject to each Shape's intersection capability.

## BDPT

BDPT builds camera and finite-area-light subpaths, connects compatible vertices,
and weights its continuous strategy family with MIS. Supported delta-caustic
camera splats are accounted separately.

BDPT preflight rejects, among other unsupported inputs:

- non-Euclidean Scene Geometry;
- a non-2D Film or non-projective Camera;
- non-reciprocal surfaces;
- thin or unsupported transmission boundaries;
- delta, environment-like, or non-sampleable emitters;
- Scenes without a finite sampleable area light.

Any rejection is returned to the caller. Running Path instead requires a new,
explicit Render job with `integrator: "path"`.

## Light Tracing

Light tracing requires a `ProjectiveCamera` and a 2D Film. It samples finite
area lights, walks light subpaths, projects eligible vertices, checks visibility,
and emits synchronized Film splats. A Scene with no usable light distribution
produces no work rather than changing algorithms.

## Media Boundary

All Integrators support homogeneous absorption (`sigma_a`) and IOR boundary
changes. Participating-medium scattering is not implemented, so `sigma_s` is
rejected by the Engine protocol and does not exist in the runtime Medium model.

## Render Input

Each item in top-level `renders` may select:

- `integrator`;
- `camera_id`;
- `samples`;
- `thread_num`;
- `spectrum_mode`;
- `wavelength_samples`.

These values are complete before the Integrator is created. Runtime handlers do
not infer render defaults.

Every scene Integrator has two phases. `Prepare` validates capabilities and
returns algorithm-specific prepared state; `Run` consumes that exact state.
BDPT scene traversal, surface validation, and finite-area-light collection occur
only in its single Prepare call.

Film shape, spectral bins, output path, and pixel windows belong to the selected
RenderTarget. Cameras are reusable imaging models; geometry and dimension belong
to the Scene.
