#!/usr/bin/env python3
"""
Generate a scene of N cells of the {4,3,5} hyperbolic honeycomb in the Klein
ball model, with each cell rendered as a closed box of 6 quad faces (12
triangles). The honeycomb is built by BFS: starting from a base cube whose
6 face-planes are the Klein-planes x=+/-a, y=+/-a, z=+/-a with

    a^2 = sqrt(5) - 2     (so the H^3 dihedral angle of the cube is 2*pi/5)

we reflect that cube across each of its 6 face-planes to get neighbours. Five
cubes really do meet along each edge in H^3 (impossible in R^3 where only 4
meet), so the corridor formed by the spine of this honeycomb is a genuine
non-euclidean structure -- not Klein coordinates of euclidean geometry.

The output JSON drops straight into the existing ray-tracing engine with
geometry.type = klein and a hyperbolic camera.

Math reference: hyperbolic reflection across Klein-plane (n unit, c in (-1,1)):

    x' = (x - k * n) / (1 - k * c)        with  k = 2 (n.x - c) / (1 - c^2)

which is the projection back to the Klein chart of the SO(3,1) reflection on
the hyperboloid {(x, t) : t^2 - |x|^2 = 1, t > 0} through the spacelike
hyperplane with Minkowski normal (n, c).
"""

import argparse
import json
import math
from dataclasses import dataclass

import os
import sys


# ---- {4,3,5} hyperbolic cube geometry --------------------------------------
SQRT5 = math.sqrt(5.0)
A2 = SQRT5 - 2.0                      # a^2
A = math.sqrt(A2)                     # cube half-edge in Klein coords ~= 0.4859

# 8 cube vertices (canonical order: bit i of index k decides sign of axis i)
BASE_VERTICES = [
    (A if (k & 1) else -A,
     A if (k & 2) else -A,
     A if (k & 4) else -A)
    for k in range(8)
]

# 6 cube faces; each is a list of 4 vertex indices in CCW order *as seen from
# OUTSIDE the cube*. We will recompute face planes from vertex coords later,
# choosing the outward sign by comparing to the cell center.
BASE_FACES = [
    # axis=0 (x), sign=-1: vertices with k & 1 == 0
    [0, 4, 6, 2],
    # axis=0 (x), sign=+1: vertices with k & 1 == 1
    [1, 3, 7, 5],
    # axis=1 (y), sign=-1
    [0, 1, 5, 4],
    # axis=1 (y), sign=+1
    [2, 6, 7, 3],
    # axis=2 (z), sign=-1
    [0, 2, 3, 1],
    # axis=2 (z), sign=+1
    [4, 5, 7, 6],
]

# 12 cube edges as vertex-index pairs. We render each edge as a thin
# finite cylinder so the cells form a wireframe corridor (faces are not
# drawn, letting the camera see through into neighbour cells).
BASE_EDGES = [
    # 4 edges along x at fixed (y,z)
    (0, 1), (2, 3), (4, 5), (6, 7),
    # 4 edges along y at fixed (x,z)
    (0, 2), (1, 3), (4, 6), (5, 7),
    # 4 edges along z at fixed (x,y)
    (0, 4), (1, 5), (2, 6), (3, 7),
]


# ---- Klein-model reflection ------------------------------------------------
def reflect_point(p, n, c):
    """Hyperbolic reflection of Klein point p across the Klein-plane n.x = c."""
    nx = n[0] * p[0] + n[1] * p[1] + n[2] * p[2]
    denom_n = 1.0 - c * c
    if denom_n <= 0:
        raise ValueError("plane outside Klein ball")
    k = 2.0 * (nx - c) / denom_n
    den = 1.0 - k * c
    return (
        (p[0] - k * n[0]) / den,
        (p[1] - k * n[1]) / den,
        (p[2] - k * n[2]) / den,
    )


# ---- Cell representation ---------------------------------------------------
@dataclass
class Cell:
    verts: list      # list[tuple3] -- 8 Klein-coord vertices
    layer: int       # BFS depth from base cell
    cell_id: int


def cell_center(c: Cell):
    sx = sy = sz = 0.0
    for v in c.verts:
        sx += v[0]; sy += v[1]; sz += v[2]
    n = float(len(c.verts))
    return (sx / n, sy / n, sz / n)


