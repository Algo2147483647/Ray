#!/usr/bin/env python3
"""
Parse a structured .bin volume file and visualize it as an interactive 3D volume.

This script auto-detects the format used by the uploaded sample:
  header: 6 little-endian uint32 values
          [unknown0, unknown1, channels, x, y, z]
  payload: x*y*z*channels float64 values
  footer: optional uint32 string length + ASCII tag, e.g. "linear_srgb"

For multi-channel data, Plotly's volume renderer needs a scalar field, so the
script converts RGB/vector channels to luminance, magnitude, mean, max, or a
selected channel.
"""

from __future__ import annotations

import argparse
import math
import struct
from dataclasses import dataclass
from pathlib import Path
from typing import Optional, Sequence

import numpy as np


DTYPE_MAP = {
    "float64": np.dtype("<f8"),
    "f64": np.dtype("<f8"),
    "double": np.dtype("<f8"),
    "float32": np.dtype("<f4"),
    "f32": np.dtype("<f4"),
    "uint8": np.dtype("u1"),
    "u8": np.dtype("u1"),
    "uint16": np.dtype("<u2"),
    "u16": np.dtype("<u2"),
    "int16": np.dtype("<i2"),
    "i16": np.dtype("<i2"),
}


@dataclass
class VolumeMeta:
    shape_zyx: tuple[int, int, int]
    channels: int
    dtype: np.dtype
    offset: int
    payload_bytes: int
    footer_bytes: bytes
    footer_text: Optional[str]
    raw_header_u32: Optional[list[int]]


def parse_shape(text: str) -> tuple[int, int, int]:
    """Parse z,y,x or x,y,z shape text. The script treats it as z,y,x."""
    parts = [int(p.strip()) for p in text.replace("x", ",").split(",")]
    if len(parts) != 3 or any(v <= 0 for v in parts):
        raise ValueError("shape must contain exactly three positive integers, e.g. 100,100,100")
    return tuple(parts)  # z, y, x


def decode_footer(footer: bytes) -> Optional[str]:
    """Decode footer of form uint32 length + ASCII/UTF-8 string when present."""
    if len(footer) >= 4:
        n = struct.unpack("<I", footer[:4])[0]
        if 0 < n <= len(footer) - 4:
            try:
                return footer[4 : 4 + n].decode("utf-8", errors="replace")
            except Exception:
                return None
    try:
        text = footer.decode("utf-8", errors="ignore").strip("\x00\r\n\t ")
        return text or None
    except Exception:
        return None


def infer_layout(path: Path, dtype: np.dtype) -> VolumeMeta:
    """Infer the uploaded sample's structured layout."""
    size = path.stat().st_size
    with path.open("rb") as f:
        first24 = f.read(24)

    if len(first24) == 24:
        header = list(struct.unpack("<6I", first24))
        channels = int(header[2])
        x, y, z = map(int, header[3:6])
        if channels in (1, 2, 3, 4) and x > 0 and y > 0 and z > 0:
            payload_bytes = x * y * z * channels * dtype.itemsize
            offset = 24
            if offset + payload_bytes <= size:
                with path.open("rb") as f:
                    f.seek(offset + payload_bytes)
                    footer = f.read()
                return VolumeMeta(
                    shape_zyx=(z, y, x),
                    channels=channels,
                    dtype=dtype,
                    offset=offset,
                    payload_bytes=payload_bytes,
                    footer_bytes=footer,
                    footer_text=decode_footer(footer),
                    raw_header_u32=header,
                )

    raise ValueError(
        "Could not infer layout. Please provide --shape, --channels, --dtype and --offset."
    )


def read_volume(
    path: Path,
    *,
    shape_zyx: Optional[tuple[int, int, int]] = None,
    channels: Optional[int] = None,
    dtype: np.dtype = np.dtype("<f8"),
    offset: Optional[int] = None,
    order: str = "C",
) -> tuple[np.ndarray, VolumeMeta]:
    """Read volume as ndarray shaped (z, y, x[, c])."""
    path = Path(path)
    if shape_zyx is None or channels is None or offset is None:
        meta = infer_layout(path, dtype)
        shape_zyx = meta.shape_zyx if shape_zyx is None else shape_zyx
        channels = meta.channels if channels is None else channels
        offset = meta.offset if offset is None else offset
    else:
        payload_bytes = int(np.prod(shape_zyx)) * channels * dtype.itemsize
        file_size = path.stat().st_size
        footer = b""
        if offset + payload_bytes <= file_size:
            with path.open("rb") as f:
                f.seek(offset + payload_bytes)
                footer = f.read()
        meta = VolumeMeta(
            shape_zyx=shape_zyx,
            channels=channels,
            dtype=dtype,
            offset=offset,
            payload_bytes=payload_bytes,
            footer_bytes=footer,
            footer_text=decode_footer(footer),
            raw_header_u32=None,
        )

    count = int(np.prod(shape_zyx)) * int(channels)
    with path.open("rb") as f:
        f.seek(offset)
        arr = np.fromfile(f, dtype=dtype, count=count)
    if arr.size != count:
        raise ValueError(f"Not enough payload data: expected {count} values, got {arr.size}.")

    if channels == 1:
        vol = arr.reshape(shape_zyx, order=order)
    else:
        vol = arr.reshape((*shape_zyx, channels), order=order)
    return vol, meta


