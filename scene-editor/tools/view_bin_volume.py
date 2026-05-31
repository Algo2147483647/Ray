#!/usr/bin/env python3
"""Parse a Ray Film .bin volume file and render it for inspection.

The file format is the one written by engine/model/camera/film.go::SaveToFile:

int64                 samples
int32                 shapeLen
int32 * shapeLen      dims, where dims[0] varies fastest in memory
float64 * prod(dims)*3 RGB payload, channel-major, each channel Fortran-order
int32                 optional colorSpaceLen
byte  * colorSpaceLen optional colorSpace tag
byte  * 4             optional "SPCT" spectral magic

Plotly's volume renderer only supports scalar fields. The default RGB mode
therefore renders a colored point cloud and writes true-color middle slices.
"""

from __future__ import annotations

import argparse
import math
import struct
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional, Sequence

import numpy as np


SPECTRAL_MAGIC = b"SPCT"
CHANNEL_COUNT = 3
F64 = np.dtype("<f8")


@dataclass
class FilmMeta:
    samples: int
    dims_go: list[int]
    shape_zyx: tuple[int, ...]
    channels: int
    payload_offset: int
    payload_bytes: int
    color_space: Optional[str] = None
    spectral_bin_count: int = 0
    spectral_min_nm: float = 0.0
    spectral_max_nm: float = 0.0
    spectral_offset: int = 0
    spectral_bytes: int = 0
    footer_extra: bytes = field(default_factory=bytes)


def parse_film(path: Path) -> tuple[np.ndarray, Optional[np.ndarray], FilmMeta]:
    """Parse a Film .bin file.

    Returns:
        rgb_volume: shape (Z, Y, X, 3), or generally reversed spatial axes + RGB.
        spectral_volume: optional shape (bin_count, Z, Y, X, ...).
        meta: parsed header/footer metadata.
    """
    raw = path.read_bytes()
    size = len(raw)

    if size < 12:
        raise ValueError(f"file too small to be a Film: {size} bytes")

    samples = struct.unpack_from("<q", raw, 0)[0]
    (shape_len,) = struct.unpack_from("<i", raw, 8)
    if shape_len <= 0 or shape_len > 8:
        raise ValueError(f"unreasonable shapeLen={shape_len}; not a Film .bin?")

    header_end = 12 + 4 * shape_len
    if header_end > size:
        raise ValueError("file truncated inside shape header")

    dims_go = list(struct.unpack_from(f"<{shape_len}i", raw, 12))
    if any(d <= 0 for d in dims_go):
        raise ValueError(f"non-positive dim in shape: {dims_go}")

    voxels = math.prod(dims_go)
    payload_bytes = voxels * CHANNEL_COUNT * F64.itemsize
    payload_end = header_end + payload_bytes
    if payload_end > size:
        raise ValueError(
            f"payload truncated: header_end={header_end}, "
            f"expected payload {payload_bytes} bytes, file size {size}"
        )

    flat = np.frombuffer(raw, dtype=F64, count=voxels * CHANNEL_COUNT, offset=header_end)
    vol = flat.reshape((*dims_go, CHANNEL_COUNT), order="F")
    spatial = list(range(len(dims_go)))
    vol = np.transpose(vol, tuple(reversed(spatial)) + (len(dims_go),))
    shape_zyx = tuple(reversed(dims_go))

    cursor = payload_end
    color_space: Optional[str] = None

    if cursor + 4 <= size:
        (cs_len,) = struct.unpack_from("<i", raw, cursor)
        if 0 < cs_len <= size - cursor - 4:
            tag_bytes = raw[cursor + 4 : cursor + 4 + cs_len]
            if tag_bytes != SPECTRAL_MAGIC:
                try:
                    color_space = tag_bytes.decode("utf-8")
                except UnicodeDecodeError:
                    color_space = None
                cursor += 4 + cs_len
        elif cs_len == 0:
            cursor += 4

    spectral_vol: Optional[np.ndarray] = None
    spectral_offset = 0
    spectral_bytes = 0
    bin_count = 0
    min_nm = 0.0
    max_nm = 0.0

    if cursor + 4 <= size and raw[cursor : cursor + 4] == SPECTRAL_MAGIC:
        spec_header = cursor + 4
        if spec_header + 20 <= size:
            (bin_count,) = struct.unpack_from("<i", raw, spec_header)
            (min_nm,) = struct.unpack_from("<d", raw, spec_header + 4)
            (max_nm,) = struct.unpack_from("<d", raw, spec_header + 12)
            spec_payload = spec_header + 20
            spec_count = voxels * max(bin_count, 0)
            spec_bytes = spec_count * F64.itemsize
            if bin_count > 0 and spec_payload + spec_bytes <= size:
                spec_flat = np.frombuffer(raw, dtype=F64, count=spec_count, offset=spec_payload)
                spec_vol = spec_flat.reshape((*dims_go, bin_count), order="F")
                spec_vol = np.transpose(spec_vol, tuple(reversed(spatial)) + (len(dims_go),))
                spectral_vol = np.moveaxis(spec_vol, -1, 0)
                spectral_offset = spec_payload
                spectral_bytes = spec_bytes
                cursor = spec_payload + spec_bytes
            else:
                cursor = spec_header

    meta = FilmMeta(
        samples=samples,
        dims_go=dims_go,
        shape_zyx=shape_zyx,
        channels=CHANNEL_COUNT,
        payload_offset=header_end,
        payload_bytes=payload_bytes,
        color_space=color_space,
        spectral_bin_count=bin_count,
        spectral_min_nm=min_nm,
        spectral_max_nm=max_nm,
        spectral_offset=spectral_offset,
        spectral_bytes=spectral_bytes,
        footer_extra=raw[cursor:],
    )
    return vol, spectral_vol, meta


