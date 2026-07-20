# Ray Tracing

A high-performance, physically-accurate ray tracing engine for optical simulation and photorealistic rendering. Simulate light propagation, reflection, refraction, and complex optical phenomena with scientific precision.

## Rendering Result Demonstration

### Complex geometry rendering

The geometry benchmark evaluates the renderer’s support for heterogeneous geometric representations in a single controlled scene. Objects are arranged in a matrix layout and rendered with shared camera, lighting, exposure, and tone-mapping settings, allowing differences in silhouettes, intersections, normals, shading, and material response to be attributed primarily to geometry rather than scene-specific tuning.

The scene combines standard primitives—spheres, cuboids, cylinders, circles, and triangle meshes—with algebraic surfaces defined by quadratic, cubic, and quartic equations. It also includes procedural and implicit forms, such as polynomial surfaces and gyroid-like scalar fields. This design verifies both the stability of conventional primitive and mesh intersection routines and the renderer’s extended support for arbitrary-order polynomial and implicit geometry.

Each geometry class stresses different parts of the pipeline. Smooth algebraic surfaces test root solving, normal evaluation, and curvature-sensitive highlights; triangle meshes test planar boundaries and composed mesh structure; implicit surfaces stress field evaluation and ray-intersection or ray-marching procedures. Because all objects share the same benchmark stage, the resulting image provides a direct visual comparison across geometry families.

The rendered matrix demonstrates coherent shading and clean silhouettes across simple primitives, meshes, higher-order polynomial surfaces, and implicit forms. Baseline primitives show stable intersections and predictable highlights, while higher-order and implicit objects exhibit complex curvature, cavities, and topology that cannot be captured by fixed primitive libraries alone. Overall, the experiment confirms that the renderer functions as a unified ray-tracing framework for both standard shapes and arbitrary polynomial or implicit geometry.

![geometry-benchmark-matrix.png](docs%2Fassets%2Fgeometry-benchmark-matrix.png)

### Material rendering

![material-benchmark-matrix.png](docs%2Fassets%2Fmaterial-benchmark-matrix.png)


### High-dimensional space rendering

The experiment was designed to evaluate two complementary visualization strategies for a four-dimensional hypercube: physically motivated appearance and topology-oriented segmentation. In both scenes, the same 4D hypercube was placed at the origin with equal extent along all four axes, and it was observed using an identical external 4D orthographic camera. This controlled configuration ensured that differences in the resulting images were caused by the material and rendering model rather than by changes in geometry or viewpoint.

In the first rendering, the hypercube was assigned a Lambertian diffuse material with a light neutral albedo, and illumination was provided by an emissive hypersphere positioned off-center in 4D space. The purpose of this design was to test whether conventional diffuse shading can convey the geometric structure of a 4D object after projection into the image domain. The warm area light introduces gradual intensity variation across visible cells and faces, making the projected form easier to perceive as a coherent solid. Reinhard tone mapping, gamma correction, and a higher sample count were used to produce a stable, visually smooth result.

The second rendering replaces the physically based surface model with a cell-palette material, assigning different solid colors to the eight cubic cells of the hypercube. This version is not intended to simulate realistic illumination; instead, it functions as an analytical visualization. By removing shading ambiguity and encoding cell identity directly through color, the image exposes how the eight constituent volumes contribute to the final 4D projection.

Together, the two results demonstrate a useful trade-off. The Lambertian rendering provides perceptual continuity, depth cues, and a more intuitive sense of global form, but individual cells can be difficult to distinguish. The colored-cell rendering sacrifices physical realism in favor of structural clarity, making adjacency and cell decomposition more explicit. The comparison therefore shows that realistic shading and categorical coloring serve different but complementary roles in 4D visualization: one supports spatial perception, while the other supports topological interpretation.

![4d-hypercube-geometry-focus-centered-combined.png](docs%2Fassets%2F4d-hypercube-geometry-focus-centered-combined.png)

