# Ray Tracing

A Go-based ray tracing engine for optical simulation and physically inspired rendering. The repository contains the renderer, JSON scene examples, design notes, rendered examples, and a React scene editor.

## Rendering Result Demonstration

### Complex geometry rendering

The geometry benchmark experiment was designed to validate the renderer’s ability to handle a broad spectrum of geometric representations within a single controlled scene. Rather than focusing on one primitive type, the scene arranges many objects in a matrix-like layout and renders them under shared camera, lighting, exposure, and tone-mapping conditions. This makes the image suitable for comparing intersection behavior, normal evaluation, shading consistency, and material response across heterogeneous geometry classes.

The test set includes conventional primitives such as spheres, cuboids, cylinders, circles, and triangle meshes, together with algebraic surfaces defined by quadratic, cubic, and fourth-order equations. It also includes more general procedural forms, including polynomial surfaces and implicit equations such as gyroid-like structures. The purpose of this arrangement is twofold: first, to confirm that standard closed-form primitives and mesh-based geometry remain stable in the renderer; second, to demonstrate that the system has been extended beyond fixed primitive libraries to support arbitrary-order polynomial surfaces and arbitrary implicit surfaces.

During rendering, each object is placed into a common benchmark stage and observed from a fixed perspective camera. The use of uniform rendering parameters ensures that visual differences arise primarily from geometry and material behavior rather than from scene-specific tuning. Smooth algebraic objects test root solving, surface-normal computation, and curvature-dependent highlights, while triangle-based objects test planar boundaries and mesh composition. Implicit surfaces further stress the ray-marching or intersection procedure because their visible shape is determined by evaluating a scalar field rather than by intersecting a predefined analytic primitive.

The resulting matrix image shows that the renderer can produce coherent shading and silhouettes across multiple geometry families. Simple primitives exhibit clean boundaries and predictable highlights, confirming the correctness of baseline intersection routines. Higher-order polynomial and implicit surfaces display more complex curvature, cavities, and self-varying topology, demonstrating that the renderer can represent forms that are difficult or impossible to express using only basic primitives. Overall, the experiment verifies the generality of the geometry subsystem: the renderer is not limited to a fixed set of hand-coded shapes, but can serve as a framework for rendering arbitrary polynomial and implicit geometry under a unified ray-tracing pipeline.

![geometry-benchmark-matrix.png](docs%2Fassets%2Fgeometry-benchmark-matrix.png)

### Material rendering

![material-benchmark-matrix.png](docs%2Fassets%2Fmaterial-benchmark-matrix.png)


### High-dimensional space rendering

The experiment was designed to evaluate two complementary visualization strategies for a four-dimensional hypercube: physically motivated appearance and topology-oriented segmentation. In both scenes, the same 4D hypercube was placed at the origin with equal extent along all four axes, and it was observed using an identical external 4D orthographic camera. This controlled configuration ensured that differences in the resulting images were caused by the material and rendering model rather than by changes in geometry or viewpoint.

In the first rendering, the hypercube was assigned a Lambertian diffuse material with a light neutral albedo, and illumination was provided by an emissive hypersphere positioned off-center in 4D space. The purpose of this design was to test whether conventional diffuse shading can convey the geometric structure of a 4D object after projection into the image domain. The warm area light introduces gradual intensity variation across visible cells and faces, making the projected form easier to perceive as a coherent solid. Reinhard tone mapping, gamma correction, and a higher sample count were used to produce a stable, visually smooth result.

The second rendering replaces the physically based surface model with a cell-palette material, assigning different solid colors to the eight cubic cells of the hypercube. This version is not intended to simulate realistic illumination; instead, it functions as an analytical visualization. By removing shading ambiguity and encoding cell identity directly through color, the image exposes how the eight constituent volumes contribute to the final 4D projection.