def scalarize(vol: np.ndarray, component: str) -> np.ndarray:
    """Convert a multi-channel volume to scalar data for volume rendering."""
    if vol.ndim == 3:
        return vol.astype(np.float32, copy=False)

    c = vol.shape[-1]
    comp = component.lower()
    data = vol.astype(np.float32, copy=False)

    if comp in {"0", "1", "2", "3"}:
        idx = int(comp)
        if idx >= c:
            raise ValueError(f"component {idx} requested, but volume has only {c} channels")
        return data[..., idx]
    if comp in {"r", "red"}:
        return data[..., 0]
    if comp in {"g", "green"}:
        return data[..., min(1, c - 1)]
    if comp in {"b", "blue"}:
        return data[..., min(2, c - 1)]
    if comp == "mean":
        return data.mean(axis=-1)
    if comp == "max":
        return data.max(axis=-1)
    if comp == "magnitude":
        return np.linalg.norm(data, axis=-1)
    if comp == "luminance":
        if c >= 3:
            return 0.2126 * data[..., 0] + 0.7152 * data[..., 1] + 0.0722 * data[..., 2]
        return data.mean(axis=-1)
    raise ValueError("unknown component: use luminance, magnitude, mean, max, r/g/b, or 0/1/2/3")


def _aces_tone_map(v: np.ndarray) -> np.ndarray:
    return (v * (2.51 * v + 0.03)) / (v * (2.43 * v + 0.59) + 0.14)


def tone_map_to_srgb_uint8(
    rgb: np.ndarray,
    *,
    exposure: float,
    tone: str,
    gamma: float,
) -> np.ndarray:
    """Apply exposure, tone mapping, clamp, and gamma encode per channel."""
    if rgb.ndim < 1 or rgb.shape[-1] < 3:
        raise ValueError(f"tone_map_to_srgb expects an RGB tensor, got shape {rgb.shape}")

    v = rgb[..., :3].astype(np.float32, copy=True)
    v[~np.isfinite(v) | (v <= 0)] = 0.0

    if exposure != 1.0:
        v *= float(exposure)

    if tone == "reinhard":
        v = v / (1.0 + v)
    elif tone == "aces":
        v = _aces_tone_map(v)
    elif tone != "linear":
        raise ValueError(f"unknown tone mapping: {tone!r} (use linear/reinhard/aces)")

    np.clip(v, 0.0, 1.0, out=v)
    if gamma > 0 and gamma != 1.0:
        v = np.power(v, 1.0 / float(gamma), dtype=np.float32)
        np.clip(v, 0.0, 1.0, out=v)

    return np.rint(v * 255.0).astype(np.uint8)


def robust_limits(values: np.ndarray, low_pct: float, high_pct: float) -> tuple[float, float]:
    finite = values[np.isfinite(values)]
    finite = finite[finite != 0] if np.any(finite != 0) else finite
    if finite.size == 0:
        return 0.0, 1.0

    lo, hi = np.percentile(finite, [low_pct, high_pct])
    if not np.isfinite(lo) or not np.isfinite(hi) or lo == hi:
        lo, hi = float(np.nanmin(finite)), float(np.nanmax(finite))
    if lo == hi:
        hi = lo + 1.0
    return float(lo), float(hi)


