# Ray Tracing Engine

This directory contains the active Go implementation of the ray tracing engine.

## Layout

```text
engine/go/
  cmd/ray/                 CLI entry point
  internal/app/            Application flow, config resolution, output handling
  internal/controller/     Scene JSON parsing and scene construction
  internal/model/          Scene, camera, object, optics, and shape models
  internal/ray_tracing/    Core tracing pipeline
  internal/utils/          Math helpers, global settings, and object pools
```

## Runtime Flow

1. `cmd/ray` delegates to `internal/app.Run`.
2. The app parses CLI flags and resolves render config.
3. The controller loads the scene script and builds the scene.
4. The renderer traces the selected camera into a film.
5. Output files are written to the configured paths.

## Defaults

The default scene path is `../../examples/scenes/default.json` when commands are run with `go -C engine/go ...`.

Default outputs:

- `../../outputs/output.png`
- `../../outputs/img.bin`
- `../../outputs/debug_traces.json`

## Common Commands

From the repository root:

```bash
npm run ray
npm run ray:test
npm run ray:build
```
