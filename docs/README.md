# Math, Physics, and Architecture in `engine`

This directory documents the mathematical, physical, and code-architecture ideas embedded in the Go ray tracing project. The goal is to connect theory to the current package layout.

## Document Map

- [`mathematical-foundations.md`](mathematical-foundations.md): linear algebra, coordinate systems, root solving, gradients, and Monte Carlo estimation.
- [`geometry-intersection-theory.md`](geometry-intersection-theory.md): geometric primitives, implicit surfaces, mesh handling, and the bounding-volume hierarchy.
- [`geometry-shape-details.md`](geometry-shape-details.md): current geometry shape support matrix, JSON fields, bounds rules, dimensions, and implementation status.
- [`optics-and-materials.md`](optics-and-materials.md): rays, spectra, reflection, refraction, Fresnel effects, wavelength sampling, dispersion, and BSDF/BxDF materials.
- [`rendering-flow-cameras-and-scenes.md`](rendering-flow-cameras-and-scenes.md): camera projection, N-dimensional imaging, recursive ray tracing, film accumulation, concurrency, and scene scripts.
- [`material-system-design.md`](material-system-design.md): current material/BxDF architecture, schema, validation rules, IOR/dispersion wiring, microfacet status, and output transform controls.
- [`material-capability-coverage.md`](material-capability-coverage.md): current engine coverage for real-world material families, implemented BxDFs, and missing/planned material models.
- [`current-scene-json.md`](current-scene-json.md): current JSON schema for media, materials, objects, cameras, render output controls, and showcase scenes.
- [`controller-json-rules.md`](controller-json-rules.md): controller/parser/factory rules for consuming scene JSON.
- [`spectral-modernization-plan.md`](spectral-modernization-plan.md): staged technical plan for moving from RGB/hero-wavelength rendering toward a modern spectral color pipeline.
- [`current-renderer-architecture.md`](current-renderer-architecture.md): current package boundaries for optics, materials, media, ray tracing, spectrum modes, and color-space types.
- [`medium-and-caustics-modernization-plan.md`](medium-and-caustics-modernization-plan.md): staged plan for nested dielectric media, explicit medium boundaries, homogeneous absorption, participating-media scattering, and caustic-capable prism validation.
- [`multidimensional-rendering.md`](multidimensional-rendering.md): implemented N-dimensional camera/film support, 4D hypercube and hypersphere scenes, and the geometric interpretation of exterior and interior hypercube experiments.
- [`ray-tracing-map.json`](ray-tracing-map.json): exported architecture/map data used by external diagramming or note tools.

## Knowledge Map

The project combines five layers of knowledge:

1. **Linear algebra**
   Vector addition, scalar multiplication, normalization, dot products, cross products, and tensor-like storage are used throughout the renderer.

2. **Geometric modeling**
   Surfaces are represented either as explicit primitives such as spheres, hyperspheres, cuboids, hypercubes, circles, cylinders, and triangles, or as algebraic surfaces such as quadratic, cubic, fourth-order, sparse polynomial, and non-polynomial implicit equations.

3. **Geometrical optics and BSDF materials**
   Rays carry position, direction, color weight, wavelength, wavelength PDF, refractive index, and a medium stack. Scene objects use the `engine/model/material` BSDF/BxDF system for diffuse, specular, dielectric, dispersive, rough-conductor, emissive, and medium-boundary behavior.

4. **Monte Carlo rendering**
   Pixels are estimated by averaging stochastic camera and material samples. Rendering can operate in RGB, hero-wavelength, or sampled wavelength modes.

5. **Higher-dimensional measurement**
   N-dimensional cameras and tensor films allow 4D scenes to be sampled through 2D or 3D observation domains. The current showcase uses this to inspect the eight cubic cells of a hypercube and the four-cell incidence structure at a hypercube vertex.

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

1. `mathematical-foundations.md`
2. `geometry-intersection-theory.md`
3. `optics-and-materials.md`
4. `rendering-flow-cameras-and-scenes.md`
5. `material-system-design.md`
6. `multidimensional-rendering.md`

If you want to extend the renderer, start from `rendering-flow-cameras-and-scenes.md`, then jump to the subsystem-specific document.
