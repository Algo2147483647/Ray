#!/usr/bin/env python3
"""Generate a compact scene containing three independent H^3 witnesses.

The scene uses the Beltrami-Klein chart and deliberately avoids relying on
ordinary perspective alone:

1. Five congruent {4,3,5} cubes meet around one geodesic edge.
2. Equal-radius hyperbolic balls are placed at equal hyperbolic intervals
   along a chord and shrink rapidly in Klein coordinates near infinity.
3. A geodesic triangle has an intrinsic angle sum strictly below pi.
"""

from __future__ import annotations

import argparse
import copy
import json
import math
from dataclasses import dataclass
from pathlib import Path


SQRT5 = math.sqrt(5.0)
A2 = SQRT5 - 2.0
A = math.sqrt(A2)
KLEIN_MARGIN2 = 0.999

BASE_VERTICES = [
    (
        A if (index & 1) else -A,
        A if (index & 2) else -A,
        A if (index & 4) else -A,
    )
    for index in range(8)
]

BASE_FACES = [
    [0, 4, 6, 2],
    [1, 3, 7, 5],
    [0, 1, 5, 4],
    [2, 6, 7, 3],
    [0, 2, 3, 1],
    [4, 5, 7, 6],
]

BASE_EDGES = [
    (0, 1), (2, 3), (4, 5), (6, 7),
    (0, 2), (1, 3), (4, 6), (5, 7),
    (0, 4), (1, 5), (2, 6), (3, 7),
]

CELL_COLORS = [
    (5.5, 2.2, 0.35),
    (5.0, 0.55, 0.35),
    (0.25, 3.4, 5.5),
    (2.4, 0.45, 5.2),
    (0.35, 4.6, 1.25),
]

TRIANGLE_COLORS = [
    (5.5, 0.35, 0.25),
    (0.25, 5.0, 0.7),
    (0.25, 1.5, 5.5),
]


@dataclass
class Cell:
    vertices: list[tuple[float, float, float]]


def dot(left, right):
    return sum(a * b for a, b in zip(left, right))


def norm2(point):
    return dot(point, point)


def add(left, right):
    return tuple(a + b for a, b in zip(left, right))


def sub(left, right):
    return tuple(a - b for a, b in zip(left, right))


def scale(value, vector):
    return tuple(value * component for component in vector)


def midpoint(left, right):
    return scale(0.5, add(left, right))


def length(vector):
    return math.sqrt(norm2(vector))


def cross(left, right):
    return (
        left[1] * right[2] - left[2] * right[1],
        left[2] * right[0] - left[0] * right[2],
        left[0] * right[1] - left[1] * right[0],
    )


def cell_center(cell):
    total = (0.0, 0.0, 0.0)
    for vertex in cell.vertices:
        total = add(total, vertex)
    return scale(1.0 / len(cell.vertices), total)


def cell_key(cell):
    return tuple(sorted(tuple(round(value, 8) for value in vertex) for vertex in cell.vertices))


def face_plane(cell, face):
    vertices = [cell.vertices[index] for index in face]
    normal = cross(sub(vertices[1], vertices[0]), sub(vertices[2], vertices[0]))
    normal = scale(1.0 / length(normal), normal)
    offset = dot(normal, vertices[0])
    if dot(normal, cell_center(cell)) > offset:
        normal = scale(-1.0, normal)
        offset = -offset
    return normal, offset


def reflect_point(point, normal, offset):
    k = 2.0 * (dot(normal, point) - offset) / (1.0 - offset * offset)
    denominator = 1.0 - k * offset
    return scale(1.0 / denominator, sub(point, scale(k, normal)))


def reflect_cell(cell, face):
    normal, offset = face_plane(cell, face)
    return Cell([reflect_point(vertex, normal, offset) for vertex in cell.vertices])