def face_plane(cell: Cell, face_idx_quad):
    """Return (n, c) Klein-plane of a face, with n pointing OUTWARD from cell.
    n is unit euclidean. Returns also the 4 vertex coords in CCW (as seen
    from outside)."""
    v = [cell.verts[i] for i in face_idx_quad]
    # Two edges in the face plane
    e1 = (v[1][0] - v[0][0], v[1][1] - v[0][1], v[1][2] - v[0][2])
    e2 = (v[2][0] - v[0][0], v[2][1] - v[0][1], v[2][2] - v[0][2])
    # Euclidean cross product e1 x e2 (normal in euclidean R^3 -- and since
    # the Klein-plane equation is purely affine, this same n parameterises
    # the H^3-totally-geodesic plane).
    n = (
        e1[1] * e2[2] - e1[2] * e2[1],
        e1[2] * e2[0] - e1[0] * e2[2],
        e1[0] * e2[1] - e1[1] * e2[0],
    )
    nn = math.sqrt(n[0] ** 2 + n[1] ** 2 + n[2] ** 2)
    if nn == 0:
        raise ValueError("degenerate face")
    n = (n[0] / nn, n[1] / nn, n[2] / nn)
    c = n[0] * v[0][0] + n[1] * v[0][1] + n[2] * v[0][2]
    # Ensure outward: cell center should give negative signed distance.
    cc = cell_center(cell)
    if n[0] * cc[0] + n[1] * cc[1] + n[2] * cc[2] - c > 0:
        n = (-n[0], -n[1], -n[2])
        c = -c
    return n, c, v


def reflect_cell(cell: Cell, n, c, new_id, layer):
    new_verts = [reflect_point(v, n, c) for v in cell.verts]
    return Cell(verts=new_verts, layer=layer, cell_id=new_id)


# ---- Cell identity (for dedup) ---------------------------------------------
def cell_key(cell: Cell):
    """Hash key: rounded sorted vertex coordinates."""
    return tuple(sorted(
        tuple(round(x, 6) for x in v) for v in cell.verts
    ))


# ---- BFS ------------------------------------------------------------------
def grow_honeycomb(max_cells, max_layer):
    base = Cell(verts=list(BASE_VERTICES), layer=0, cell_id=0)
    cells = [base]
    seen = {cell_key(base): 0}
    queue = [base]
    while queue and len(cells) < max_cells:
        cur = queue.pop(0)
        if cur.layer >= max_layer:
            continue
        for fi, face_quad in enumerate(BASE_FACES):
            # Note: BASE_FACES indexing only valid for the original cube
            # vertex ordering, which we preserve by construction (we never
            # reorder verts when reflecting -- vertex i in cell C is the
            # reflected image of vertex i in C's parent).
            n, c, _ = face_plane(cur, face_quad)
            try:
                neigh = reflect_cell(cur, n, c, new_id=len(cells),
                                     layer=cur.layer + 1)
            except (ValueError, ZeroDivisionError):
                continue
            # All vertices must stay strictly inside the Klein ball (their
            # hyperbolic existence is guaranteed; failure here is a numerical
            # blow-up near the boundary).
            ok = all(v[0] ** 2 + v[1] ** 2 + v[2] ** 2 < 0.999 for v in neigh.verts)
            if not ok:
                continue
            key = cell_key(neigh)
            if key in seen:
                continue
            seen[key] = neigh.cell_id
            cells.append(neigh)
            queue.append(neigh)
            if len(cells) >= max_cells:
                break
    return cells


# ---- JSON emission ---------------------------------------------------------
# Per-BFS-layer edge colors. Emissive (constant emission) rather than lambert
# so the wireframe glows in its own color regardless of where the corridor
# lights are; this makes the {4,3,5} layer structure pop visually even after
# heavy hyperbolic fog absorption on distant layers.
LAYER_EDGE_EMISSION = [
    # (rgb scalar emission). Base cell warm white, then layer-rings. Magnitudes
    # tuned so deep layers survive hyperbolic fog and the closest cells don't
    # bloom out.
    (1.50, 1.30, 0.90),  # layer 0 -- base cube (warm white)
    (2.20, 0.55, 0.35),  # layer 1 -- 6 face-neighbours (warm red)
    (0.55, 1.10, 2.40),  # layer 2 -- 18 second-shell (cool cyan/blue)
    (1.80, 1.40, 0.40),  # layer 3 (if extended)
    (0.55, 1.80, 0.65),  # layer 4
    (1.30, 0.55, 1.50),  # layer 5+
]


def edge_emission_for_layer(layer):
    return LAYER_EDGE_EMISSION[min(layer, len(LAYER_EDGE_EMISSION) - 1)]


