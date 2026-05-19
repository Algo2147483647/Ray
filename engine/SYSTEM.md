# Ray Tracing Engine

This directory contains the active Go implementation of the ray tracing engine.

## Layout

```text
engine/
  main.go                CLI entry point
  app/                   Application flow, config resolution, output handling
  controller/            Compatibility facade over scene IO
  model/                 Scene, camera, material, object, optics, and shape models
  ray_tracing/           Core tracing pipeline
  sceneio/               Scene schema, parser, and factory code
  utils/                 Math helpers, global settings, and object pools
```

## Runtime Flow

1. `main.go` delegates to `app.Run`.
2. The app parses CLI flags and resolves render config.
3. The controller loads the scene script through `sceneio`.
4. The renderer traces the selected camera into a film.
5. Output files are written to the configured paths.

## Defaults

The default scene path is `../../examples/scenes/default.json` when commands are run from `engine`.

Default outputs:

- `../../outputs/output.png`
- `../../outputs/img.bin`

## Common Commands

From the repository root:

```bash
go -C engine run .
go -C engine test ./...
go -C engine build -o ../bin/ray .
```
