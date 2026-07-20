#!/usr/bin/env python3
"""Render a Ray Film .bin volume with Matplotlib.

The default mode is offline-friendly and writes PNG previews. Pass --show to
open an interactive Matplotlib 3D window that can be rotated, zoomed, and panned
with the standard toolbar.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path
from typing import Optional, Sequence

import numpy as np

import matplotlib


TOOLS_DIR = Path(__file__).resolve().parent
if str(TOOLS_DIR) not in sys.path:
    sys.path.insert(0, str(TOOLS_DIR))

from view_bin_volume import collapse_extra_axes, parse_film, tone_map_to_srgb_uint8

plt = None


def configure_matplotlib(*, show: bool, backend: str) -> None:
    """Select a Matplotlib backend before importing pyplot."""
    global plt
    if plt is not None:
        return

    if show:
        try:
            matplotlib.use(backend, force=True)
        except Exception as exc:
            print(f"warning: failed to use interactive backend {backend!r}: {exc}")
            print("warning: falling back to Agg; interactive window will not open")
            matplotlib.use("Agg", force=True)
    else:
        matplotlib.use("Agg", force=True)

    import matplotlib.pyplot as pyplot

    plt = pyplot


def choose_visible_voxels(
    rgb_u8: np.ndarray,
    *,
    luminance_floor: float,
    max_points: int,
) -> tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray, np.ndarray]:
    """Return visible voxel coordinates, RGB colors, and luminance values."""
    if rgb_u8.ndim != 4 or rgb_u8.shape[-1] != 3:
        raise ValueError(f"expected (Z,Y,X,3) uint8 volume, got {rgb_u8.shape}")

    z, y, x = np.mgrid[0 : rgb_u8.shape[0], 0 : rgb_u8.shape[1], 0 : rgb_u8.shape[2]]
    rgb_flat = rgb_u8.reshape(-1, 3)
    lum = (
        0.2126 * rgb_flat[:, 0].astype(np.float32)
        + 0.7152 * rgb_flat[:, 1].astype(np.float32)
        + 0.0722 * rgb_flat[:, 2].astype(np.float32)
    )

    keep = lum >= float(luminance_floor)
    kept = int(keep.sum())
    if kept == 0:
        keep = lum > 0
        kept = int(keep.sum())
        print(
            "warning: no voxels above luminance floor; falling back to non-black voxels"
        )

    if kept > max_points:
        idx = np.flatnonzero(keep)
        order = np.argpartition(lum[idx], kept - max_points)[kept - max_points :]
        mask = np.zeros_like(keep)
        mask[idx[order]] = True
        keep = mask
        print(f"info: {kept} visible voxels, keeping brightest {max_points}")

    return (
        x.ravel()[keep],
        y.ravel()[keep],
        z.ravel()[keep],
        rgb_flat[keep],
        lum[keep],
    )


def render_point_cloud(
    rgb_u8: np.ndarray,
    png_out: Path,
    *,
    title: str,
    luminance_floor: float,
    max_points: int,
    point_size: float,
    alpha: float,
    elev: float,
    azim: float,
    show: bool,
) -> None:
    if plt is None:
        raise RuntimeError("configure_matplotlib must be called before rendering")

    xs, ys, zs, colors_u8, lum = choose_visible_voxels(
        rgb_u8,
        luminance_floor=luminance_floor,
        max_points=max_points,
    )
    if xs.size == 0:
        raise ValueError("no visible voxels to render")

    colors = colors_u8.astype(np.float32) / 255.0

    fig = plt.figure(figsize=(10, 10), facecolor="black")
    ax = fig.add_subplot(111, projection="3d", facecolor="black")
    ax.scatter(
        xs,
        ys,
        zs,
        c=colors,
        alpha=alpha,
        s=point_size,
        linewidths=0,
        depthshade=False,
    )

    ax.set_title(title, color="white", pad=18)
    ax.set_xlabel("X", color="white")
    ax.set_ylabel("Y", color="white")
    ax.set_zlabel("Z", color="white")
    ax.tick_params(colors="white")
    ax.view_init(elev=elev, azim=azim)
    ax.set_box_aspect(rgb_u8.shape[:3][::-1])

    for axis in (ax.xaxis, ax.yaxis, ax.zaxis):
        axis.pane.set_facecolor((0.0, 0.0, 0.0, 1.0))
        axis.pane.set_edgecolor((0.35, 0.35, 0.35, 1.0))

    fig.tight_layout()
    fig.savefig(png_out, dpi=180, facecolor="black")
    print(f"Wrote 3D point-cloud preview: {png_out}")
    if show:
        print("Opening interactive Matplotlib 3D window. Close the window to finish.")
        plt.show()
    plt.close(fig)


def save_rgb_slices(rgb_u8: np.ndarray, png_out: Path) -> None:
    if plt is None:
        raise RuntimeError("configure_matplotlib must be called before rendering")

    zc, yc, xc = [s // 2 for s in rgb_u8.shape[:3]]
    fig, axes = plt.subplots(1, 3, figsize=(12, 4), constrained_layout=True)
    fig.patch.set_facecolor("black")

    axes[0].imshow(rgb_u8[zc, :, :, :])
    axes[0].set_title(f"XY slice, z={zc}", color="white")
    axes[1].imshow(rgb_u8[:, yc, :, :], aspect="auto")
    axes[1].set_title(f"XZ slice, y={yc}", color="white")
    axes[2].imshow(rgb_u8[:, :, xc, :], aspect="auto")
    axes[2].set_title(f"YZ slice, x={xc}", color="white")

    for ax in axes:
        ax.set_axis_off()

    fig.savefig(png_out, dpi=180, facecolor="black")
    plt.close(fig)
    print(f"Wrote RGB middle-slice preview: {png_out}")


def default_output_paths(bin_file: Path, png_arg: Optional[Path], slices_arg: Optional[Path]) -> tuple[Path, Path]:
    png = png_arg if png_arg is not None else bin_file.with_suffix(".matplotlib-volume.png")
    slices = slices_arg if slices_arg is not None else bin_file.with_suffix(".matplotlib-slices.png")
    return png, slices


def main(argv: Optional[Sequence[str]] = None) -> None:
    parser = argparse.ArgumentParser(description="Render a Film .bin volume with Matplotlib.")
    parser.add_argument("bin_file", type=Path, help="input .bin file")
    parser.add_argument("--png", type=Path, help="output 3D point-cloud PNG")
    parser.add_argument("--slices", type=Path, help="output middle-slice PNG")
    parser.add_argument(
        "--reduce",
        default="mean",
        choices=["mean", "max", "sum", "first", "last"],
        help="for >3D films, collapse extra leading axes with this reducer",
    )
    parser.add_argument("--exposure", type=float, default=1.0, help="multiplicative exposure")
    parser.add_argument(
        "--tone",
        default="aces",
        choices=["linear", "reinhard", "aces"],
        help="tone mapping operator",
    )
    parser.add_argument("--gamma", type=float, default=2.2, help="output gamma")
    parser.add_argument(
        "--luminance-floor",
        type=float,
        default=8.0,
        help="drop encoded voxels below this Rec.709 luminance threshold",
    )
    parser.add_argument("--max-points", type=int, default=800_000, help="maximum scatter points")
    parser.add_argument(
        "--point-size",
        type=float,
        default=2.0,
        help="scatter marker size; old interactive viewer used s=1",
    )
    parser.add_argument(
        "--alpha",
        type=float,
        default=0.05,
        help="scatter alpha; old interactive viewer used 0.05",
    )
    parser.add_argument("--elev", type=float, default=22.0, help="camera elevation")
    parser.add_argument("--azim", type=float, default=42.0, help="camera azimuth")
    parser.add_argument("--show", action="store_true", help="open an interactive Matplotlib 3D window")
    parser.add_argument(
        "--backend",
        default="TkAgg",
        help="Matplotlib backend to use with --show; default: TkAgg",
    )
    parser.add_argument("--no-slices", action="store_true", help="skip slice PNG")
    args = parser.parse_args(argv)

    configure_matplotlib(show=args.show, backend=args.backend)

    png_out, slices_out = default_output_paths(
        args.bin_file,
        args.png,
        None if args.no_slices else args.slices,
    )

    vol, _spectral, meta = parse_film(args.bin_file)
    rgb_3d = collapse_extra_axes(vol, args.reduce, keep_trailing=4)
    if rgb_3d.ndim != 4 or rgb_3d.shape[-1] < 3:
        raise SystemExit(f"expected RGB volume after reduction, got {rgb_3d.shape}")

    print("Parsed Film .bin")
    print(f"  go dims:        {meta.dims_go}")
    print(f"  numpy shape:    {vol.shape}")
    print(f"  render shape:   {rgb_3d.shape}")
    print(f"  samples:        {meta.samples}")
    print(f"  color space:    {meta.color_space!r}")

    rgb_u8 = tone_map_to_srgb_uint8(
        rgb_3d,
        exposure=args.exposure,
        tone=args.tone,
        gamma=args.gamma,
    )
    visible_pct = float((rgb_u8.any(axis=-1)).mean()) * 100.0
    print(
        f"  sRGB encode:    exposure={args.exposure} tone={args.tone} gamma={args.gamma} "
        f"-> {visible_pct:.2f}% non-black voxels"
    )

    render_point_cloud(
        rgb_u8,
        png_out,
        title=args.bin_file.name,
        luminance_floor=args.luminance_floor,
        max_points=args.max_points,
        point_size=args.point_size,
        alpha=args.alpha,
        elev=args.elev,
        azim=args.azim,
        show=args.show,
    )

    if not args.no_slices:
        save_rgb_slices(rgb_u8, slices_out)


if __name__ == "__main__":
    main()
