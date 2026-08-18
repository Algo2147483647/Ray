"""Generate the closed, faceted moissanite mesh used by this experiment."""

from __future__ import annotations

import argparse
import json
import math
from pathlib import Path


SEGMENTS = 32


def ring(radius: float, z: float, phase: float = 0.0) -> list[list[float]]:
    return [
        [
            round(radius * math.cos(2 * math.pi * i / SEGMENTS + phase), 8),
            round(radius * math.sin(2 * math.pi * i / SEGMENTS + phase), 8),
            z,
        ]
        for i in range(SEGMENTS)
    ]


def triangle(facet_id: str, p1: list[float], p2: list[float], p3: list[float]) -> dict:
    return {
        "id": facet_id,
        "shape": "triangle",
        "p1": p1,
        "p2": p2,
        "p3": p3,
        "medium_boundary": {
            "outside": "air",
            "inside": "moissanite-crystal",
            "priority": 20,
            "thin": False,
        },
    }


def connect_rings(name: str, lower: list[list[float]], upper: list[list[float]]) -> list[dict]:
    facets = []
    for i in range(SEGMENTS):
        j = (i + 1) % SEGMENTS
        facets.append(triangle(f"{name}-{i:02d}-a", lower[i], lower[j], upper[j]))
        facets.append(triangle(f"{name}-{i:02d}-b", lower[i], upper[j], upper[i]))
    return facets


def build_mesh() -> list[dict]:
    half_step = math.pi / SEGMENTS
    pavilion_break = ring(0.31, 0.155, half_step)
    lower_girdle = ring(0.5, 0.39)
    upper_girdle = ring(0.5, 0.42)
    star = ring(0.365, 0.525, half_step)
    table = ring(0.265, 0.59)
    culet = [0.0, 0.0, 0.0]
    table_center = [0.0, 0.0, 0.59]

    facets: list[dict] = []
    for i in range(SEGMENTS):
        j = (i + 1) % SEGMENTS
        facets.append(triangle(f"culet-main-{i:02d}", culet, pavilion_break[j], pavilion_break[i]))
    facets.extend(connect_rings("pavilion-lower", pavilion_break, lower_girdle))
    facets.extend(connect_rings("girdle", lower_girdle, upper_girdle))
    facets.extend(connect_rings("crown-bezel", upper_girdle, star))
    facets.extend(connect_rings("crown-star", star, table))
    for i in range(SEGMENTS):
        j = (i + 1) % SEGMENTS
        facets.append(triangle(f"table-{i:02d}", table_center, table[i], table[j]))
    return facets


def document() -> dict:
    return {
        "_comment": "Generated round-brilliant moissanite in local coordinates. The scene rig owns placement and lighting.",
        "media": {
            "moissanite-crystal": {
                "type": "homogeneous",
                "ior": {"type": "cauchy", "a": 2.41, "b": 0.083, "c": 0.0},
            }
        },
        "materials": [
            {
                "id": "faceted-moissanite",
                "surface": {
                    "type": "specular_dielectric",
                    "reflectance": {"type": "constant", "value": 1.0},
                    "transmittance": {"type": "constant", "value": 0.995},
                    "eta_outside": 1.0,
                    "ior": {"type": "cauchy", "a": 2.41, "b": 0.083, "c": 0.0},
                },
            }
        ],
        "objects": [
            {
                "id": "moissanite-fire-rig",
                "shape": "group",
                "_comment": "Model layer of the shared rig. The scene layer with the same Group id supplies placement and the collimated light.",
                "objects": [
                    {
                        "id": "moissanite",
                        "shape": "group",
                        "_comment": "Local cut: culet is [0, 0, 0], table is centred on the local z axis.",
                        "material_id": "faceted-moissanite",
                        "objects": build_mesh(),
                    }
                ],
            }
        ],
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, default=Path(__file__).with_name("moissanite.json"))
    args = parser.parse_args()
    args.output.write_text(json.dumps(document(), indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
