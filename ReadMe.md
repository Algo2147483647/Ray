# Ray Tracing

A Go-based ray tracing engine for optical simulation and physically inspired rendering. The project includes a renderer, scene JSON examples, documentation, and a React scene editor.


![feature-showcase-800x800-2000spp.png](docs%2Fassets%2Ffeature-showcase-800x800-2000spp.png)
![four-order-equation-wide-soft-gold-800x800-1000spp.png](docs%2Fassets%2Ffour-order-equation-wide-soft-gold-800x800-1000spp.png)
![RayTracingTest.png](docs%2Fassets%2FRayTracingTest.png)
![4D.png](docs%2Fassets%2F4D.png)

## Project Structure

```text
Ray/
  apps/
    scene-editor/        React/TypeScript scene editor
  docs/                  Design notes, math notes, and rendered examples
  engine/
    go/                  Go ray tracing engine
      cmd/ray/           CLI entry point
      internal/app/      Application orchestration and render config
      internal/controller/
      internal/model/
      internal/ray_tracing/
      internal/utils/
  examples/
    scenes/              Scene JSON files
  outputs/               Render outputs, ignored by git
```

The previous C++ implementation has been removed so the repository has one active engine path.

## Requirements

- Go 1.24+
- Node.js 16+ for the scene editor
- Python 3.x for auxiliary visualization scripts in the editor tools

## Render From The CLI

From the repository root:

```bash
npm run ray
```

Use a specific scene:

```bash
npm run ray -- --script ../../examples/scenes/default.json
```

Useful render flags:

```bash
npm run ray -- --width 800 --height 600 --samples 64 --threads 8
npm run ray -- --output-image ../../outputs/render.png
```

The default scene is `examples/scenes/default.json`. Default outputs are written under `outputs/`.

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

The editor lives in `apps/scene-editor` and provides a structured interface for viewing and editing scene data.

## Scene Scripts

Scenes are JSON files containing materials, objects, cameras, and optional render settings:

```json
{
  "materials": [
    {
      "id": "glass",
      "color": [1, 1, 1],
      "reflectivity": 0.1,
      "refractivity": 0.9,
      "refractive_index": [1.5]
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
  ]
}
```

## Extending The Engine

To add a shape:

1. Add the implementation under `engine/go/internal/model/shape`.
2. Implement the `Shape` interface.
3. Register JSON parsing in `engine/go/internal/controller/parse_shape.go`.

To add material behavior:

1. Update `engine/go/internal/model/optics/material.go`.
2. Add or update parser support under `engine/go/internal/controller`.
3. Add focused tests beside the changed package.

## Documentation

More detailed notes are in `docs/`, including mathematical foundations, geometry and intersection behavior, optics/material notes, and scene JSON rules.
