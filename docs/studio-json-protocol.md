# Studio JSON Protocol

This file describes the authoring JSON protocol consumed by `studio`.

`studio` is the adaptation layer in front of `engine`. Its input command and
base scene fields intentionally match `engine`, but it may accept authoring
conveniences, analyze them, write normalized intermediate JSON, and then launch
`engine` as a separate executable.

## Command

```bash
npm run studio -- --script path/to/scene.json
go -C studio run . --script ../path/to/scene.json
```

Studio accepts the same render override flags as engine:

```text
--dimension
--camera-index
--threads
--width
--height
--samples
--output-image
--output-film
--resume-film
--exposure
--tone-mapping
--gamma
--spectrum-mode
--wavelength-samples
--working-space
--engine-bin
```

By default, studio runs engine as a separate `go -C engine run .` process so
development runs always use current source. Use `--engine-bin path/to/ray` or
`RAY_ENGINE_BIN=path/to/ray` to select an explicit built engine executable.

`studio` owns authoring and output orchestration. `output_image` and
`resume_film` are accepted by studio but are not emitted to the generated engine
JSON and are not passed to engine. Engine renders Film data only; studio reads
Film files to merge resumed samples and to write images.

Studio accepts repeated `--script` flags and merges them in order before
analysis:

```bash
npm run studio -- --script studio.json --script geometry.json --script render.json
```

Top-level object ids must normally be unique. The exception is authoring-only
containers: if repeated files define the same `id` with `shape: "group"` or the
same `id` with `shape: "array"`, studio merges those containers instead of
raising a duplicate-id error. Later files override container-level fields, while
child objects are merged by id recursively for nested groups and arrays. This
allows a scene grid or group scaffold to live in one file and individual cells
or children to live in other files.

After adaptation, studio writes the generated engine script to:

```text
outputs/intermediate/<source-or-merged>.studio.<hash>.json
```

Then it runs a separate engine process with:

```text
--script outputs/intermediate/<...>.json
```

Engine receives only that single generated script.

### Film, Image, And Resume

Studio keeps `output_film`, `output_image`, and `resume_film` at the command
layer:

```bash
go -C studio run . --script ../examples/scenes/default.json \
  --samples 100 \
  --output-film ../outputs/render.bin \
  --output-image ../outputs/render.png
```

For a resumed render, studio asks engine to render a temporary Film, merges that
temporary Film with `resume_film`, writes the merged Film to `output_film`, and
then writes `output_image` from the merged Film:

```bash
go -C studio run . --script ../examples/scenes/default.json \
  --samples 100 \
  --resume-film ../outputs/render.bin \
  --output-film ../outputs/render.next.bin \
  --output-image ../outputs/render.next.png
```

If `output_film` is omitted, studio uses the engine default Film path. If
`output_image` is omitted, studio uses the default image path.

### Endless Checkpoints

Studio can run indefinitely and save a Film plus image checkpoint every fixed
sample interval:

```bash
go -C studio run . --script ../examples/scenes/default.json \
  --endless \
  --checkpoint-interval 100 \
  --checkpoint-dir ../outputs/checkpoints
```

Each checkpoint renders `checkpoint-interval` new samples, merges them into the
previous accumulated Film, and writes:

```text
iteration-000000000100.bin
iteration-000000000100.png
iteration-000000000200.bin
iteration-000000000200.png
```

Endless mode can resume from a known checkpoint by giving the iteration count
represented by that Film:

```bash
go -C studio run . --script ../examples/scenes/default.json \
  --endless \
  --checkpoint-interval 100 \
  --checkpoint-dir ../outputs/checkpoints \
  --start-iteration 300 \
  --resume-film ../outputs/checkpoints/iteration-000000000300.bin
```

The next checkpoint in that example is `iteration-000000000400.*`. Endless mode
currently supports one render job; scenes with `renders` should be split and run
separately.

The intermediate file contains `_studio` metadata:

```json
{
  "_studio": {
    "version": "0.1",
    "source": ["scene.json"],
    "generated_at": "2026-07-21T00:00:00Z",
    "dimension": 3
  }
}
```

## Base Protocol

Studio input starts from the same top-level shape as engine:

```json
{
  "includes": [],
  "media": {},
  "materials": [],
  "objects": [],
  "cameras": [],
  "render": {},
  "renders": []
}
```

Materials, media, render settings, includes, and multi-render jobs use the same
fields as [`engine-json-protocol.md`](engine-json-protocol.md). Cameras also
support the defaulting conveniences described below.

## Cameras

Studio accepts camera input shaped like engine JSON and can fill omitted default
camera fields.

```json
{
  "type": "3d",
  "position": [-4, 0, 1],
  "direction": [4, 0, -1],
  "up": [0, 0, 1],
  "field_of_view": 60
}
```

For `3d` and `hyperbolic`, studio emits engine canonical JSON with `direction`
and `field_of_views`. If `direction` is omitted, studio uses the default 3D
camera direction.

When a 3D scene omits `cameras`, studio inserts:

```json
{
  "type": "3d",
  "position": [-1.7, 0.1, 0.5],
  "direction": [3.7, -0.1, -0.5],
  "up": [0, 0, 1],
  "field_of_views": [100, 100]
}
```

Studio still accepts authoring-time `field_of_view` plus `aspect_ratio` for
compatibility, then converts them to `field_of_views` before calling engine.
It fills missing `field_of_view` with `100` and missing `aspect_ratio` with `1`
for 3D-like and spherical cameras.

## Group Objects

Studio adds `shape: "group"` as an authoring-only object. Engine never receives
group objects.

```json
{
  "id": "cluster",
  "shape": "group",
  "center": [2, 0, 0],
  "scale": 2,
  "objects": [
    {
      "id": "ball",
      "shape": "sphere",
      "center": [1, 0, 0],
      "r": 0.5,
      "material_id": "glass"
    }
  ]
}
```

Group behavior:

```text
center: inherited placement offset
scale: scalar or vector placement scale
objects: required child object list
id: prefixes child ids as group/child
material_id, medium_id, emission_id, bounds: inherited by children unless overridden
```

Groups may omit `material_id`; only the final flattened renderable objects need
materials. Groups may nest. Studio applies group placement to child geometry and
flattens every group before engine execution. Primitives that cannot represent
non-uniform scaling without changing type, such as circles, cylinders, and STL
meshes, require uniform group scale. In 3D, a non-uniformly scaled sphere is
converted to an equivalent `quadratic equation` ellipsoid.

Placement composition:

```text
world_center = parent.center + parent.scale * local.center
world_scale = parent.scale * local.scale
```

## Array Objects

Studio adds `shape: "array"` as an authoring-only object for sparse 1D, 2D, or
3D cell layouts. Engine never receives array objects. Array placement and field
inheritance match `group`, but each occupied cell adds a regular lattice
offset.

```json
{
  "id": "matrix",
  "shape": "array",
  "origin": [0, 0, 0],
  "delta": [
    [0.56, 0, 0],
    [0, 0.56, 0]
  ],
  "counts": [3, 2],
  "material_id": "white-card",
  "objects": {
    "1,1": [
      {
        "id": "ball",
        "shape": "sphere",
        "center": [0, 0, 0],
        "r": 0.12
      }
    ],
    "3,2": [
      {
        "id": "box",
        "shape": "cuboid",
        "center": [0, 0, 0],
        "size": [0.2, 0.2, 0.2]
      }
    ]
  }
}
```

Array behavior:

```text
origin: required base position of cell 1,1,1
delta: required list of 1 to 3 spacing vectors
counts: required list of 1 to 3 positive integer cell counts
objects: required map from 1-based cell key to child object list
id: prefixes child ids as array/i1-j1/child
material_id, medium_id, emission_id, bounds: inherited by children unless overridden
```