The experiment visualizes a four-dimensional hypercube from an exterior viewpoint and examines how its eight cubic cells appear after projection into a three-dimensional volume. In the figure, the projected object is shown from several viewing angles, together with a bottom-surface projection. The most important feature is that the object does not appear as a single undifferentiated solid. Instead, it is divided into several colored volumetric regions, each corresponding to one cubic cell of the original 4D hypercube.

The visible red, cyan, yellow, and blue regions should be interpreted as projected 3D cells rather than as ordinary colored faces. From the outside, these cells form a cube-like envelope, but their boundaries reveal the higher-dimensional structure behind it. In particular, different colored volumes meet along shared planes, edges, and vertices, showing how the 4D hypercube’s boundary is assembled from cubic components. This is the direct 4D analogue of a 3D cube being assembled from six square faces, except that the hypercube is assembled from eight cubes.

The point-sampled volume representation makes this incidence structure easier to observe. The dotted layers expose the spatial distribution of the projected cells, while the color labels preserve their identity after projection. In the upper and oblique views, the red cell occupies the upper region, the yellow and cyan cells occupy lateral or lower regions, and the blue cell appears on another side of the projected body. As the viewpoint changes across the figure, the apparent proportions of these regions change, but the same cell contacts remain visible. This indicates that the colored partitions are not arbitrary visual artifacts; they encode the adjacency relations among the hypercube’s cubic boundary cells.

The bottom projection further clarifies the geometric result. Seen from below, the projected volume separates into four major colored domains, meeting near a central junction. This view demonstrates that multiple cubic cells can overlap or become adjacent within the same 3D projection, even though they are distinct cells in 4D space. Thus, the experiment shows both the external shape of the projected hypercube and the internal cell decomposition responsible for that shape.

The geometric conclusion is that the boundary of a 4D hypercube is composed of eight cubic cells, and this structure remains observable after projection when cell identity is preserved by color. The image therefore provides a concrete way to inspect the hypercube not merely as a projected outline, but as a volume assembled from eight mutually connected 3D cubes.

![4d-hypercube-geometry-focus-centered2.volume.png](docs%2Fassets%2F4d-hypercube-geometry-focus-centered2.volume.png)

The experiment visualizes the local corner structure of a four-dimensional hypercube from an interior viewpoint. In the image grid, each panel shows the camera looking from inside the hypercube toward one of its vertices. The visible colored regions are not ordinary 2D faces, but projected views of the cubic cells that meet at that vertex. The green, red, cyan, and blue regions form four wall-like volumes converging into the same corner, which is the key geometric observation of the experiment.

This result should be read by analogy with a three-dimensional cube. When standing inside a cube and looking at a corner, one sees three mutually perpendicular square faces meeting at a point. In the four-dimensional case, the corresponding boundary elements are not squares but cubes. Therefore, a vertex of the hypercube is incident to four cubic cells. The image makes this relation visible: around the observed corner, four differently oriented cubic regions appear simultaneously, each occupying a distinct direction in the projected view.

The hyperspherical light source provides an additional geometric cue. Its bright white projections and reflected illumination reveal how these cubic cells occupy different directions around the camera. In several panels, the light appears as circular or elliptical highlights against different colored regions, while the colored cell boundaries remain aligned toward the same corner. This helps distinguish the light source from the hypercube structure itself: the light produces local brightness, but the persistent four-region arrangement is caused by the incidence relation of the hypercube.

The mathematical conclusion is therefore observed directly from the convergence pattern. The corner is not bounded by three surfaces, as in a 3D cube, but by four cubic cells meeting orthogonally in 4D space. The grid presentation reinforces this conclusion by showing the same structure under slightly varying views: although the apparent sizes and brightness of the colored regions change, the four-cell meeting pattern remains stable. This stability indicates that the observed structure is not a rendering artifact, but the expected local geometry of a tesseract vertex.

![4D-2000-grid-20x10.png](docs%2Fassets%2F4D-2000-grid-20x10.png)![4D.png](docs%2Fassets%2F4D.png)

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

See [`docs/engine-json-protocol.md`](docs/engine-json-protocol.md) for the normalized engine protocol, and [`docs/studio-json-protocol.md`](docs/studio-json-protocol.md) for the studio authoring protocol.

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