def five_cells_around_edge():
    """Orbit the base cube around its y=+A,z=+A edge."""
    base = Cell(list(BASE_VERTICES))
    cells = [base]
    current = base
    for step in range(1, 5):
        face_index = 3 if step % 2 else 5
        current = reflect_cell(current, BASE_FACES[face_index])
        cells.append(current)

    closing_face = 3 if 5 % 2 else 5
    closed = reflect_cell(current, BASE_FACES[closing_face])
    if cell_key(closed) != cell_key(base):
        raise ValueError("five-cell orbit did not close")
    return cells


def hyperbolic_distance(left, right):
    denominator = math.sqrt((1.0 - norm2(left)) * (1.0 - norm2(right)))
    value = (1.0 - dot(left, right)) / denominator
    return math.acosh(max(1.0, value))


def klein_inner(point, left, right):
    weight = 1.0 - norm2(point)
    return dot(left, right) / weight + dot(point, left) * dot(point, right) / (weight * weight)


def hyperbolic_angle(vertex, left, right):
    first = sub(left, vertex)
    second = sub(right, vertex)
    cosine = klein_inner(vertex, first, second)
    cosine /= math.sqrt(klein_inner(vertex, first, first) * klein_inner(vertex, second, second))
    return math.acos(max(-1.0, min(1.0, cosine)))


def equally_spaced_chord_points(count, spacing, start_x, y, z):
    boundary_x = math.sqrt(1.0 - y * y - z * z)
    points = [(start_x, y, z)]
    for _ in range(count - 1):
        previous = points[-1]
        low = previous[0]
        high = boundary_x * (1.0 - 1e-10)
        for _ in range(80):
            middle = 0.5 * (low + high)
            candidate = (middle, y, z)
            if hyperbolic_distance(previous, candidate) < spacing:
                low = middle
            else:
                high = middle
        points.append((0.5 * (low + high), y, z))
    return points


def hyperbolic_ball(center, radius):
    """Return A, B, C for x^T A x + B^T x + C = 0."""
    factor = math.cosh(radius) ** 2 * (1.0 - norm2(center))
    matrix = []
    for row in range(3):
        for column in range(3):
            value = center[row] * center[column]
            if row == column:
                value += factor
            matrix.append(value)
    linear = [-2.0 * value for value in center]
    constant = 1.0 - factor
    return matrix, linear, constant


def emission_material(material_id, color):
    return {
        "id": material_id,
        "emission": {
            "type": "constant",
            "radiance": {"type": "rgb", "value": list(color)},
        },
    }


def cylinder_object(object_id, material_id, start, end, radius):
    axis = sub(end, start)
    return {
        "id": object_id,
        "material_id": material_id,
        "shape": "finite cylinder",
        "center": list(midpoint(start, end)),
        "axis": list(axis),
        "r": radius,
        "height": length(axis),
    }


def ball_object(object_id, material_id, center, intrinsic_radius):
    matrix, linear, constant = hyperbolic_ball(center, intrinsic_radius)
    return {
        "id": object_id,
        "material_id": material_id,
        "shape": "quadratic equation",
        "a": matrix,
        "b": linear,
        "c": constant,
    }