Together, the two results demonstrate a useful trade-off. The Lambertian rendering provides perceptual continuity, depth cues, and a more intuitive sense of global form, but individual cells can be difficult to distinguish. The colored-cell rendering sacrifices physical realism in favor of structural clarity, making adjacency and cell decomposition more explicit. The comparison therefore shows that realistic shading and categorical coloring serve different but complementary roles in 4D visualization: one supports spatial perception, while the other supports topological interpretation.

![4d-hypercube-geometry-focus-centered-combined.png](docs%2Fassets%2F4d-hypercube-geometry-focus-centered-combined.png)



![4D.png](docs%2Fassets%2F4D.png)

## Project Structure

```text
Ray/
  docs/                  Design notes, schema notes, and rendered examples
  engine/                Go renderer
    main.go              CLI entry point
    controller/          CLI-facing orchestration, render config, JSON parsing, scene factory
      factory/           Builds cameras, materials, media, objects, and shapes from JSON
      parser/            Scene JSON structs and file loading
    model/               Renderer domain model
      camera/            3D and N-dimensional cameras, film storage, output transforms
      material/          Material container, BSDF/BxDF, emission, media, microfacet models
      object/            Objects, BVH build/update, traversal, surface hit records
      optics/            Rays, spectra, wavelength sampling helpers
      shape/             Analytic primitives and intersection logic
    ray_tracing/         Integrator, pixel sampling, recursive path tracing, render tiling
    utils/               Shared numeric and parsing helpers
  examples/scenes/       Scene JSON files
  outputs/               Render outputs, ignored by git
  scene-editor/          React/TypeScript scene editor
```

The active engine is the Go implementation under `engine/`.

## Requirements

- Go 1.24+
- Node.js 16+ for the scene editor
- Python 3.x only for auxiliary editor/visualization tooling

## Render From The CLI

From the repository root:

```bash
npm run ray
```

Use a specific scene:

```bash
npm run ray -- --script ../examples/scenes/feature-showcase.json
```

Useful render flags:

```bash
npm run ray -- --width 800 --height 600 --samples 64 --threads 8
npm run ray -- --output-image ../outputs/render.png
```

Default outputs are written under `outputs/`.

## Go Development

```bash
npm run ray:test
npm run ray:build
```

The build command writes the CLI binary to `bin/ray`.

## Scene Editor

```bash
npm run ui:install
npm run ui
```

The editor lives in `scene-editor` and provides a structured interface for viewing and editing scene data.

## Scene Scripts

Scenes are JSON files containing optional media, materials, objects, cameras, and render settings:

```json
{
  "materials": [
    {
      "id": "glass",
      "surface": {
        "type": "specular_dielectric",
        "reflectance": [1, 1, 1],
        "transmittance": [1, 1, 1],
        "ior": {
          "type": "constant",
          "eta": 1.5
        }
      }
    }
  ],
  "objects": [
    {
      "id": "ball",
      "shape": "sphere",
      "material_id": "glass",
      "position": [0, 0, 0],
      "r": 1
    }
  ],
  "cameras": [
    {
      "type": "3d",
      "position": [-4, 0, 1],
      "look_at": [0, 0, 0],
      "field_of_view": 60
    }
  ],
  "render": {
    "samples": 64,
    "spectrum_mode": "hero_wavelength"
  }
}
```

See [`docs/scene-json-current.md`](docs/scene-json-current.md) for the current schema.

## Extending The Engine

To add a shape:

1. Add the implementation under `engine/model/shape`.
2. Implement the `Shape` interface.
3. Register JSON parsing in `engine/controller/factory/shapes.go`.

To add material behavior:

1. Add the BxDF, BSDF, emission, or medium code under `engine/model/material`.
2. Add parser support under `engine/controller/factory`.
3. Add focused tests beside the changed package.

## Documentation

More detailed notes are in `docs/`, including mathematical foundations, geometry and BVH behavior, optics/material notes, camera/rendering flow, and scene JSON rules.
