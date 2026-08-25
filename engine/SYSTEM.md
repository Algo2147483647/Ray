# Engine System

This file is the canonical architecture description for the current Engine.
It describes only code that exists today.

The Engine accepts exactly one command-line option:

```text
--script <canonical-engine-scene.json>
```

Omitting `--script` is an error. Authoring composition, defaults, image output,
resume, and CLI render overrides belong to Studio.

## Spec

`controller/parser` decodes the canonical Engine JSON protocol. A script owns
one scene dimension, Geometry, typed Material and Object specs, Cameras, Media,
and Render jobs. Object shapes, Material surfaces, emission models, emission
distributions, and IOR models use validated discriminators. Unknown fields and
fields belonging to a different discriminator variant are rejected at this
boundary. Raw JSON is confined to genuinely polymorphic leaves such as
spectral parameters, expression fields, parametric functions, and sparse
coefficient encodings. Each Camera owns its Film. Each Render selects a Camera
and an Integrator.

## Compile

`controller/factory.LoadSceneFromScript` resolves the script into `model.Scene`.
It constructs a non-nil `SceneSpace{Geometry, Dimension}`, validates fixed
non-Euclidean dimensions, compiles typed Material/Object specs and
absorption-only Media, parses Shapes and Cameras with that SceneSpace, and
builds the ObjectTree acceleration structure. Numerical and cross-reference
validation happens here; unsupported configuration is not retained as inactive
runtime state.

## Render

`controller.Handler` executes each Render job against the compiled Scene.
`ray_tracing.NewHandler` receives the SceneSpace explicitly and creates Rays for
that space. `NewSceneIntegrator` selects exactly the requested algorithm:

- `path`: tiled camera-path work through `pixelSceneIntegrator`;
- `bdpt`: global bidirectional splat work through `splatSceneIntegrator`;
- `light_tracing`: global light-path splat work through `splatSceneIntegrator`.

Capability preflight failures are errors. The Engine never substitutes another
Integrator.

## Film

A Camera owns one spectral `camera.Film`. Integrators write wavelength samples
through `FilmAccumulator`; splat-based algorithms use per-pixel synchronization.
After a successful Render, the effective sample count is recorded on the Film
and the controller writes its configured binary Film file. PNG conversion,
display color transforms, merging, resume, and checkpoint orchestration belong
to Studio.

## Code Map

```text
engine/
  main.go                 Engine entry point
  controller/             CLI, job orchestration, parser, and factories
  maths/geometry/         Geometry and SceneSpace
  model/                  Scene, Camera, Film, Material, Object, Shape, Ray
  ray_tracing/            Integrators, transport, scheduling, accumulation
  utils/                  Stateless parsing and numerical helpers
```

Protocol details live in `docs/engine_json_protocol.md`; estimator capability
details live in `docs/integrator.md`.
