"""Visualize a 3D film produced by the Go ray tracer.

The renderer stores film data as:

  int64 samples
  int32 shape_len
  int32[shape_len] shape
  float64 red[prod(shape)]
  float64 green[prod(shape)]
  float64 blue[prod(shape)]

Tensor order is x-fastest: index = x + y*width + z*width*height.

This script prefers the `.bin` film because it preserves linear radiance and
the original 3D tensor. It can also read the PNG atlas emitted by the renderer.
It writes static analysis images instead of opening a fragile GUI window.
"""

from __future__ import annotations

import argparse
import math
import struct
from pathlib import Path

import matplotlib

matplotlib.use("Agg")

import matplotlib.pyplot as plt
import numpy as np
from PIL import Image, ImageDraw


DEFAULT_FILM = (
    Path(__file__).resolve().parents[2]
    / "outputs"
    / "4d-hypercube-lit-by-hypersphere-100x100x100-200spp.bin"
)


def load_film_bin(path: Path) -> tuple[np.ndarray, dict]:
    with path.open("rb") as f:
        samples = struct.unpack("<q", f.read(8))[0]
        shape_len = struct.unpack("<i", f.read(4))[0]
        shape = list(struct.unpack("<" + "i" * shape_len, f.read(4 * shape_len)))
        total = math.prod(shape)

        channels = []
        for _ in range(3):
            channel = np.frombuffer(f.read(total * 8), dtype="<f8").copy()
            if channel.size != total:
                raise ValueError(f"{path} ended before all RGB film data was read")
            channels.append(_reshape_channel(channel, shape))

    volume = np.stack(channels, axis=-1)
    meta = {"samples": samples, "shape": shape, "source": str(path)}
    return volume, meta


def _reshape_channel(flat: np.ndarray, shape: list[int]) -> np.ndarray:
    if len(shape) != 3:
        raise ValueError(f"expected a 3D film shape, got {shape}")
    width, height, depth = shape
    return flat.reshape((depth, height, width)).transpose(2, 1, 0)


def load_png_atlas(path: Path, width: int, height: int, depth: int) -> tuple[np.ndarray, dict]:
    img = Image.open(path).convert("RGB")
    atlas = np.asarray(img, dtype=np.float32) / 255.0
    cols = math.ceil(math.sqrt(depth))
    rows = math.ceil(depth / cols)
    if atlas.shape[1] < cols * width or atlas.shape[0] < rows * height:
        raise ValueError(
            f"atlas {path} is {atlas.shape[1]}x{atlas.shape[0]}, too small for "
            f"{width}x{height}x{depth}"
        )

    volume = np.zeros((width, height, depth, 3), dtype=np.float32)
    for z in range(depth):
        ax = z % cols
        ay = z // cols
        tile = atlas[ay * height : (ay + 1) * height, ax * width : (ax + 1) * width]
        volume[:, :, z, :] = tile.transpose(1, 0, 2)
    return volume, {"samples": None, "shape": [width, height, depth], "source": str(path)}


def tone_map(volume: np.ndarray, exposure: float, percentile: float, gamma: float) -> np.ndarray:
    scaled = np.maximum(volume * exposure, 0)
    hi = np.percentile(scaled, percentile)
    if not np.isfinite(hi) or hi <= 0:
        hi = max(float(scaled.max()), 1.0)
    mapped = np.clip(scaled / hi, 0, 1)
    if gamma > 0 and gamma != 1:
        mapped = np.power(mapped, 1 / gamma)
    return mapped


def luminance(rgb: np.ndarray) -> np.ndarray:
    return 0.2126 * rgb[..., 0] + 0.7152 * rgb[..., 1] + 0.0722 * rgb[..., 2]