def downsample_to_budget(values: np.ndarray, max_voxels: int) -> tuple[np.ndarray, int]:
    n = int(np.prod(values.shape))
    if n <= max_voxels:
        return values, 1
    step = int(math.ceil((n / max_voxels) ** (1 / 3)))
    return values[::step, ::step, ::step], step


def render_plotly_volume(
    scalar: np.ndarray,
    html_out: Path,
    *,
    title: str,
    max_voxels: int,
    low_pct: float,
    high_pct: float,
    opacity: float,
    surface_count: int,
    show: bool,
) -> None:
    try:
        import plotly.graph_objects as go
    except ImportError as exc:
        raise SystemExit("Plotly is required. Install with: pip install plotly") from exc

    if scalar.ndim != 3:
        raise ValueError(f"render_plotly_volume expects a 3D scalar field, got shape {scalar.shape}")

    scalar = np.nan_to_num(scalar.astype(np.float32, copy=False), copy=False)
    scalar_ds, step = downsample_to_budget(scalar, max_voxels)
    z, y, x = np.mgrid[0 : scalar_ds.shape[0], 0 : scalar_ds.shape[1], 0 : scalar_ds.shape[2]]
    isomin, isomax = robust_limits(scalar_ds, low_pct, high_pct)

    fig = go.Figure(
        data=go.Volume(
            x=(x.ravel(order="C") * step),
            y=(y.ravel(order="C") * step),
            z=(z.ravel(order="C") * step),
            value=scalar_ds.ravel(order="C"),
            isomin=isomin,
            isomax=isomax,
            opacity=opacity,
            surface_count=surface_count,
            caps=dict(x_show=False, y_show=False, z_show=False),
            colorbar=dict(title="scalar"),
        )
    )
    fig.update_layout(
        title=title,
        scene=dict(xaxis_title="X", yaxis_title="Y", zaxis_title="Z", aspectmode="data"),
        margin=dict(l=0, r=0, t=45, b=0),
    )
    fig.write_html(str(html_out), include_plotlyjs="cdn")
    print(f"Wrote interactive volume: {html_out}")
    if show:
        fig.show()


def render_plotly_rgb_pointcloud(
    rgb_u8: np.ndarray,
    html_out: Path,
    *,
    title: str,
    max_voxels: int,
    luminance_floor: float,
    point_size: float,
    show: bool,
) -> None:
    """Interactive 3D scatter where each voxel keeps its encoded RGB color."""
    try:
        import plotly.graph_objects as go
    except ImportError as exc:
        raise SystemExit("Plotly is required. Install with: pip install plotly") from exc

    if rgb_u8.ndim != 4 or rgb_u8.shape[-1] != 3:
        raise ValueError(f"render_plotly_rgb_pointcloud expects (Z,Y,X,3) uint8, got {rgb_u8.shape}")
    if rgb_u8.dtype != np.uint8:
        raise ValueError(f"expected uint8 RGB, got dtype {rgb_u8.dtype}")

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
        print(
            f"warning: no voxels above luminance_floor={luminance_floor}; "
            "lower --luminance-floor or tweak --exposure/--tone."
        )
    if kept > max_voxels:
        idx = np.flatnonzero(keep)
        order = np.argpartition(lum[idx], kept - max_voxels)[kept - max_voxels :]
        mask = np.zeros_like(keep)
        mask[idx[order]] = True
        keep = mask
        print(f"info: {kept} voxels above floor, keeping brightest {max_voxels}.")

    xs = x.ravel()[keep]
    ys = y.ravel()[keep]
    zs = z.ravel()[keep]
    colors = [f"rgb({r},{g},{b})" for r, g, b in rgb_flat[keep]]

    fig = go.Figure(
        data=go.Scatter3d(
            x=xs,
            y=ys,
            z=zs,
            mode="markers",
            marker=dict(size=point_size, color=colors, opacity=0.7, line=dict(width=0)),
            hoverinfo="skip",
        )
    )
    fig.update_layout(
        title=title,
        paper_bgcolor="black",
        font=dict(color="white"),
        scene=dict(
            xaxis_title="X",
            yaxis_title="Y",
            zaxis_title="Z",
            aspectmode="data",
            bgcolor="black",
            xaxis=dict(color="white"),
            yaxis=dict(color="white"),
            zaxis=dict(color="white"),
        ),
        margin=dict(l=0, r=0, t=45, b=0),
    )
    fig.write_html(str(html_out), include_plotlyjs="cdn")
    print(f"Wrote interactive RGB point cloud: {html_out}")
    if show:
        fig.show()