def build_scene(width, height, samples):
    cells = five_cells_around_edge()
    ladder = equally_spaced_chord_points(
        count=6,
        spacing=0.55,
        start_x=-0.42,
        y=-0.26,
        z=-0.26,
    )
    triangle = [
        (-0.62, 0.00, -0.48),
        (0.62, 0.00, -0.48),
        (0.00, 0.62, -0.48),
    ]

    materials = [
        emission_material("shared_edge", (7.0, 6.2, 4.2)),
        emission_material("metric_ball", (0.25, 3.2, 6.5)),
        emission_material("metric_axis", (0.15, 0.8, 2.4)),
        emission_material("vertex_marker", (5.5, 4.5, 1.0)),
    ]
    materials.extend(
        emission_material(f"cell_{index}", color)
        for index, color in enumerate(CELL_COLORS)
    )
    materials.extend(
        emission_material(f"triangle_{index}", color)
        for index, color in enumerate(TRIANGLE_COLORS)
    )

    objects = []
    shared_edge = (BASE_VERTICES[6], BASE_VERTICES[7])
    shared_key = tuple(sorted(shared_edge))
    seen_edges = set()
    for cell_index, cell in enumerate(cells):
        for edge_index, (left_index, right_index) in enumerate(BASE_EDGES):
            start = cell.vertices[left_index]
            end = cell.vertices[right_index]
            key = tuple(sorted((
                tuple(round(value, 7) for value in start),
                tuple(round(value, 7) for value in end),
            )))
            if key in seen_edges:
                continue
            seen_edges.add(key)
            if tuple(sorted((start, end))) == shared_key:
                continue
            objects.append(cylinder_object(
                f"orbit_cell_{cell_index}_edge_{edge_index}",
                f"cell_{cell_index}",
                start,
                end,
                A * 0.026,
            ))
        objects.append(ball_object(
            f"orbit_cell_{cell_index}_center",
            f"cell_{cell_index}",
            cell_center(cell),
            0.075,
        ))

    objects.append(cylinder_object(
        "five_cells_shared_geodesic_edge",
        "shared_edge",
        shared_edge[0],
        shared_edge[1],
        A * 0.075,
    ))

    objects.append(cylinder_object(
        "equal_spacing_geodesic_axis",
        "metric_axis",
        ladder[0],
        ladder[-1],
        0.008,
    ))
    for index, center in enumerate(ladder):
        objects.append(ball_object(
            f"equal_h3_ball_{index}",
            "metric_ball",
            center,
            0.105,
        ))

    for index in range(3):
        start = triangle[index]
        end = triangle[(index + 1) % 3]
        objects.append(cylinder_object(
            f"angle_defect_edge_{index}",
            f"triangle_{index}",
            start,
            end,
            0.018,
        ))
        objects.append(ball_object(
            f"angle_defect_vertex_{index}",
            "vertex_marker",
            start,
            0.055,
        ))

    dihedral_angle = math.acos(A2 / (1.0 - A2))
    triangle_angles = [
        hyperbolic_angle(triangle[index], triangle[(index + 1) % 3], triangle[(index + 2) % 3])
        for index in range(3)
    ]
    ladder_distances = [
        hyperbolic_distance(ladder[index], ladder[index + 1])
        for index in range(len(ladder) - 1)
    ]

    all_points = [vertex for cell in cells for vertex in cell.vertices] + ladder + triangle
    if max(map(norm2, all_points)) >= KLEIN_MARGIN2:
        raise ValueError("showcase point crossed the Klein safety margin")
    if abs(dihedral_angle - 2.0 * math.pi / 5.0) > 1e-12:
        raise ValueError("cube dihedral angle is not 2*pi/5")
    if max(abs(distance - 0.55) for distance in ladder_distances) > 1e-10:
        raise ValueError("metric ladder spacing drifted")
    if sum(triangle_angles) >= math.pi:
        raise ValueError("triangle has no hyperbolic angle defect")

    return {
        "_comment": (
            "H^3/Klein showcase. Top: five congruent {4,3,5} cubes meet around "
            "one white geodesic edge. Middle: six equal-radius H^3 balls are "
            "equally spaced by arc length 0.55 along one chord, while their "
            "Klein coordinate size collapses near the ideal boundary. Bottom: "
            "an RGB geodesic triangle has intrinsic angle sum below 180 degrees."
        ),
        "_validation": {
            "cell_count_around_edge": 5,
            "cube_dihedral_degrees": math.degrees(dihedral_angle),
            "metric_ball_intrinsic_radius": 0.105,
            "metric_ladder_arc_distances": ladder_distances,
            "triangle_angles_degrees": [math.degrees(angle) for angle in triangle_angles],
            "triangle_angle_sum_degrees": math.degrees(sum(triangle_angles)),
            "max_klein_radius": math.sqrt(max(map(norm2, all_points))),
        },
        "render": {
            "dimension": 3,
            "samples": samples,
            "thread_num": 0,
            "width": width,
            "height": height,
            "camera_index": 0,
            "output_image": "../outputs/hyperbolic_showcase_edge.png",
            "output_film": "../outputs/hyperbolic_showcase_edge.bin",
            "exposure": 1.0,
            "tone_mapping": "aces",
            "gamma": 2.2,
            "spectrum_mode": "rgb",
            "color_space": "linear_srgb",
        },
        "geometry": {"type": "klein", "max_arc": 0},
        "cameras": [
            {
                "id": "five-around-edge",
                "type": "hyperbolic",
                "position": [-0.68, 0.459, 0.459],
                "look_at": [0.25, A, A],
                "up": [0.0, 0.0, 1.0],
                "field_of_view": 72,
                "aspect_ratio": width / height,
            },
            {
                "id": "metric-ladder-and-angle-defect",
                "type": "hyperbolic",
                "position": [0.0, -0.75, -0.45],
                "look_at": [0.20, -0.02, -0.37],
                "up": [0.0, 0.0, 1.0],
                "field_of_view": 72,
                "aspect_ratio": width / height,
            },
            {
                "id": "angle-defect",
                "type": "hyperbolic",
                "position": [0.0, -0.55, 0.72],
                "look_at": [0.0, 0.20, -0.48],
                "up": [0.0, 1.0, 0.0],
                "field_of_view": 38,
                "aspect_ratio": width / height,
            },
            {
                "id": "overview",
                "type": "hyperbolic",
                "position": [0.0, -0.84, 0.04],
                "look_at": [0.12, 0.22, -0.02],
                "up": [0.0, 0.0, 1.0],
                "field_of_view": 96,
                "aspect_ratio": width / height,
            },
        ],
        "materials": materials,
        "objects": objects,
    }


