# Math and Physics in `engine/go`

This directory organizes the mathematical and physical ideas embedded in the Go ray tracing project. The goal is not only to describe the code, but to explain the theory that the code is implementing and where each idea appears in the codebase.

## Document Map

- [`01-mathematical-foundations.md`](01-mathematical-foundations.md): linear algebra, coordinate systems, root solving, gradients, and Monte Carlo estimation.
- [`02-geometry-and-intersection.md`](02-geometry-and-intersection.md): geometric primitives, implicit surfaces, mesh handling, and the bounding-volume hierarchy.
- [`03-optics-and-materials.md`](03-optics-and-materials.md): reflection, refraction, Fresnel effects, total internal reflection, diffuse scattering, wavelength, and dispersion.
- [`04-cameras-rendering-and-scene.md`](04-cameras-rendering-and-scene.md): camera projection, N-dimensional imaging, recursive ray tracing, film accumulation, concurrency, and scene scripts.
- [`material-system-design.md`](material-system-design.md): current material/BxDF architecture, schema, validation rules, IOR/dispersion wiring, microfacet status, and output transform controls.
- [`spectral-modernization-plan.md`](spectral-modernization-plan.md): staged technical plan for moving from RGB/hero-wavelength rendering toward a modern spectral color pipeline.
- [`scene-json-current.md`](scene-json-current.md): current JSON schema for materials, render output controls, and showcase scenes.

## Knowledge Map

The project combines four layers of knowledge:

1. **Linear algebra**
   Vector addition, scalar multiplication, normalization, dot products, cross products, orthonormalization, and tensor-like storage are used throughout the project.

2. **Geometric modeling**
   Surfaces are represented either as explicit primitives such as spheres, triangles, and cuboids, or as algebraic surfaces such as quadratic and fourth-order implicit equations.

3. **Geometrical optics and BSDF materials**
   Rays carry position, direction, color, wavelength, wavelength PDF, and refractive index. Current scene objects use the `internal/material` BSDF/BxDF system for diffuse, specular, dielectric, dispersive, rough-conductor, and emissive behavior.

4. **Monte Carlo rendering**
   Pixels are estimated by averaging repeated stochastic ray samples. Randomized camera jitter and randomized material branching are the main sampling mechanisms.

## Primary Code Entry Points

- Render pipeline: `main.go`, `handler.go`, `ray_tracing/`
- Scene parsing: `controller/`
- Cameras and film: `model/camera/`
- Optical state: `model/optics/`
- Current material system: `material/`
- Shapes and acceleration structures: `model/shape/`, `model/object/`
- Geometrical optics utilities: `utils/geometrical_optics.go`

## Reading Guide

If you want to understand the project from first principles, the recommended order is:

1. `01-mathematical-foundations.md`
2. `02-geometry-and-intersection.md`
3. `03-optics-and-materials.md`
4. `04-cameras-rendering-and-scene.md`
5. `material-system-design.md`

If you want to extend the renderer, start from `04-cameras-rendering-and-scene.md` and then jump to the topic-specific document for the subsystem you want to change.
