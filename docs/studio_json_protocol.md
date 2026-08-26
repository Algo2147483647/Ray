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
--threads
--width
--height
--samples
--output-image
--output-film
--resume-film
--exposure
--tone-mapping
--tanh-omega
--gamma
--spectrum-mode
--wavelength-samples
--color-space
--pixel-window
```

Studio can also convert an existing binary Film directly to PNG without reading
a scene or launching Engine:

```bash
go -C studio run . --input-film ../outputs/render.bin \
  --output-image ../outputs/render.png \
  --exposure 1.25 \
  --tone-mapping aces \
  --gamma 2.2 \
  --color-space linear_srgb
```

`--output-image` is optional in this mode. By default Studio replaces the
input `.bin` extension with `.png`. Only the six image-output options shown
above can be combined with `--input-film`; scene and rendering options are
rejected because this mode performs post-processing only.

Studio also provides a spectrum-preserving highlight mode:

```bash
go -C studio run . --input-film ../outputs/render.bin \
  --output-image ../outputs/render.spectral-tanh.png \
  --tone-mapping spectral_tanh \
  --tanh-omega 1 \
  --gamma 2.2
```

For each pixel this mode sums all spectral bins to obtain brightness `x` and
maps it to the normalized range with `tanh(tanh_omega * exposure * x)`. The
resulting brightness is applied to every bin through one shared multiplier, so
the ratios between the physical spectral channels are unchanged. After
conversion to RGB, one additional
shared scale is used if a positive RGB component exceeds 1; this avoids hue
changes caused by clipping individual high channels. Negative RGB components
can still occur for spectra outside the selected RGB gamut and cannot be
represented exactly by an RGB PNG. `tanh_omega` defaults to 1 and must be
positive.

`studio` owns authoring and output orchestration. `output_image` and
`resume_film`, exposure, tone mapping, gamma, and `color_space` are accepted by
studio but are not emitted to the generated engine JSON and are not passed to
engine. Engine renders a physical spectral Film only. Studio receives a newly
rendered Film in memory, then saves it and creates images without rereading it.
Only an explicitly requested resume Film is read from disk.

Studio accepts repeated `--script` flags and merges them in order before
analysis:

```bash
npm run studio -- --script studio.json --script geometry.json --script render.json
```

Top-level object ids must normally be unique. The exception is authoring-only
containers: if repeated files define the same `id` with `shape: "group"` or the
same `id` with `shape: "array"`, studio merges those containers instead of
raising a duplicate-id error. Container parameters are combined directly: a
fragment may add a missing field or repeat the same value, but conflicting values
are errors. Underscore-prefixed authoring metadata such as `_comment` is ignored
for conflict purposes. Child objects are merged by id recursively for nested
groups and arrays. A repeated object id must also have the same parent path;
placing that id beneath different groups or array cells is an error. A top-level
Group or Array has no bound parent: when one nested container with the same id
exists, the top-level fragment is merged into and bound at that nested location.
If no such location exists, it remains a normal top-level scene object. This
lets model files stay independent of scene assembly Group ids. Binding is unique; two
different nested parents for one id are still an error. Together these rules
allow scene scaffolds and their children to live in separate files without
order-dependent overrides.

After adaptation, studio writes the generated engine script to:

```text
outputs/intermediate/<source-or-merged>.studio.<hash>.json
```

Then it invokes the Engine controller with:

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

If `output_film` is omitted, studio uses its command-layer default Film path. If
`output_image` is omitted, studio uses the default image path.

### Films

Films own the image grid, camera association, Film path, and image presentation
options. A render selects one with `film_id`; Studio emits one canonical Engine
render target containing `camera_id`, `film`, and `output`. Cameras are emitted
once and can be shared by any number of render targets; Engine has no top-level
`films` or `film_id`.

```json
{
	"dimension": 3,
  "cameras": [{ "id": "main", "position": [0, 0, 0] }],
  "films": [{
    "id": "main-film",
    "camera_id": "main",
    "shape": [960, 800],
    "spectral_bin_count": 128,
    "output_image": "../outputs/geometry-benchmark-matrix.png",
    "output_film": "../outputs/geometry-benchmark-matrix.bin",
    "exposure": 1,
    "tone_mapping": "spectral_tanh",
    "tanh_omega": 1,
    "gamma": 2.2,
    "color_space": "linear_srgb"
  }],
  "render": { "samples": 100, "film_id": "main-film" }
}
```

`dimension` belongs to the Scene and applies to every render job.
`render.dimension` and `renders[].dimension` are invalid; generated Engine JSON
uses only the top-level field.

`spectral_bin_count` controls the number of wavelength bins stored in the
scene-linear Film over the 380–750 nm range. It defaults to 64 and may be set
from 1 through 4096. Increasing it raises Film memory and checkpoint size
linearly; it does not change `render.wavelength_samples`, which controls Monte
Carlo wavelength samples per render sample.

`render.width`, `render.height`, `camera_index`, output fields, display fields,
and `pixel_windows` are not part of the Studio source format.

### Pixel Windows

Studio accepts engine-compatible `pixel_windows` in a Film definition and passes
them through to the generated engine JSON. The output Film and image keep
their configured dimensions; pixels outside the requested windows remain zero.

Ranges are half-open: `min` is included and `max` is excluded.

```json
{
  "films": [{
    "id": "main-film",
    "camera_id": "main",
    "shape": [800, 800],
    "pixel_windows": [
      { "min": [100, 600], "max": [150, 650] }
    ]
  }],
  "render": { "film_id": "main-film" }
}
```

The CLI override is repeatable and accepts `:` or `-` separators:

```bash
go -C studio run . --script ../scene.json --pixel-window 100:150,600:650
go -C studio run . --script ../scene.json --pixel-window 100-150,600-650
```

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

The intermediate file contains only canonical Engine protocol fields. Studio
keeps source and generation metadata in its own process state; it does not add
Studio-only keys to the Engine input:

```json
{
	"dimension": 3
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
  "films": [],
  "render": { "film_id": "main-film" },
  "renders": []
}
```

Materials, media, objects, and includes follow the Engine protocol. Studio
Camera, Film, Render, and multi-render job fields follow the stricter authoring
model documented here; Studio converts them to canonical Engine fields.

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
  "basis": [[1, 0, 0], [0, 0, -1], [0, 1, 0]],
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
basis: optional orthonormal local-to-parent rotation matrix (rows are world axes)
objects: required child object list
id: prefixes child ids as group/child
material_id, medium_id, emission_id, bounds: inherited by children unless overridden
```

