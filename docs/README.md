# Math, Physics, and Architecture in `engine`

This directory documents the mathematical, physical, and code-architecture ideas embedded in the Go ray tracing project. The goal is to connect theory to the current package layout.

## Document Map

- [`01-mathematical-foundations.md`](01-mathematical-foundations.md): linear algebra, coordinate systems, root solving, gradients, and Monte Carlo estimation.
- [`02-geometry-and-intersection.md`](02-geometry-and-intersection.md): geometric primitives, implicit surfaces, mesh handling, and the bounding-volume hierarchy.
- [`03-optics-and-materials.md`](03-optics-and-materials.md): rays, spectra, reflection, refraction, Fresnel effects, wavelength sampling, dispersion, and BSDF/BxDF materials.
- [`04-cameras-rendering-and-scene.md`](04-cameras-rendering-and-scene.md): camera projection, N-dimensional imaging, recursive ray tracing, film accumulation, concurrency, and scene scripts.
- [`material-system-design.md`](material-system-design.md): current material/BxDF architecture, schema, validation rules, IOR/dispersion wiring, microfacet status, and output transform controls.
- [`scene-json-current.md`](scene-json-current.md): current JSON schema for media, materials, objects, cameras, render output controls, and showcase scenes.
- [`controller-json-rules.md`](controller-json-rules.md): controller/parser/factory rules for consuming scene JSON.
- [`spectral-modernization-plan.md`](spectral-modernization-plan.md): staged technical plan for moving from RGB/hero-wavelength rendering toward a modern spectral color pipeline.
- [`current-architecture.md`](current-architecture.md): current package boundaries for optics, materials, media, ray tracing, spectrum modes, and color-space types.
- [`medium-and-caustics-modernization-plan.md`](medium-and-caustics-modernization-plan.md): staged plan for nested dielectric media, explicit medium boundaries, homogeneous absorption, participating-media scattering, and caustic-capable prism validation.
- [`ray-tracing-map.json`](ray-tracing-map.json): exported architecture/map data used by external diagramming or note tools.

## Knowledge Map

The project combines four layers of knowledge:

1. **Linear algebra**
   Vector addition, scalar multiplication, normalization, dot products, cross products, and tensor-like storage are used throughout the renderer.

2. **Geometric modeling**
   Surfaces are represented either as explicit primitives such as spheres, circles, cylinders, triangles, and cuboids, or as algebraic surfaces such as quadratic and fourth-order implicit equations.

3. **Geometrical optics and BSDF materials**
   Rays carry position, direction, color weight, wavelength, wavelength PDF, refractive index, and a medium stack. Scene objects use the `engine/model/material` BSDF/BxDF system for diffuse, specular, dielectric, dispersive, rough-conductor, emissive, and medium-boundary behavior.

4. **Monte Carlo rendering**
   Pixels are estimated by averaging stochastic camera and material samples. Rendering can operate in RGB, hero-wavelength, or sampled wavelength modes.

## Primary Code Entry Points

- CLI and render orchestration: `engine/main.go`, `engine/controller/`
- Scene parsing and construction: `engine/controller/parser/`, `engine/controller/factory/`
- Cameras and film: `engine/model/camera/`
- Optical state and spectra: `engine/model/optics/`
- Material system: `engine/model/material/`
- Shapes and BVH acceleration: `engine/model/shape/`, `engine/model/object/`
- Integrator and render loop: `engine/ray_tracing/`
- Shared numeric/parsing helpers: `engine/utils/`

## Reading Guide

If you want to understand the project from first principles, read:

1. `01-mathematical-foundations.md`
2. `02-geometry-and-intersection.md`
3. `03-optics-and-materials.md`
4. `04-cameras-rendering-and-scene.md`
5. `material-system-design.md`

If you want to extend the renderer, start from `04-cameras-rendering-and-scene.md`, then jump to the subsystem-specific document.