def save_slice_montage(rgb: np.ndarray, out_path: Path, columns: int = 10) -> None:
    width, height, depth, _ = rgb.shape
    rows = math.ceil(depth / columns)
    canvas = Image.new("RGB", (columns * width, rows * height), "black")
    draw = ImageDraw.Draw(canvas)

    image_u8 = (np.clip(rgb, 0, 1) * 255 + 0.5).astype(np.uint8)
    for z in range(depth):
        tile = Image.fromarray(image_u8[:, :, z, :].transpose(1, 0, 2), "RGB")
        x = (z % columns) * width
        y = (z // columns) * height
        canvas.paste(tile, (x, y))
        draw.text((x + 3, y + 3), f"z={z:02d}", fill=(190, 210, 230))

    canvas.save(out_path)


def save_projection_panel(rgb: np.ndarray, out_path: Path) -> None:
    projections = [
        ("XY max over film-z", rgb.max(axis=2).transpose(1, 0, 2)),
        ("XZ max over film-y", rgb.max(axis=1).transpose(1, 0, 2)),
        ("YZ max over film-x", rgb.max(axis=0).transpose(1, 0, 2)),
    ]

    fig, axes = plt.subplots(1, 3, figsize=(15, 5), facecolor="black")
    for ax, (title, img) in zip(axes, projections):
        ax.imshow(np.clip(img, 0, 1), origin="lower")
        ax.set_title(title, color="white", fontsize=11)
        ax.set_xticks([])
        ax.set_yticks([])
        ax.set_facecolor("black")
    fig.tight_layout()
    fig.savefig(out_path, dpi=160, facecolor=fig.get_facecolor())
    plt.close(fig)


def save_point_cloud(
    rgb: np.ndarray,
    out_path: Path,
    threshold_percentile: float,
    max_points: int,
    seed: int,
) -> dict:
    lum = luminance(rgb)
    threshold = np.percentile(lum[lum > 0], threshold_percentile) if np.any(lum > 0) else 1.0
    mask = lum >= threshold
    coords = np.argwhere(mask)
    if coords.size == 0:
        raise ValueError("threshold removed every voxel; lower --threshold-percentile")

    rng = np.random.default_rng(seed)
    if len(coords) > max_points:
        coords = coords[rng.choice(len(coords), size=max_points, replace=False)]

    colors = rgb[coords[:, 0], coords[:, 1], coords[:, 2]]
    alpha = np.clip(lum[coords[:, 0], coords[:, 1], coords[:, 2]], 0.15, 1.0)

    fig = plt.figure(figsize=(10, 9), facecolor="black")
    ax = fig.add_subplot(111, projection="3d", facecolor="black")
    ax.scatter(
        coords[:, 0],
        coords[:, 1],
        coords[:, 2],
        c=colors,
        alpha=alpha,
        s=2,
        linewidths=0,
        depthshade=False,
    )
    ax.set_title("3D film point cloud: bright visible voxels", color="white", pad=16)
    ax.set_xlabel("film x", color="white")
    ax.set_ylabel("film y", color="white")
    ax.set_zlabel("film z / 4D sweep", color="white")
    ax.tick_params(colors="white")
    ax.view_init(elev=22, azim=-58)
    ax.set_box_aspect([rgb.shape[0], rgb.shape[1], rgb.shape[2]])
    for axis in [ax.xaxis, ax.yaxis, ax.zaxis]:
        axis.pane.set_facecolor((0, 0, 0, 1))
        axis.pane.set_edgecolor((0.35, 0.35, 0.35, 1))
    fig.tight_layout()
    fig.savefig(out_path, dpi=170, facecolor=fig.get_facecolor())
    plt.close(fig)

    return {
        "threshold": float(threshold),
        "selected_voxels": int(mask.sum()),
        "plotted_voxels": int(len(coords)),
    }


def save_luminance_profile(rgb: np.ndarray, out_path: Path) -> dict:
    lum = luminance(rgb)
    profile = lum.mean(axis=(0, 1))
    occupied = (lum > np.percentile(lum[lum > 0], 70)).sum(axis=(0, 1)) if np.any(lum > 0) else np.zeros(lum.shape[2])

    fig, ax1 = plt.subplots(figsize=(11, 4), facecolor="black")
    ax1.set_facecolor("black")
    ax1.plot(profile, color="#ffd28a", label="mean luminance")
    ax1.set_xlabel("film z slice", color="white")
    ax1.set_ylabel("mean luminance", color="#ffd28a")
    ax1.tick_params(colors="white")
    ax2 = ax1.twinx()
    ax2.plot(occupied, color="#7fd4ff", label="occupied voxels")
    ax2.set_ylabel("occupied voxels", color="#7fd4ff")
    ax2.tick_params(colors="white")
    ax1.grid(color="#333333", linewidth=0.5)
    fig.suptitle("Slice-by-slice 4D sweep profile", color="white")
    fig.tight_layout()
    fig.savefig(out_path, dpi=170, facecolor=fig.get_facecolor())
    plt.close(fig)

    return {
        "brightest_slice": int(np.argmax(profile)),
        "mean_luminance_min": float(profile.min()),
        "mean_luminance_max": float(profile.max()),
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Analyze and visualize a 3D renderer film.")
    parser.add_argument("input", nargs="?", type=Path, default=DEFAULT_FILM)
    parser.add_argument("--png-atlas", action="store_true", help="read input as a PNG atlas instead of a .bin film")
    parser.add_argument("--shape", nargs=3, type=int, metavar=("W", "H", "D"), help="shape required for --png-atlas")
    parser.add_argument("--out-dir", type=Path, default=Path(__file__).resolve().parents[2] / "outputs" / "volume_analysis")
    parser.add_argument("--exposure", type=float, default=1.8)
    parser.add_argument("--percentile", type=float, default=99.5, help="white point percentile for display tone mapping")
    parser.add_argument("--gamma", type=float, default=2.2)
    parser.add_argument("--threshold-percentile", type=float, default=82.0)
    parser.add_argument("--max-points", type=int, default=60000)
    parser.add_argument("--seed", type=int, default=7)
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    source = args.input.resolve()
    args.out_dir.mkdir(parents=True, exist_ok=True)

    if args.png_atlas:
        if not args.shape:
            raise SystemExit("--shape W H D is required when reading a PNG atlas")
        volume, meta = load_png_atlas(source, *args.shape)
    else:
        volume, meta = load_film_bin(source)

    display_rgb = tone_map(volume, args.exposure, args.percentile, args.gamma)
    stem = source.stem

    montage_path = args.out_dir / f"{stem}-slices.png"
    projections_path = args.out_dir / f"{stem}-projections.png"
    cloud_path = args.out_dir / f"{stem}-point-cloud.png"
    profile_path = args.out_dir / f"{stem}-slice-profile.png"

    save_slice_montage(display_rgb, montage_path)
    save_projection_panel(display_rgb, projections_path)
    cloud_stats = save_point_cloud(
        display_rgb,
        cloud_path,
        threshold_percentile=args.threshold_percentile,
        max_points=args.max_points,
        seed=args.seed,
    )
    profile_stats = save_luminance_profile(display_rgb, profile_path)

    print("3D film visualization complete")
    print(f"source: {meta['source']}")
    print(f"samples: {meta['samples']}")
    print(f"shape: {meta['shape']}")
    print(f"slice montage: {montage_path}")
    print(f"projections: {projections_path}")
    print(f"point cloud: {cloud_path}")
    print(f"slice profile: {profile_path}")
    print(f"point cloud stats: {cloud_stats}")
    print(f"profile stats: {profile_stats}")


if __name__ == "__main__":
    main()
