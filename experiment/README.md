# Ray Experiment Suite

This directory is the repository's experiment layer. It turns engine features into
reproducible scenes, generated constructions, visual comparison matrices, and
numerically checked optical tests.

The experiments answer four different kinds of question:

1. **Coverage:** which engine geometries and materials can be instantiated and
   rendered in a common scene?
2. **Construction:** can higher-dimensional and non-Euclidean objects be encoded
   from their mathematical definitions?
3. **Behavior:** do transport effects such as dispersion and absorption follow the
   intended ordering and trend?
4. **Presentation:** can the same underlying experiment produce a useful diagnostic
   view and a readable final image?

For the complete methodology, equations, configuration inventory, observed results,
limitations, and recommended next steps, see
[EXPERIMENT_REPORT.md](EXPERIMENT_REPORT.md).

## Experiment catalog

| Family | Experiment | Primary question | Main inputs | Existing result | Evidence level |
|---|---|---|---|---|---|
| Geometry | [Geometry benchmark matrix](geometric_object/geometry-benchmark-matrix/) | Can a broad cross-section of 3D shapes render under a shared camera, room, and material library? | Composed JSON fragments | [Matrix image](geometric_object/geometry-benchmark-matrix/results/geometry-benchmark-matrix.png) | Visual coverage |
| Geometric spaces | [4D scenes](geometric_spaces/4d/) | Can rank-3 films expose projections and slices of 4D geometry? | Standalone JSON scenes | Four stored visualizations | Visual/prototype |
| Geometric spaces | [Hyperbolic constructions](geometric_spaces/non-euclidean/) | Can Klein-model reflection, honeycomb growth, distances, and geodesic constructions be generated consistently? | Python generators and JSON scenes | [Showcase image](geometric_spaces/non-euclidean/results/hyperbolic_showcase_art.png) | Generator invariants plus visual evidence |
| Material | [Material benchmark matrix](material/material-benchmark-matrix/) | How do engine materials differ under fixed geometry and lighting? | Standalone JSON scene | [Matrix image](material/material-benchmark-matrix/results/material-benchmark-matrix.png) | Visual comparison |
| Material/transport | [Triangular-prism dispersion](material/triangular_prism_dispersion/) | Does a dispersive prism separate spectral bands, and does wavelength-dependent absorption attenuate them in the expected order? | Composed JSON fragments, batch scripts, spectral probe | [Beauty image](material/triangular_prism_dispersion/results/Triangular_Prism_Dispersion.png) | Automated numerical verification plus visual evidence |

Evidence levels are deliberately not interchangeable. A stored PNG demonstrates a
rendered outcome; it does not by itself prove a numerical invariant. The prism test
is currently the only experiment with an automated pass/fail analysis of rendered
film data.

## Prerequisites

- Go, for the engine and Studio command-line programs.
- Python 3, for the hyperbolic scene generators.
- Windows `cmd.exe`, if using the supplied `.cmd` wrappers.
- Sufficient memory and disk space for high-sample or rank-3 films. Volume films can
  be substantially larger than ordinary images.

From the repository root, the main commands are:

```powershell
npm run ray:build
npm run ray:test
npm run studio -- --help
```

`npm run studio -- ...` delegates scene composition and rendering to Studio. Direct
engine utilities are run from `engine`, for example:

```powershell
Set-Location engine
go run ./cmd/spectral_film_probe --help
```

## Running the experiments

### Geometry benchmark matrix

The wrapper composes the room, camera/render settings, material library, and six
geometry-row fragments:

```powershell
Set-Location experiment/geometric_object/geometry-benchmark-matrix
cmd /c run.cmd
```

For an unbounded accumulation run with periodic checkpoints:

```powershell
cmd /c run_endless.cmd
```

The matrix is a coverage and visual-regression scene, not a timed performance
benchmark. Its detailed 42-cell catalog is documented in the experiment's own
[README](geometric_object/geometry-benchmark-matrix/README.md).

### 4D scenes

Render a selected scene explicitly with Studio. For example:

```powershell
npm run studio -- --script experiment/geometric_spaces/4d/4d-hypercube-geometry-focus.json
```

These scenes use a three-dimensional film to record a projection or slice field of a
four-dimensional scene. Inspect rank-3 film output with the repository's
[`film-volume-viewer`](../tools/film-volume-viewer/README.md). The stored PNGs in the
results directory are presentation exports rather than raw volume films.