def scalarize(vol: np.ndarray, component: str) -> np.ndarray:
    """Convert volume or multi-channel volume to scalar data for volume rendering."""
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
            # Rec. 709 / sRGB luminance coefficients.
            return 0.2126 * data[..., 0] + 0.7152 * data[..., 1] + 0.0722 * data[..., 2]
        return data.mean(axis=-1)
    raise ValueError("unknown component: use luminance, magnitude, mean, max, r/g/b, or 0/1/2/3")


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

    scalar = np.nan_to_num(scalar.astype(np.float32, copy=False), copy=False)
    scalar_ds, step = downsample_to_budget(scalar, max_voxels)
    z, y, x = np.mgrid[
        0 : scalar_ds.shape[0],
        0 : scalar_ds.shape[1],
        0 : scalar_ds.shape[2],
    ]
    value = scalar_ds.ravel(order="C")
    isomin, isomax = robust_limits(scalar_ds, low_pct, high_pct)

    fig = go.Figure(
        data=go.Volume(
            x=(x.ravel(order="C") * step),
            y=(y.ravel(order="C") * step),
            z=(z.ravel(order="C") * step),
            value=value,
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
        scene=dict(
            xaxis_title="X",
            yaxis_title="Y",
            zaxis_title="Z",
            aspectmode="data",
        ),
        margin=dict(l=0, r=0, t=45, b=0),
    )
    fig.write_html(str(html_out), include_plotlyjs="cdn")
    print(f"Wrote interactive volume: {html_out}")
    if show:
        fig.show()


def save_middle_slices(scalar: np.ndarray, png_out: Path) -> None:
    try:
        import matplotlib.pyplot as plt
    except ImportError as exc:
        raise SystemExit("Matplotlib is required for slices. Install with: pip install matplotlib") from exc

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


def main(argv: Optional[Sequence[str]] = None) -> None:
    parser = argparse.ArgumentParser(description="Parse .bin volume and render a 3D stacked volume image.")
    parser.add_argument("bin_file", type=Path, help="input .bin file")
    parser.add_argument("--dtype", default="float64", choices=sorted(DTYPE_MAP), help="payload dtype")
    parser.add_argument("--shape", help="manual shape as z,y,x, e.g. 100,100,100; auto-detected when omitted")
    parser.add_argument("--channels", type=int, help="number of channels; auto-detected when omitted")
    parser.add_argument("--offset", type=int, help="payload byte offset; auto-detected when omitted")
    parser.add_argument("--order", default="C", choices=["C", "F"], help="reshape memory order")
    parser.add_argument(
        "--component",
        default="luminance",
        help="for multi-channel data: luminance, magnitude, mean, max, r/g/b, or channel index 0/1/2/3",
    )
    parser.add_argument("--html", type=Path, default=Path("volume_render.html"), help="output interactive HTML")
    parser.add_argument("--slices", type=Path, help="optional output PNG with three middle slices")
    parser.add_argument("--max-voxels", type=int, default=800_000, help="downsample budget for 3D rendering")
    parser.add_argument("--low-pct", type=float, default=2.0, help="low percentile for isomin")
    parser.add_argument("--high-pct", type=float, default=99.7, help="high percentile for isomax")
    parser.add_argument("--opacity", type=float, default=0.08, help="Plotly volume opacity")
    parser.add_argument("--surface-count", type=int, default=18, help="number of semi-transparent isosurfaces")
    parser.add_argument("--no-show", action="store_true", help="write HTML only; do not open viewer")
    args = parser.parse_args(argv)

    dtype = DTYPE_MAP[args.dtype]
    shape_zyx = parse_shape(args.shape) if args.shape else None

    vol, meta = read_volume(
        args.bin_file,
        shape_zyx=shape_zyx,
        channels=args.channels,
        dtype=dtype,
        offset=args.offset,
        order=args.order,
    )
    scalar = scalarize(vol, args.component)

    print("Parsed volume")
    print(f"  ndarray shape: {vol.shape}  # z,y,x[,channels]")
    print(f"  scalar shape:  {scalar.shape}")
    print(f"  dtype:         {meta.dtype}")
    print(f"  offset:        {meta.offset} bytes")
    print(f"  payload:       {meta.payload_bytes} bytes")
    print(f"  footer bytes:  {len(meta.footer_bytes)}")
    if meta.raw_header_u32 is not None:
        print(f"  header uint32: {meta.raw_header_u32}")
    if meta.footer_text:
        print(f"  footer text:   {meta.footer_text}")
    print(f"  scalar range:  min={np.nanmin(scalar):.6g}, max={np.nanmax(scalar):.6g}")

    render_plotly_volume(
        scalar,
        args.html,
        title=f"{args.bin_file.name} | {args.component}",
        max_voxels=args.max_voxels,
        low_pct=args.low_pct,
        high_pct=args.high_pct,
        opacity=args.opacity,
        surface_count=args.surface_count,
        show=not args.no_show,
    )
    if args.slices:
        save_middle_slices(scalar, args.slices)


if __name__ == "__main__":
    main()