def emit_scene(cells, output_path, image_path, film_path,
               camera_pos, camera_lookat, fov, samples, width, height,
               sigma_a, max_arc):
    materials = []
    layers_used = sorted({c.layer for c in cells})
    for layer in layers_used:
        r, g, b = edge_emission_for_layer(layer)
        materials.append({
            "id": f"cell_edge_L{layer}",
            "emission": {
                "type": "constant",
                "radiance": {"type": "rgb", "value": [r, g, b]},
            },
        })

    # ---- WIREFRAME ----
    # Faces are NOT drawn. Only edges, as thin finite cylinders. The cylinder
    # axis follows the CHORD between the two Klein-coord vertices -- which IS
    # the H^3 geodesic between them in the Klein model. The cylinder radius
    # is constant in Klein euclidean coordinates, so distant cells' edges
    # appear visibly thinner (a fair "depth queue" matching hyperbolic
    # foreshortening). We deduplicate edges across cells: edges of layer-k
    # cells take the color of the lower-layer neighbour they came from, so
    # the corridor shows clean layer rings.
    objects = []
    edge_seen = {}  # key: (sorted rounded endpoints) -> layer
    edge_radius_base = A * 0.045
    for cell in cells:
        edge_mat = f"cell_edge_L{cell.layer}"
        for ei, (i, j) in enumerate(BASE_EDGES):
            p = cell.verts[i]
            q = cell.verts[j]
            key = tuple(sorted([
                tuple(round(x, 6) for x in p),
                tuple(round(x, 6) for x in q),
            ]))
            if key in edge_seen:
                continue
            edge_seen[key] = cell.layer
            # Edge midpoint and chord vector
            cx = 0.5 * (p[0] + q[0])
            cy = 0.5 * (p[1] + q[1])
            cz = 0.5 * (p[2] + q[2])
            ax = q[0] - p[0]
            ay = q[1] - p[1]
            az = q[2] - p[2]
            length = math.sqrt(ax * ax + ay * ay + az * az)
            if length < 1e-9:
                continue
            objects.append({
                "id": f"cell_{cell.cell_id}_edge_{ei}",
                "material_id": edge_mat,
                "shape": "finite cylinder",
                "center": [cx, cy, cz],
                "axis": [ax, ay, az],
                "r": edge_radius_base,
                "height": length,
            })

    # No discrete lamp -- the edge cylinders themselves are emissive, so the
    # cell wireframe is self-lit.

    scene = {
        "_comment_C_overview": (
            "PLAN C - {4,3,5} hyperbolic honeycomb corridor, "
            f"{len(cells)} cube cells generated by BFS. Each cube is a regular "
            "H^3 cube with H^3 dihedral angle 2*pi/5 = 72deg (5 cubes meet at "
            "each edge -- impossible in euclidean R^3 where only 4 meet). The "
            "neighbour cells are produced by SO(3,1) reflections of the base "
            "cube through its 6 face-planes; each cube is hyperbolic-congruent "
            "to every other cube even though Klein coordinates make distant "
            "cells look exponentially smaller."
        ),
        "render": {
            "dimension": 3,
            "samples": samples,
            "thread_num": 0,
            "width": width,
            "height": height,
            "camera_index": 0,
            "output_image": image_path,
            "output_film": film_path,
            "exposure": 1.0,
            "tone_mapping": "aces",
            "gamma": 2.2,
            "spectrum_mode": "rgb",
            "color_space": "linear_srgb",
        },
        "geometry": {"type": "klein", "max_arc": max_arc},
        "media": {
            "air": {
                "type": "homogeneous",
                "sigma_a": list(sigma_a),
            },
        },
        "cameras": [
            {
                "id": "main",
                "type": "hyperbolic",
                "position": list(camera_pos),
                "look_at": list(camera_lookat),
                "up": [0.0, 0.0, 1.0],
                "field_of_view": fov,
                "aspect_ratio": width / height,
            }
        ],
        "materials": materials,
        "objects": objects,
    }

    with open(output_path, "w") as f:
        json.dump(scene, f, indent=2)
    return scene


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--cells", type=int, default=25)
    ap.add_argument("--max-layer", type=int, default=3)
    ap.add_argument("--out", default="examples/scenes/hyperbolic_honeycomb.json")
    ap.add_argument("--image", default="../outputs/hyperbolic_honeycomb.png")
    ap.add_argument("--film", default="../outputs/hyperbolic_honeycomb.bin")
    ap.add_argument("--samples", type=int, default=160)
    ap.add_argument("--width", type=int, default=400)
    ap.add_argument("--height", type=int, default=400)
    ap.add_argument("--fov", type=float, default=95.0)
    args = ap.parse_args()

    cells = grow_honeycomb(max_cells=args.cells, max_layer=args.max_layer)
    print(f"Generated {len(cells)} cells")
    layer_counts = {}
    for c in cells:
        layer_counts[c.layer] = layer_counts.get(c.layer, 0) + 1
    for layer in sorted(layer_counts):
        print(f"  layer {layer}: {layer_counts[layer]} cells")

    # Camera: just inside the base cube near the -x face, looking +x down the
    # spine. With BFS reflecting through face x=+a first, the +x neighbour is
    # the first hop, and the corridor extends in that direction. Mild
    # hyperbolic fog so distant cells are tinted but still visible as a
    # self-similar receding lattice -- the genuine non-euclidean signature.
    cam_pos = (-A * 0.92, 0.0, 0.0)
    cam_look = (A * 4.0, 0.0, 0.0)
    sigma_a = (0.10, 0.18, 0.32)

    repo_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    out_path = os.path.join(repo_root, args.out)
    emit_scene(cells, out_path, args.image, args.film,
               cam_pos, cam_look, args.fov,
               args.samples, args.width, args.height,
               sigma_a, max_arc=0)
    print(f"Wrote {out_path}")


if __name__ == "__main__":
    sys.exit(main())
