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


def equally_spaced_vertical_chord_points(count, spacing, x, y, start_z):
    boundary_z = math.sqrt(1.0 - x * x - y * y)
    points = [(x, y, start_z)]
    for _ in range(count - 1):
        previous = points[-1]
        low = previous[2]
        high = boundary_z * (1.0 - 1e-10)
        for _ in range(80):
            middle = 0.5 * (low + high)
            candidate = (x, y, middle)
            if hyperbolic_distance(previous, candidate) < spacing:
                low = middle
            else:
                high = middle
        points.append((x, y, 0.5 * (low + high)))
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


def lambert_material(material_id, color):
    return {
        "id": material_id,
        "surface": {
            "type": "lambert",
            "albedo": list(color),
        },
    }


def mirror_material(material_id, color):
    return {
        "id": material_id,
        "surface": {
            "type": "specular_reflection",
            "reflectance": list(color),
        },
    }


def rough_conductor_material(material_id, eta, extinction, roughness):
    return {
        "id": material_id,
        "surface": {
            "type": "rough_conductor",
            "eta": list(eta),
            "k": list(extinction),
            "roughness": roughness,
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


def triangle_object(object_id, material_id, p1, p2, p3):
    return {
        "id": object_id,
        "material_id": material_id,
        "shape": "triangle",
        "p1": list(p1),
        "p2": list(p2),
        "p3": list(p3),
    }


def horizontal_frame_point(point):
    """Rotate the common x edge onto z so the five-cell orbit lies horizontally."""
    return (point[1], point[2], point[0])


def horizontal_frame_object(obj):
    """Apply the same axis permutation to cylinders and implicit H^3 balls."""
    result = copy.deepcopy(obj)
    if result["shape"] == "finite cylinder":
        result["center"] = list(horizontal_frame_point(result["center"]))
        result["axis"] = list(horizontal_frame_point(result["axis"]))
    elif result["shape"] == "quadratic equation":
        permutation = (1, 2, 0)
        matrix = result["a"]
        result["a"] = [
            matrix[permutation[row] * 3 + permutation[column]]
            for row in range(3)
            for column in range(3)
        ]
        result["b"] = [result["b"][index] for index in permutation]
    return result


def horizontal_cell_face_objects(cells):
    """Build pastel exterior panels, open shared faces, and inset ceiling lights."""
    grouped_faces = {}
    for cell_index, cell in enumerate(cells):
        for face_index, face in enumerate(BASE_FACES):
            vertices = [
                horizontal_frame_point(cell.vertices[index])
                for index in face
            ]
            key = tuple(sorted(
                tuple(round(component, 8) for component in vertex)
                for vertex in vertices
            ))
            grouped_faces.setdefault(key, []).append(
                (cell_index, face_index, vertices)
            )

    objects = []
    for entries in grouped_faces.values():
        if len(entries) == 2:
            # Adjacent rooms remain completely open to each other.
            continue
        elif len(entries) == 1:
            cell_index, face_index, vertices = entries[0]
            prefix = f"orbit_cell_{cell_index}_outer_face_{face_index}"
            material_id = f"cell_diffuse_{cell_index}"
        else:
            raise ValueError("unexpected five-cell face multiplicity")

        objects.extend([
            triangle_object(
                f"{prefix}_triangle_0",
                material_id,
                vertices[0],
                vertices[1],
                vertices[2],
            ),
            triangle_object(
                f"{prefix}_triangle_1",
                material_id,
                vertices[0],
                vertices[2],
                vertices[3],
            ),
        ])

    for cell_index, cell in enumerate(cells):
        rotated_faces = [
            [
                horizontal_frame_point(cell.vertices[index])
                for index in face
            ]
            for face in BASE_FACES
        ]
        top_vertices = max(
            rotated_faces,
            key=lambda vertices: sum(vertex[2] for vertex in vertices),
        )
        face_center = scale(
            0.25,
            tuple(sum(vertex[axis] for vertex in top_vertices) for axis in range(3)),
        )
        room_center = horizontal_frame_point(cell_center(cell))
        light_vertices = [
            add(
                add(face_center, scale(0.58, sub(vertex, face_center))),
                scale(0.008, sub(room_center, face_center)),
            )
            for vertex in top_vertices
        ]
        prefix = f"orbit_cell_{cell_index}_ceiling_light"
        objects.extend([
            triangle_object(
                f"{prefix}_triangle_0",
                "cell_ceiling_light",
                light_vertices[0],
                light_vertices[1],
                light_vertices[2],
            ),
            triangle_object(
                f"{prefix}_triangle_1",
                "cell_ceiling_light",
                light_vertices[0],
                light_vertices[2],
                light_vertices[3],
            ),
        ])
    return objects


def blue_room_wall_triangle(cells, camera_position):
    """Place a triangle inside the blue room's vertical exterior wall facing camera."""
    grouped_faces = {}
    for cell_index, cell in enumerate(cells):
        for face_index, face in enumerate(BASE_FACES):
            vertices = [
                horizontal_frame_point(cell.vertices[index])
                for index in face
            ]
            key = tuple(sorted(
                tuple(round(component, 8) for component in vertex)
                for vertex in vertices
            ))
            grouped_faces.setdefault(key, []).append(
                (cell_index, face_index, vertices)
            )

    blue_cell_index = 4
    candidates = []
    for entries in grouped_faces.values():
        if len(entries) != 1 or entries[0][0] != blue_cell_index:
            continue
        _, _, vertices = entries[0]
        normal = cross(
            sub(vertices[1], vertices[0]),
            sub(vertices[2], vertices[0]),
        )
        normal = scale(1.0 / length(normal), normal)
        if abs(normal[2]) > 1e-8:
            continue
        face_center = scale(
            0.25,
            tuple(sum(vertex[axis] for vertex in vertices) for axis in range(3)),
        )
        to_camera = sub(camera_position, face_center)
        score = abs(dot(normal, scale(1.0 / length(to_camera), to_camera)))
        candidates.append((score, vertices, face_center))

    if not candidates:
        raise ValueError("blue room has no vertical exterior wall")
    _, vertices, face_center = max(candidates, key=lambda candidate: candidate[0])

    top_vertices = sorted(vertices, key=lambda vertex: vertex[2], reverse=True)[:2]
    bottom_vertices = sorted(vertices, key=lambda vertex: vertex[2])[:2]
    top_center = midpoint(top_vertices[0], top_vertices[1])
    bottom_vertices.sort(key=lambda vertex: (vertex[0], vertex[1]))
    wall_offset = scale(0.004, sub(camera_position, face_center))
    return [
        add(add(face_center, scale(0.62, sub(top_center, face_center))), wall_offset),
        add(add(face_center, scale(0.62, sub(bottom_vertices[0], face_center))), wall_offset),
        add(add(face_center, scale(0.62, sub(bottom_vertices[1], face_center))), wall_offset),
    ]


def build_scene(width, height, samples):
    cells = five_cells_around_edge()
    ladder_spacing = 0.18
    ladder = equally_spaced_vertical_chord_points(
        count=6,
        spacing=ladder_spacing,
        x=0.68,
        y=-0.08,
        start_z=-0.29,
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
    if max(abs(distance - ladder_spacing) for distance in ladder_distances) > 1e-10:
        raise ValueError("metric ladder spacing drifted")
    if sum(triangle_angles) >= math.pi:
        raise ValueError("triangle has no hyperbolic angle defect")

    return {
        "_comment": (
            "H^3/Klein showcase. Top: five congruent {4,3,5} cubes meet around "
            "one white geodesic edge. Six equal-radius H^3 balls form a vertical "
            "connector inside the yellow room, equally spaced by arc length 0.18 "
            "along one chord. An RGB geodesic triangle has intrinsic angle sum "
            "below 180 degrees."
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


def artistic_scene(scene):
    result = copy.deepcopy(scene)
    cells = five_cells_around_edge()
    center_column_midpoint = horizontal_frame_point(midpoint(
        BASE_VERTICES[6],
        BASE_VERTICES[7],
    ))
    camera_position = (-0.22, -0.22, 0.12)
    wall_triangle = blue_room_wall_triangle(cells, camera_position)
    result["_comment"] = (
        "Horizontal five-room H^3 frame around a white vertical center column. "
        "Each room has a pure red, yellow, green, cyan, or blue emissive frame, "
        "matching pastel diffuse exterior faces, open shared faces, a square "
        "white ceiling light, and a perfectly reflective ball at its center."
    )
    result["render"].update({
        "camera_index": 0,
        "samples": 128,
        "width": 640,
        "height": 640,
        "output_image": "../outputs/hyperbolic_showcase_art.png",
        "output_film": "../outputs/hyperbolic_showcase_art.bin",
        "exposure": 1.15,
        "tone_mapping": "aces",
    })
    result["cameras"] = [{
        "id": "h3-observatory",
        "type": "hyperbolic",
        "position": list(camera_position),
        "direction": list(sub(center_column_midpoint, camera_position)),
        "up": [0.0, 0.0, 1.0],
        "field_of_views": [100, 100],
    }]

    result["materials"] = [
        emission_material("cell_glow_0", (4.0, 0.0, 0.0)),
        emission_material("cell_glow_1", (4.0, 4.0, 0.0)),
        emission_material("cell_glow_2", (0.0, 4.0, 0.0)),
        emission_material("cell_glow_3", (0.0, 4.0, 4.0)),
        emission_material("cell_glow_4", (0.0, 0.0, 4.0)),
        lambert_material("cell_diffuse_0", (0.92, 0.42, 0.42)),
        lambert_material("cell_diffuse_1", (0.92, 0.88, 0.42)),
        lambert_material("cell_diffuse_2", (0.42, 0.88, 0.48)),
        lambert_material("cell_diffuse_3", (0.42, 0.88, 0.88)),
        lambert_material("cell_diffuse_4", (0.42, 0.56, 0.92)),
        mirror_material("cell_center_mirror", (1.0, 1.0, 1.0)),
        emission_material("cell_ceiling_light", (8.0, 8.0, 8.0)),
        lambert_material("metric_porcelain", (0.16, 0.48, 0.92)),
        emission_material("metric_axis", (0.08, 0.55, 1.8)),
        emission_material("vertex_marker", (3.8, 2.5, 0.65)),
        emission_material("triangle_0", (4.5, 0.28, 0.16)),
        emission_material("triangle_1", (0.16, 3.8, 0.65)),
        emission_material("triangle_2", (0.18, 0.9, 4.6)),
        lambert_material("stage", (0.025, 0.032, 0.048)),
        emission_material("warm_moon", (8.0, 4.2, 1.8)),
        emission_material("cool_moon", (1.4, 3.8, 8.0)),
        emission_material("rear_softbox", (10.0, 8.8, 7.2)),
    ]

    for obj in result["objects"]:
        object_id = obj["id"]
        if object_id.startswith("orbit_cell_"):
            parts = object_id.split("_")
            cell_index = int(parts[2])
            if "_edge_" in object_id:
                obj["material_id"] = f"cell_glow_{cell_index}"
                obj["r"] *= 0.70
            elif object_id.endswith("_center"):
                obj["material_id"] = "cell_center_mirror"
            rotated = horizontal_frame_object(obj)
            obj.clear()
            obj.update(rotated)
        elif object_id == "five_cells_shared_geodesic_edge":
            rotated = horizontal_frame_object(obj)
            obj.clear()
            obj.update(rotated)
        elif object_id.startswith("angle_defect_edge_"):
            index = int(object_id.rsplit("_", 1)[1])
            moved = cylinder_object(
                object_id,
                f"triangle_{index}",
                wall_triangle[index],
                wall_triangle[(index + 1) % 3],
                0.012,
            )
            obj.clear()
            obj.update(moved)
        elif object_id.startswith("angle_defect_vertex_"):
            index = int(object_id.rsplit("_", 1)[1])
            moved = ball_object(
                object_id,
                "vertex_marker",
                wall_triangle[index],
                0.032,
            )
            obj.clear()
            obj.update(moved)
        if object_id.startswith("equal_h3_ball_"):
            obj["material_id"] = "metric_porcelain"

    result["objects"].extend(horizontal_cell_face_objects(cells))
    result["objects"].extend([
        {
            "id": "dark_diffuse_stage",
            "material_id": "stage",
            "shape": "circle",
            "center": [0.0, 0.02, -0.555],
            "normal": [0.0, 0.0, 1.0],
            "r": 0.76,
        },
        ball_object(
            "warm_h3_moon",
            "warm_moon",
            (-0.68, -0.34, 0.43),
            0.13,
        ),
        ball_object(
            "cool_h3_moon",
            "cool_moon",
            (0.66, -0.30, 0.40),
            0.12,
        ),
        {
            "id": "rear_invisible_softbox",
            "material_id": "rear_softbox",
            "shape": "circle",
            "center": [0.0, -0.88, 0.05],
            "normal": [0.0, 1.0, 0.0],
            "r": 0.28,
        },
    ])

    mirror_material_ids = {
        material["id"]
        for material in result["materials"]
        if material.get("surface", {}).get("type") == "specular_reflection"
    }

    def is_room_center_ball(obj):
        return (
            obj["id"].startswith("orbit_cell_")
            and obj["id"].endswith("_center")
        )

    def is_center_column(obj):
        if obj.get("shape") != "finite cylinder":
            return False
        center = obj.get("center", ())
        axis = obj.get("axis", ())
        return (
            len(center) == 3
            and len(axis) == 3
            and abs(center[0] - center_column_midpoint[0]) < 1e-8
            and abs(center[1] - center_column_midpoint[1]) < 1e-8
            and abs(axis[0]) < 1e-8
            and abs(axis[1]) < 1e-8
        )

    result["objects"] = [
        obj for obj in result["objects"]
        if not is_center_column(obj)
        and (
            obj["material_id"] not in mirror_material_ids
            or is_room_center_ball(obj)
        )
    ]
    used_materials = {obj["material_id"] for obj in result["objects"]}
    result["materials"] = [
        material for material in result["materials"]
        if material["id"] in used_materials
    ]

    triangle_angles = [
        hyperbolic_angle(
            wall_triangle[index],
            wall_triangle[(index + 1) % 3],
            wall_triangle[(index + 2) % 3],
        )
        for index in range(3)
    ]
    result["_validation"]["triangle_angles_degrees"] = [
        math.degrees(angle) for angle in triangle_angles
    ]
    result["_validation"]["triangle_angle_sum_degrees"] = math.degrees(
        sum(triangle_angles)
    )
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--out",
        default="examples/scenes/non-euclidean/hyperbolic_showcase_art.json",
    )
    parser.add_argument("--width", type=int, default=640)
    parser.add_argument("--height", type=int, default=640)
    parser.add_argument("--samples", type=int, default=64)
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parents[3]
    output_path = repo_root / args.out
    scene = build_scene(args.width, args.height, args.samples)
    art_scene = artistic_scene(scene)
    write_scene(output_path, art_scene)
    print(json.dumps(art_scene["_validation"], indent=2))


if __name__ == "__main__":
    main()