Cell keys are comma-separated, 1-based coordinates. Their dimension must match
`counts`, and each coordinate must be inside the corresponding count. Missing
cells are empty. For a cell key `(i, j, k)`, the cell context center is:

```text
cell_center =
  parent.center
  + parent.scale * origin
  + parent.scale * array.scale * ((i - 1) * delta[0]
                                + (j - 1) * delta[1]
                                + (k - 1) * delta[2])
```

`array.scale` is optional and uses the same scalar-or-vector rules as `group`.
Groups may be nested inside arrays, and arrays may be nested inside groups or
other arrays.

## Current Shape Adapters

Studio currently adapts these authoring forms before writing intermediate JSON.

### Cuboid, Hypercuboid, Hypercube

Authoring input may use `center` or `position` plus `size`:

```json
{
  "shape": "cuboid",
  "position": [0, 0, 0],
  "size": [2, 4, 6]
}
```

Studio outputs engine-native `pmin` and `pmax`, applying inherited group
placement. For `hypercube`, studio validates equal local side lengths and emits
`shape: "cuboid"` because engine does not parse `hypercube`.

### Triangle

Authoring input may provide an object-level `center` offset:

```json
{
  "shape": "triangle",
  "center": [1, 0, 0],
  "p1": [0, 0, 0],
  "p2": [1, 0, 0],
  "p3": [0, 1, 0]
}
```

Studio bakes the offset and group placement into `p1`, `p2`, and `p3`, then
removes `center`.

### Quadratic Equation

Authoring input may use local `center` and `scale`:

```json
{
  "shape": "quadratic equation",
  "a": [1, 0, 0, 0, 1, 0, 0, 0, 1],
  "b": [0, 0, 0],
  "c": -1,
  "center": [2, 0, 0],
  "scale": 3
}
```

Studio bakes the placement into world-space `a`, `b`, and `c`, then removes
`center` and `scale`.

### Cubic Equation

Authoring input may use local `center`, `scale`, and sparse `A` coefficients.
Studio resolves coefficients, bakes placement into world-space `a`, and removes
`A`, `center`, and `scale`.

### Four-Order Equation

Authoring input may use local `center`, `scale`, `basis`, and sparse `A`
coefficients. Studio resolves coefficients, bakes placement and basis into
world-space `a`, and removes `A`, `center`, `scale`, and `basis` before calling
engine.

### Polynomial Surface

Authoring input may use local `center`, `scale`, and `basis` fields. Studio
combines them into engine-native `transform`, a 4 x 4 world-to-local homogeneous
matrix, then removes `center`, `scale`, and `basis` before calling engine. If an
author already provides `transform`, studio preserves it and composes group
placement into it.

### Bounds

Any object may use authoring bounds with `center` or `position` plus `size`:

```json
{
  "bounds": {
    "center": [0, 0, 0],
    "size": [2, 2, 2]
  }
}
```

Studio emits engine-native bounds:

```json
{
  "bounds": {
    "pmin": [-1, -1, -1],
    "pmax": [1, 1, 1]
  }
}
```

## Pass-Through Shapes

Shapes without a studio adapter are passed through after id inheritance and
group field inheritance and bounds normalization. This includes:

```text
sphere
circle
cylinder
finite cylinder
implicit equation
stl
klein_bottle
```

For pass-through shapes, the engine protocol remains authoritative.

## Intermediate JSON Contract

The intermediate file should be valid engine JSON:

```text
no shape: "group"
adapted ids after group prefixing
inherited material/media/emission/bounds fields applied
adapted cuboid/triangle/quadratic/cubic/four-order/polynomial-surface/bounds fields normalized
_studio metadata included for traceability
```

Studio keeps the intermediate file on disk so the generated engine input can be
inspected, tested, or replayed directly with `npm run ray`.

## Example

```bash
cmd /c examples\scenes\studio-geometry-benchmark-matrix\run_example.cmd --width 1 --height 1 --samples 1
```

That example copies the geometry benchmark matrix workflow but routes it through
studio first, then lets engine render the generated intermediate JSON.