def write_scene(path, scene):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(scene, indent=2) + "\n", encoding="utf-8")
    print(f"Wrote {path}")


def specialized_scene(scene, feature, camera_index, object_prefixes):
    result = copy.deepcopy(scene)
    result["_comment"] = (
        f"Focused {feature} view generated from hyperbolic_showcase.py. "
        + scene["_comment"]
    )
    result["render"]["camera_index"] = 0
    result["render"]["output_image"] = f"../outputs/hyperbolic_showcase_{feature}.png"
    result["render"]["output_film"] = f"../outputs/hyperbolic_showcase_{feature}.bin"
    result["cameras"] = [result["cameras"][camera_index]]
    result["objects"] = [
        obj for obj in result["objects"]
        if any(obj["id"].startswith(prefix) for prefix in object_prefixes)
    ]
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--out",
        default="examples/scenes/non-euclidean/hyperbolic_showcase.json",
    )
    parser.add_argument("--width", type=int, default=640)
    parser.add_argument("--height", type=int, default=640)
    parser.add_argument("--samples", type=int, default=64)
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parents[3]
    output_path = repo_root / args.out
    scene = build_scene(args.width, args.height, args.samples)
    write_scene(output_path, scene)

    focused_scenes = {
        "edge": specialized_scene(
            scene,
            "edge",
            camera_index=0,
            object_prefixes=("orbit_cell_", "five_cells_"),
        ),
        "metric": specialized_scene(
            scene,
            "metric",
            camera_index=1,
            object_prefixes=("equal_",),
        ),
        "triangle": specialized_scene(
            scene,
            "triangle",
            camera_index=2,
            object_prefixes=("angle_defect_",),
        ),
    }
    for suffix, focused in focused_scenes.items():
        focused_path = output_path.with_name(f"{output_path.stem}_{suffix}{output_path.suffix}")
        write_scene(focused_path, focused)

    print(json.dumps(scene["_validation"], indent=2))


if __name__ == "__main__":
    main()