Groups may omit `material_id`; only the final flattened renderable objects need
materials. Groups may nest. Studio applies group placement to child geometry and
flattens every group before engine execution. Primitives that cannot represent
non-uniform scaling without changing type, such as circles, cylinders, and STL
meshes, require uniform group scale. In 3D, a non-uniformly scaled sphere is
converted to an equivalent degree-two `polynomial` with a world-to-local transform.

Placement composition:

```text
world_center = parent.center + parent.basis * (parent.scale * local.center)
world_scale = parent.scale * local.scale
world_basis = parent.basis * local.basis
```

Rotated groups support triangles directly and rotate the axes of circles and
finite cylinders. Quadrilaterals are expanded into triangles before placement,
so they support the same rotations as triangles. Spheres are rotation invariant.
Axis-aligned cuboids cannot represent a rotated result and are rejected; author
an oriented box as triangles or quadrilaterals.
Combining a rotated nested group with a non-uniform parent scale is rejected to
avoid silently introducing shear.

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

### Quadrilateral

`quadrilateral` is a Studio-only convenience shape. Its four vertices must be
listed in boundary order:

```json
{
  "id": "panel",
  "shape": "quadrilateral",
  "p1": [0, 0, 0],
  "p2": [1, 0, 0],
  "p3": [1, 1, 0],
  "p4": [0, 1, 0]
}
```

Studio splits the shape along the `p1`-`p3` diagonal into triangles
`(p1, p2, p3)` and `(p3, p4, p1)`. They are emitted with ids
`panel/triangle-1` and `panel/triangle-2`. Object fields and inherited group
fields apply to both triangles, and Engine never receives a `quadrilateral`
shape.

### Polynomial

Studio also provides two compact authoring forms that compile to `polynomial`.
A plane uses `normal` and either `point` or `offset`, where `offset` means
$n\cdot x=\mathtt{offset}$:

```json
{
  "shape": "plane",
  "normal": [0, 0, 1],
  "point": [0, 0, 2]
}
```

A quadratic surface uses the strict symmetric-matrix form
$x^TQx+l^Tx+c=0$. `shape` may be `quadratic`, `quadric`, or
`quadratic surface`:

```json
{
  "shape": "quadratic",
  "matrix": [[1, 0, 0], [0, 1, 0], [0, 0, 1]],
  "linear": [0, 0, 0],
  "constant": -1,
  "center": [2, 0, 0]
}
```

Both forms support Studio group placement and are absent from intermediate
Engine JSON.

All algebraic surfaces use one sparse authoring form:

```json
{
  "shape": "polynomial",
  "degree": 2,
  "terms": [
    { "exponents": [2, 0, 0], "coefficient": 1 },
    { "exponents": [0, 2, 0], "coefficient": 1 },
    { "exponents": [0, 0, 2], "coefficient": 1 },
    { "exponents": [0, 0, 0], "coefficient": -1 }
  ],
  "center": [2, 0, 0],
  "scale": 3
}
```

Studio converts `center`, `scale`, and `basis` into the canonical world-to-local
`transform`, then removes those authoring placement fields. Engine receives only
`degree`, `terms`, and `transform`. Degrees one and two use direct cached
kernels; degrees three and four use specialized real-root solvers. The public
Engine shape does not expose separate variants.

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

## Native Shapes

Studio applies placement and emits canonical Engine discriminators for native
shapes. Authoring aliases `hypersphere`, `hypercuboid`, and `finite cylinder`
become `sphere`, `cuboid`, and `cylinder` respectively. Native shapes include:

```text
sphere
circle
cylinder
implicit equation
stl
klein_bottle
```

For pass-through shapes, the engine protocol remains authoritative.

## Intermediate JSON Contract

The intermediate file should be valid engine JSON:

```text
no shape: "group"
only canonical Engine shape, surface, geometry, and integrator names
adapted ids after group prefixing
inherited material/media/emission/bounds fields applied
adapted cuboid/triangle/plane/quadratic/polynomial/bounds fields normalized
no Studio-only metadata keys
```

Studio keeps the intermediate file on disk so the generated engine input can be
inspected, tested, or replayed directly with `npm run ray`.

## Example

```bash
cmd /c examples\scenes\studio-geometry-benchmark-matrix\run_example.cmd --width 1 --height 1 --samples 1
```

That example copies the geometry benchmark matrix workflow but routes it through
studio first, then lets engine render the generated intermediate JSON.