### Hyperbolic honeycomb

Generate a finite breadth-first subset of the regular cubic honeycomb and write it
back into this experiment directory:

```powershell
python "experiment/geometric_spaces/non-euclidean/hyperbolic honeycomb.py" --cells 25 --max-layer 3 --out experiment/geometric_spaces/non-euclidean/hyperbolic.json
npm run studio -- --script experiment/geometric_spaces/non-euclidean/hyperbolic.json
```

The generator's default output path points into `examples/scenes`, so pass `--out`
when the intention is to update the experiment copy.

### Hyperbolic showcase

Regenerate the five-cell orbit, metric ladder, and geodesic triangle:

```powershell
python experiment/geometric_spaces/non-euclidean/hyperbolic_showcase.py --out experiment/geometric_spaces/non-euclidean/hyperbolic_showcase_art.json
cmd /c experiment\geometric_spaces\non-euclidean\run.cmd
```

Generation itself checks orbit closure, the target dihedral angle, equal hyperbolic
ladder spacing, triangle angle defect, and the Klein-ball safety radius.

### Material benchmark matrix

```powershell
npm run studio -- --script experiment/material/material-benchmark-matrix/material-benchmark-matrix.json
```

All samples use a shared sphere geometry and room. This isolates material response
more effectively than comparing unrelated objects, while repeated materials at
different positions expose lighting and view dependence.

### Triangular-prism dispersion

Create the final absorbing-prism render:

```powershell
Set-Location experiment/material/triangular_prism_dispersion
cmd /c run.cmd
```

Run the control and absorbing variants and analyze their spectral films:

```powershell
cmd /c verify.cmd
```

The verifier checks blue/green/red centroid ordering and separation. With a control
film supplied, it also checks reference agreement and the expected absorption ratio
ordering. The systematic report records the latest observed values.

## Reproducibility contract

Every new or updated experiment should preserve the following information:

- the question or hypothesis;
- the exact scene and generator inputs;
- the engine dimension, camera, integrator, spectrum mode, sample count, and film
  shape;
- the exact command used to generate and render the scene;
- quantitative acceptance criteria where a numerical claim is made;
- result artifacts, including raw film data when later analysis requires it;
- engine revision, generator revision, random seed or sampler settings when exposed,
  date, hardware, elapsed time, and peak memory;
- a clear distinction between a baseline, a diagnostic export, and an illustrative
  image.

The current suite predates this full contract. In particular, some stored images
differ in resolution or naming from their present JSON configurations, and no common
manifest ties every artifact to a commit and command line. Those gaps are cataloged
in the report rather than hidden.

## Result policy

Do not replace a result image merely because a new render looks better. First decide
whether it is:

- a **baseline update**, which must be reviewed for intended behavior changes;
- a **diagnostic result**, which should retain analysis-friendly settings;
- a **presentation result**, which may use tone mapping and layout overrides; or
- a **new sample**, which should coexist with the baseline until compared.

For stochastic results, compare robust statistics or toleranced image metrics rather
than requiring byte-identical PNG files.

## Known limitations

- The two benchmark matrices are visual coverage experiments; they currently record
  no timing, memory, convergence, or error metrics.
- Several image resolutions do not match the current scene defaults, which indicates
  historical overrides or incomplete provenance.
- The 4D result export workflow is not scripted inside the experiment directory.
- Most raw films are not stored beside the result images, limiting retrospective
  numerical analysis.
- Random seeds are not recorded, so Monte Carlo rerenders need not be pixel-identical.
- Generated Python bytecode is present in a few directories and is not experimental
  source.
- The prism subdirectory's older README contains encoding damage and stale narrative;
  use this README and the systematic report as the current overview.

## Adding an experiment

Use a self-contained directory with this minimum layout:

```text
experiment-name/
|-- README.md
|-- scene.json
|-- generate.py          # optional
|-- run.cmd              # optional convenience wrapper
|-- verify.cmd           # recommended when a numerical claim is possible
`-- results/
    |-- manifest.json
    `-- representative.png
```

Prefer small composable JSON fragments when variants share most of a scene. Keep
beauty-only lighting and tone mapping separate from diagnostic configurations, as the
prism experiment already does.
