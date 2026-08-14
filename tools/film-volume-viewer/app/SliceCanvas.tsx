"use client";

import { useEffect, useRef } from "react";
import { displayChannel, type RayFilm, type ToneMap } from "./film";

export type SlicePlane = "xy" | "xz" | "yz";

type Props = {
  film: RayFilm;
  plane: SlicePlane;
  index: number;
  exposure: number;
  toneMap: ToneMap;
  gamma: number;
};

const planeLabels: Record<SlicePlane, [string, string, string]> = {
  xy: ["X", "Y", "Z"],
  xz: ["X", "Z", "Y"],
  yz: ["Y", "Z", "X"],
};

export function SliceCanvas({
  film,
  plane,
  index,
  exposure,
  toneMap,
  gamma,
}: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [horizontal, vertical, fixed] = planeLabels[plane];

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const [width, height, depth] = film.shape;
    const outputWidth = plane === "yz" ? height : width;
    const outputHeight = plane === "xy" ? height : depth;
    canvas.width = outputWidth;
    canvas.height = outputHeight;
    const context = canvas.getContext("2d");
    if (!context) return;
    const image = context.createImageData(outputWidth, outputHeight);

    for (let row = 0; row < outputHeight; row += 1) {
      for (let column = 0; column < outputWidth; column += 1) {
        let x = column;
        let y = outputHeight - 1 - row;
        let z = index;
        if (plane === "xz") {
          y = index;
          z = outputHeight - 1 - row;
        } else if (plane === "yz") {
          x = index;
          y = column;
          z = outputHeight - 1 - row;
        }
        const sourceIndex = x + y * width + z * width * height;
        const targetIndex = (column + row * outputWidth) * 4;
        for (let channel = 0; channel < 3; channel += 1) {
          image.data[targetIndex + channel] = Math.round(
            displayChannel(
              film.channels[channel][sourceIndex],
              exposure,
              toneMap,
              gamma,
            ) * 255,
          );
        }
        image.data[targetIndex + 3] = 255;
      }
    }
    context.putImageData(image, 0, 0);
  }, [film, plane, index, exposure, toneMap, gamma]);

  return (
    <figure className="slice-card">
      <figcaption>
        <span>{plane.toUpperCase()} 切片</span>
        <span className="slice-coordinate">{fixed} = {index}</span>
      </figcaption>
      <div className="slice-frame">
        <canvas ref={canvasRef} aria-label={`${plane.toUpperCase()} 张量切片`} />
        <span className="slice-axis slice-axis-h">{horizontal}</span>
        <span className="slice-axis slice-axis-v">{vertical}</span>
      </div>
    </figure>
  );
}