def save_rgb_slices(rgb_u8: np.ndarray, png_out: Path) -> None:
    """Save true-color RGB slices through the middle of the volume."""
    try:
        import matplotlib.pyplot as plt
    except ImportError as exc:
        raise SystemExit("Matplotlib is required for slices. Install with: pip install matplotlib") from exc

    if rgb_u8.ndim != 4 or rgb_u8.shape[-1] != 3:
        raise ValueError(f"save_rgb_slices expects (Z,Y,X,3) uint8, got shape {rgb_u8.shape}")

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


def save_middle_slices(scalar: np.ndarray, png_out: Path) -> None:
    try:
        import matplotlib.pyplot as plt
    except ImportError as exc:
        raise SystemExit("Matplotlib is required for slices. Install with: pip install matplotlib") from exc

    if scalar.ndim != 3:
        raise ValueError(f"save_middle_slices expects a 3D field, got shape {scalar.shape}")

    zc, yc, xc = [s // 2 for s in scalar.shape]
    lo, hi = robust_limits(scalar, 1.0, 99.5)
    fig, axes = plt.subplots(1, 3, figsize=(12, 4), constrained_layout=True)
    axes[0].imshow(scalar[zc, :, :], cmap="gray", vmin=lo, vmax=hi)
    axes[0].set_title(f"XY slice, z={zc}")
    axes[1].imshow(scalar[:, yc, :], cmap="gray", vmin=lo, vmax=hi, aspect="auto")
    axes[1].set_title(f"XZ slice, y={yc}")
    axes[2].imshow(scalar[:, :, xc], cmap="gray", vmin=lo, vmax=hi, aspect="auto")
    axes[2].set_title(f"YZ slice, x={xc}")
    for ax in axes:
        ax.set_axis_off()
    fig.savefig(png_out, dpi=180)
    plt.close(fig)
    print(f"Wrote middle-slice preview: {png_out}")


def collapse_extra_axes(field: np.ndarray, reduce: str, keep_trailing: int) -> np.ndarray:
    """Reduce leading axes until exactly keep_trailing axes remain."""
    while field.ndim > keep_trailing:
        if reduce == "mean":
            field = field.mean(axis=0)
        elif reduce == "max":
            field = field.max(axis=0)
        elif reduce == "sum":
            field = field.sum(axis=0)
        elif reduce == "first":
            field = field[0]
        elif reduce == "last":
            field = field[-1]
        else:
            raise ValueError(f"unknown reduce mode: {reduce}")
    return field


def collapse_to_3d(scalar: np.ndarray, reduce: str) -> np.ndarray:
    return collapse_extra_axes(scalar, reduce, keep_trailing=3)


def default_output_paths(bin_file: Path, html_arg: Optional[Path], slices_arg: Optional[Path]) -> tuple[Path, Optional[Path]]:
    html = html_arg if html_arg is not None else bin_file.with_suffix(".volume.html")
    slices = slices_arg
    if slices is None:
        slices = bin_file.with_suffix(".slices.png")
    return html, slices


def main(argv: Optional[Sequence[str]] = None) -> None:
    parser = argparse.ArgumentParser(description="Parse .bin volume and render a 3D volume preview.")
    parser.add_argument("bin_file", type=Path, help="input .bin file")
    parser.add_argument(
        "--mode",
        default="rgb",
        choices=["rgb", "scalar"],
        help="rgb: color-preserving point cloud; scalar: Plotly volume render",
    )
    parser.add_argument(
        "--component",
        default="luminance",
        help="scalar mode only: luminance, magnitude, mean, max, r/g/b, or channel index",
    )
    parser.add_argument(
        "--reduce",
        default="mean",
        choices=["mean", "max", "sum", "first", "last"],
        help="for >3D films, collapse the slowest extra axes with this reducer",
    )
    parser.add_argument("--html", type=Path, help="output interactive HTML")
    parser.add_argument("--slices", type=Path, help="output PNG with three middle slices")
    parser.add_argument("--no-slices", action="store_true", help="do not write middle-slice PNG")
    parser.add_argument("--max-voxels", type=int, default=800_000, help="downsample/point budget")
    parser.add_argument("--low-pct", type=float, default=2.0, help="scalar mode: low percentile")
    parser.add_argument("--high-pct", type=float, default=99.7, help="scalar mode: high percentile")
    parser.add_argument("--opacity", type=float, default=0.08, help="scalar mode: Plotly volume opacity")
    parser.add_argument("--surface-count", type=int, default=18, help="scalar mode: isosurface count")
    parser.add_argument("--exposure", type=float, default=1.0, help="rgb mode: multiplicative exposure")
    parser.add_argument(
        "--tone",
        default="aces",
        choices=["linear", "reinhard", "aces"],
        help="rgb mode: tone mapping operator",
    )
    parser.add_argument("--gamma", type=float, default=2.2, help="rgb mode: output gamma")
    parser.add_argument(
        "--luminance-floor",
        type=float,
        default=8.0,
        help="rgb mode: drop voxels whose encoded luminance is below this 0..255 threshold",
    )
    parser.add_argument("--point-size", type=float, default=2.5, help="rgb mode: scatter point size")
    parser.add_argument("--no-show", action="store_true", help="write files only; do not open viewer")
    args = parser.parse_args(argv)

    html_out, slices_out = default_output_paths(
        args.bin_file,
        args.html,
        None if args.no_slices else args.slices,
    )

    vol, _spectral, meta = parse_film(args.bin_file)

    print("Parsed Film .bin")
    print(f"  go dims:        {meta.dims_go}  # dims[0] varies fastest in file")
    print(f"  numpy shape:    {vol.shape}  # (slowest spatial, ..., fastest spatial, channels)")
    print(f"  channels:       {meta.channels}")
    print(f"  samples:        {meta.samples}")
    print(f"  payload offset: {meta.payload_offset} bytes")
    print(f"  payload size:   {meta.payload_bytes} bytes")
    print(f"  color space:    {meta.color_space!r}")
    if meta.spectral_bin_count:
        print(
            f"  spectral bins:  {meta.spectral_bin_count} "
            f"({meta.spectral_min_nm:g}..{meta.spectral_max_nm:g} nm), "
            f"{meta.spectral_bytes} bytes at offset {meta.spectral_offset}"
        )
    if meta.footer_extra:
        print(f"  extra footer:   {len(meta.footer_extra)} bytes (unparsed)")

    if args.mode == "scalar":
        scalar = scalarize(vol, args.component)
        scalar3d = collapse_to_3d(scalar, args.reduce)
        print(f"  scalar shape:   {scalar.shape}")
        print(f"  render shape:   {scalar3d.shape}  # collapsed via reduce={args.reduce}")
        finite = scalar3d[np.isfinite(scalar3d)]
        if finite.size:
            print(f"  scalar range:   min={float(finite.min()):.6g}, max={float(finite.max()):.6g}")

        render_plotly_volume(
            scalar3d,
            html_out,
            title=f"{args.bin_file.name} | {args.component}",
            max_voxels=args.max_voxels,
            low_pct=args.low_pct,
            high_pct=args.high_pct,
            opacity=args.opacity,
            surface_count=args.surface_count,
            show=not args.no_show,
        )
        if slices_out:
            save_middle_slices(scalar3d, slices_out)
        return

    rgb_3d = collapse_extra_axes(vol, args.reduce, keep_trailing=4)
    if rgb_3d.ndim != 4 or rgb_3d.shape[-1] < 3:
        raise SystemExit(
            f"rgb mode needs an (Z,Y,X,>=3) volume after reduce, got shape {rgb_3d.shape}; "
            "use --mode scalar for non-RGB data"
        )

    print(f"  render shape:   {rgb_3d.shape}  # collapsed via reduce={args.reduce}")
    raw_lin = rgb_3d[..., :3]
    finite_mask = np.isfinite(raw_lin)
    if finite_mask.any():
        finite_vals = raw_lin[finite_mask]
        print(f"  linear range:   min={float(finite_vals.min()):.6g}, max={float(finite_vals.max()):.6g}")

    rgb_u8 = tone_map_to_srgb_uint8(
        rgb_3d,
        exposure=args.exposure,
        tone=args.tone,
        gamma=args.gamma,
    )
    nonzero_pct = float((rgb_u8.any(axis=-1)).mean()) * 100
    print(
        f"  sRGB encode:    exposure={args.exposure} tone={args.tone} gamma={args.gamma} "
        f"-> {nonzero_pct:.2f}% non-black voxels"
    )

    render_plotly_rgb_pointcloud(
        rgb_u8,
        html_out,
        title=f"{args.bin_file.name} | rgb (tone={args.tone}, exposure={args.exposure}, gamma={args.gamma})",
        max_voxels=args.max_voxels,
        luminance_floor=args.luminance_floor,
        point_size=args.point_size,
        show=not args.no_show,
    )
    if slices_out:
        save_rgb_slices(rgb_u8, slices_out)


if __name__ == "__main__":
    main()
