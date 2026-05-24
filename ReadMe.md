# Ray Tracing

A Go-based ray tracing engine for optical simulation and physically inspired rendering. The repository contains the renderer, JSON scene examples, design notes, rendered examples, and a React scene editor.

![feature-showcase-800x800-2000spp.png](docs%2Fassets%2Ffeature-showcase-800x800-2000spp.png)
![material-benchmark-matrix.png](docs%2Fassets%2Fmaterial-benchmark-matrix.png)
![RayTracingTest.png](docs%2Fassets%2FRayTracingTest.png)
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
