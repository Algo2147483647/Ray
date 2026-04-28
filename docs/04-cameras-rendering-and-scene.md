# Cameras, Rendering, and Scene Modeling

This document explains how the project turns scene descriptions into rays, rays into samples, and samples into an image.

## 1. End-to-End Rendering Flow

At a high level, the program performs:

1. parse command-line overrides,
2. load a JSON scene script,
3. build materials, objects, and cameras,
4. select the active camera,
5. trace rays for all pixels,
6. save the film and image outputs.

Relevant code:

- `main.go`
- `handler.go`
- `render_config.go`

## 2. Scene as the Main Physical Container

The scene stores:

- an object tree,
- a list of cameras.

Conceptually, the scene is the full optical experiment:

- geometry defines interfaces,
- materials define interaction laws,
- the camera defines measurement.

Relevant code:

- `model/scene.go`
- `controller/srcipt.go`

## 3. Scene Scripts as a Declarative Model

The JSON script maps physical and geometric concepts into data:

- `materials`: optical properties,
- `objects`: geometry plus material assignment,
- `cameras`: measurement configuration,
- `render`: numerical and output settings.

This makes the project a small domain-specific environment for optical scene construction rather than just a hard-coded demo renderer.

Relevant code:

- `controller/srcipt.go`
- `controller/parse_materials.go`
- `controller/parse_shape.go`
- `controller/parse_camera.go`

## 4. 3D Camera Model

The main camera is a standard perspective camera defined by:

- position,
- viewing direction or look-at target,
- up vector,
- field of view,
- image resolution,
- aspect ratio.

The camera converts pixel indices into rays by:

1. computing normalized image-plane coordinates,
2. adding random sub-pixel jitter,
3. building the `right` vector from `direction x up`,
4. offsetting the forward direction by horizontal and vertical field-of-view terms,
5. normalizing the result.

This is the project’s measurement model: it specifies which light paths are sampled by each pixel.

Relevant code:

- `model/camera/camera_3d.go`
- `controller/parse_camera.go`

## 5. N-Dimensional Camera Model

The project contains a generalized `CameraNDim` that extends the same idea to arbitrary dimension.

The mathematical pattern is:

- one forward basis vector,
- one image-axis basis vector per sensor dimension,
- one field-of-view value per sensor dimension,
- one discrete coordinate per sensor dimension.

The camera orthonormalizes its basis, converts grid coordinates to normalized offsets, and forms:

```text
d' = normalize(e0 + sum_i u_i tan(FOV_i / 2) e_i)
```

where `e0` is the forward basis vector and the other `e_i` are image-axis directions.

This is one of the most distinctive ideas in the codebase. It shows that the renderer is not only about standard 3D imaging, but also about experimenting with generalized projection in higher-dimensional spaces.

Relevant code:

- `model/camera/camera_n_dim.go`
- `model/camera/camera_n_dim_test.go`

## 6. Pixel Sampling

For each pixel, the renderer:

1. generates `N` rays,
2. traces each ray recursively,
3. accumulates the returned color,
4. averages over the sample count.

Mathematically:

```text
pixel(x, y) = (1 / N) sum_i TraceRay_i
```

This is the discrete Monte Carlo estimator for the pixel intensity.

Relevant code:

- `ray_tracing/trace_pixel.go`

## 7. Recursive Ray Tracing

`TraceRay` is the core light-transport routine.

For each ray:

1. stop if recursion depth exceeds `MaxRayLevel`,
2. query the nearest hit from the object tree,
3. if no hit exists, return black,
4. move the ray origin to the hit point,
5. compute the surface normal,
6. orient the normal against the incoming ray,
7. apply the material interaction,
8. recurse with the updated ray.

This structure expresses the physical idea that the path of light is piecewise linear between interaction events.

Relevant code:

- `ray_tracing/trace_ray.go`

## 8. Why the Normal May Be Flipped

After intersection, the code flips the normal if it points in the same general direction as the incoming ray. This is important because reflection and refraction formulas assume a consistent surface orientation relative to the incoming path.

Without this correction:

- reflection could point inward when it should point outward,
- refraction tests could use the wrong incidence angle,
- medium transitions could be interpreted incorrectly.

Relevant code:

- `ray_tracing/trace_ray.go`

## 9. Film as a Tensor-Valued Measurement Buffer

The film stores three tensors, one for each RGB channel, plus a sample count.

This is mathematically a sampled function over a discrete grid:

```text
Film: grid -> R^3
```

The design also supports more than two spatial axes. If the film tensor has three dimensions, the image export lays out slices vertically in one 2D image.

This matches the project's general higher-dimensional mindset.

Relevant code:

- `model/camera/film.go`

## 10. Merging Films

Two films with the same shape can be merged by weighted averaging using their sample counts:

```text
F = (s1 F1 + s2 F2) / (s1 + s2)
```

This is statistically correct for combining independent sample batches from the same estimator.

Relevant code:

- `model/camera/film.go`

## 11. Parallel Rendering

The renderer parallelizes across pixels. Worker goroutines repeatedly claim the next pixel index using an atomic counter and write the corresponding RGB values into the film.

This is a standard task-parallel pattern for embarrassingly parallel rendering workloads:

- each pixel estimate is independent,
- synchronization cost is small,
- load balancing is handled dynamically by atomic work distribution.

Relevant code:

- `ray_tracing/trace_scene.go`

## 12. Ray Pooling and Allocation Control

The render handler maintains a `sync.Pool` of rays. This reduces repeated allocation during sampling and recursive tracing.

From a systems perspective, this is not physics, but it supports the practical numerical workload created by the mathematical model.

Relevant code:

- `ray_tracing/handler.go`
- `utils/pools.go`

## 13. Render Configuration as a Numerical Experiment Setup

The render configuration controls:

- camera selection,
- thread count,
- output resolution,
- samples per pixel,
- output paths.

This can be interpreted as the numerical setup of the simulation:

- resolution controls spatial discretization,
- sample count controls estimator variance,
- recursion depth controls path complexity,
- thread count controls computational throughput.

Relevant code:

- `render_config.go`

## 14. What the Default Scene Demonstrates

The sample `test.json` showcases several important ideas:

- emissive triangles as area-light approximations,
- diffuse room boundaries,
- a quadratic implicit surface,
- fourth-order algebraic surfaces,
- dispersive glass with Cauchy coefficients,
- perspective viewing from a configurable camera.

So even the default scene is already a compact demonstration of:

- analytic geometry,
- optical interfaces,
- stochastic rendering,
- wavelength-dependent refraction.

Relevant code:

- `test.json`

## 15. Important Current Boundaries

To understand the project accurately, it helps to separate implemented capability from architectural direction.

Currently true:

- runtime spatial dimension is fixed to `3` via `utils.Dimension`,
- `CameraNDim` and 4D diffuse sampling exist as generalized experiments,
- plane geometry exists in code but is blocked in script parsing,
- `ImplicitEquation` is a stub rather than a finished numeric intersector.

This means the codebase contains both:

- production-useful 3D ray tracing features,
- exploratory infrastructure for more advanced mathematical rendering work.

## 16. Summary

The rendering subsystem embeds the following ideas:

- perspective projection,
- generalized N-dimensional projection,
- recursive light transport,
- Monte Carlo pixel estimation,
- tensor-based image storage,
- sample-consistent film merging,
- parallel workload distribution,
- declarative optical scene specification.

Together, these pieces turn the geometry and optics layers into a complete simulation-and-rendering pipeline.
