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
coefficient encodings. Each Camera describes an imaging model. Each Render
binds a Camera, Film, output destination, Integrator, and resolved execution
counts as one immutable `ray_tracing.RenderJob`.
`parser.ShapeKind` plus `objectDefinitionFactories` is the sole shape
discriminator registry. Factories consume the decoded `ObjectDefinition`
struct and do not maintain another list of shape strings.

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
`controller.ResolveRenderJob` is the only render-default and Camera-binding
boundary. Missing execution counts receive defaults; explicitly non-positive
counts are rejected by the parser rather than reinterpreted as missing.
`ray_tracing.NewHandler` receives the SceneSpace explicitly and creates Rays for
that space. `NewSceneIntegrator` selects exactly the requested algorithm:

- `path`: tiled camera-path work through `pixelSceneIntegrator`;
- `bdpt`: global bidirectional splat work through `splatSceneIntegrator`;
- `light_tracing`: global light-path splat work through `splatSceneIntegrator`.

Capability preflight failures are errors. The Engine never substitutes another
Integrator. Each Integrator performs one `Prepare` call, returns immutable
prepared state, and receives that state in `Run`; BDPT capability validation and
light collection therefore happen once per render.

A Material surface is one `bxdf.Scattering` value. Atomic scattering models,
weighted mixtures, and procedural mixtures implement the same interface
directly; there is no BSDF/BxDF interface split or single-model wrapper. A Ray
owns one `PathState{Throughput, Radiance, Wavelength}`. Throughput and radiance
are scalar powers at that wavelength; authored RGB remains behind material
`SpectralParameter` boundaries and is resolved before entering transport.

## Film

A `RenderJob` binds a reusable Camera to one spectral `film.Film`, its output
destination, and resolved execution settings. Integrators write wavelength samples
through a private Film accumulator; splat-based algorithms use per-pixel synchronization.
After a successful Render, the effective sample count is recorded on the Film
and the controller writes its configured binary Film file. PNG conversion,
display color transforms, merging, resume, and checkpoint orchestration belong
to Studio. Film storage, pixel windows, authored `film.Spec`, and the binary
codec live in `model/film`; the runtime bin count is always
`len(Film.SpectralBins)` and is not stored separately.

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
