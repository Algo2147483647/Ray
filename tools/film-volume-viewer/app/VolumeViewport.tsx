"use client";

import { useEffect, useRef, useState } from "react";
import type { RayFilm, ToneMap } from "./film";

export type VolumeMode = "dvr" | "mip" | "iso";

type VolumeControls = {
  mode: VolumeMode;
  threshold: number;
  softness: number;
  opacity: number;
  steps: number;
  exposure: number;
  toneMap: ToneMap;
  gamma: number;
  linearFilter: boolean;
};

type Props = {
  film: RayFilm | null;
  controls: VolumeControls;
  onError: (message: string) => void;
};

const vertexShader = `#version 300 es
precision highp float;
out vec2 vUv;
void main() {
  vec2 position;
  if (gl_VertexID == 0) position = vec2(-1.0, -1.0);
  else if (gl_VertexID == 1) position = vec2(3.0, -1.0);
  else position = vec2(-1.0, 3.0);
  vUv = position * 0.5 + 0.5;
  gl_Position = vec4(position, 0.0, 1.0);
}`;

const fragmentShader = `#version 300 es
precision highp float;
precision highp sampler3D;

in vec2 vUv;
out vec4 outColor;

uniform sampler3D uVolume;
uniform vec2 uResolution;
uniform vec3 uCamera;
uniform vec3 uForward;
uniform vec3 uRight;
uniform vec3 uUp;
uniform vec3 uBoxScale;
uniform vec3 uDims;
uniform float uTextureScale;
uniform float uThreshold;
uniform float uSoftness;
uniform float uOpacity;
uniform float uExposure;
uniform float uGamma;
uniform int uSteps;
uniform int uMode;
uniform int uToneMap;

vec3 tone(vec3 color) {
  color = max(color * uTextureScale * uExposure, vec3(0.0));
  if (uToneMap == 1) color = color / (vec3(1.0) + color);
  if (uToneMap == 2) {
    color = (color * (2.51 * color + 0.03)) /
      (color * (2.43 * color + 0.59) + 0.14);
  }
  color = clamp(color, 0.0, 1.0);
  return pow(color, vec3(1.0 / max(uGamma, 0.1)));
}

vec2 intersectBox(vec3 ro, vec3 rd, vec3 halfBox) {
  vec3 safeDir = sign(rd) * max(abs(rd), vec3(0.000001));
  vec3 t0 = (-halfBox - ro) / safeDir;
  vec3 t1 = (halfBox - ro) / safeDir;
  vec3 nearV = min(t0, t1);
  vec3 farV = max(t0, t1);
  return vec2(max(max(nearV.x, nearV.y), nearV.z),
              min(min(farV.x, farV.y), farV.z));
}

float densityAt(vec3 uvw) {
  return texture(uVolume, clamp(uvw, 0.0, 1.0)).a;
}

void main() {
  vec2 ndc = vUv * 2.0 - 1.0;
  ndc.x *= uResolution.x / max(1.0, uResolution.y);
  vec3 rayDirection = normalize(
    uForward + ndc.x * 0.58 * uRight + ndc.y * 0.58 * uUp
  );
  vec3 halfBox = 0.5 * uBoxScale;
  vec2 hit = intersectBox(uCamera, rayDirection, halfBox);
  float nearT = max(hit.x, 0.0);
  float farT = hit.y;
  if (farT <= nearT) {
    outColor = vec4(0.0);
    return;
  }

  float stepLength = (farT - nearT) / float(max(uSteps, 1));
  float jitter = fract(sin(dot(gl_FragCoord.xy, vec2(12.9898, 78.233))) * 43758.5453);
  float travel = nearT + jitter * stepLength;
  vec4 accumulated = vec4(0.0);
  float bestDensity = 0.0;
  vec3 bestColor = vec3(0.0);

  for (int index = 0; index < 512; index++) {
    if (index >= uSteps || travel > farT) break;
    vec3 position = uCamera + rayDirection * travel;
    vec3 uvw = position / uBoxScale + 0.5;
    vec4 sampleValue = texture(uVolume, clamp(uvw, 0.0, 1.0));
    float occupancy = smoothstep(
      uThreshold,
      min(1.0, uThreshold + max(uSoftness, 0.0001)),
      sampleValue.a
    );

    if (uMode == 1) {
      if (occupancy > bestDensity) {
        bestDensity = occupancy;
        bestColor = sampleValue.rgb;
      }
    } else if (uMode == 2) {
      if (occupancy > 0.5) {
        vec3 delta = 1.0 / max(uDims, vec3(1.0));
        vec3 gradient = vec3(
          densityAt(uvw + vec3(delta.x, 0.0, 0.0)) - densityAt(uvw - vec3(delta.x, 0.0, 0.0)),
          densityAt(uvw + vec3(0.0, delta.y, 0.0)) - densityAt(uvw - vec3(0.0, delta.y, 0.0)),
          densityAt(uvw + vec3(0.0, 0.0, delta.z)) - densityAt(uvw - vec3(0.0, 0.0, delta.z))
        );
        vec3 normal = normalize(gradient + vec3(0.000001));
        float light = 0.28 + 0.72 * abs(dot(normal, normalize(vec3(0.35, 0.7, 0.55))));
        outColor = vec4(tone(sampleValue.rgb) * light, 1.0);
        return;
      }
    } else {
      float alpha = 1.0 - exp(-occupancy * sampleValue.a * uOpacity * 0.12);
      vec3 color = tone(sampleValue.rgb);
      accumulated.rgb += (1.0 - accumulated.a) * alpha * color;
      accumulated.a += (1.0 - accumulated.a) * alpha;
      if (accumulated.a > 0.985) break;
    }
    travel += stepLength;
  }

  if (uMode == 1) {
    outColor = vec4(tone(bestColor), bestDensity);
  } else if (uMode == 2) {
    outColor = vec4(0.0);
  } else {
    outColor = accumulated;
  }
}`;

function createShader(
  gl: WebGL2RenderingContext,
  type: number,
  source: string,
) {
  const shader = gl.createShader(type);
  if (!shader) throw new Error("无法创建 WebGL shader。");
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    const message = gl.getShaderInfoLog(shader) ?? "未知 shader 编译错误";
    gl.deleteShader(shader);
    throw new Error(message);
  }
  return shader;
}

function normalize(v: [number, number, number]) {
  const length = Math.hypot(v[0], v[1], v[2]) || 1;
  return [v[0] / length, v[1] / length, v[2] / length] as const;
}

function cross(
  a: readonly [number, number, number],
  b: readonly [number, number, number],
) {
  return normalize([
    a[1] * b[2] - a[2] * b[1],
    a[2] * b[0] - a[0] * b[2],
    a[0] * b[1] - a[1] * b[0],
  ]);
}

export function VolumeViewport({ film, controls, onError }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const controlsRef = useRef(controls);
  const [webglReady, setWebglReady] = useState(true);

  useEffect(() => {
    controlsRef.current = controls;
  }, [controls]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || !film) return;
    const gl = canvas.getContext("webgl2", {
      alpha: true,
      antialias: false,
      premultipliedAlpha: true,
    });
    if (!gl) {
      setWebglReady(false);
      onError("浏览器不支持 WebGL2；切片视图仍可使用。");
      return;
    }

    let animation = 0;
    let dragging = false;
    let lastX = 0;
    let lastY = 0;
    let yaw = 0.72;
    let pitch = 0.34;
    let distance = 2.05;

    try {
      const program = gl.createProgram();
      if (!program) throw new Error("无法创建 WebGL program。");
      const vertex = createShader(gl, gl.VERTEX_SHADER, vertexShader);
      const fragment = createShader(gl, gl.FRAGMENT_SHADER, fragmentShader);
      gl.attachShader(program, vertex);
      gl.attachShader(program, fragment);
      gl.linkProgram(program);
      if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
        throw new Error(gl.getProgramInfoLog(program) ?? "WebGL program 链接失败。");
      }

      const texture = gl.createTexture();
      if (!texture) throw new Error("无法创建三维纹理。");
      gl.bindTexture(gl.TEXTURE_3D, texture);
      gl.pixelStorei(gl.UNPACK_ALIGNMENT, 1);
      gl.texParameteri(gl.TEXTURE_3D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
      gl.texParameteri(gl.TEXTURE_3D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
      gl.texParameteri(gl.TEXTURE_3D, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE);
      gl.texImage3D(
        gl.TEXTURE_3D,
        0,
        gl.RGBA8,
        film.shape[0],
        film.shape[1],
        film.shape[2],
        0,
        gl.RGBA,
        gl.UNSIGNED_BYTE,
        film.textureData,
      );

      const uniforms = {
        resolution: gl.getUniformLocation(program, "uResolution"),
        camera: gl.getUniformLocation(program, "uCamera"),
        forward: gl.getUniformLocation(program, "uForward"),
        right: gl.getUniformLocation(program, "uRight"),
        up: gl.getUniformLocation(program, "uUp"),
        boxScale: gl.getUniformLocation(program, "uBoxScale"),
        dims: gl.getUniformLocation(program, "uDims"),
        textureScale: gl.getUniformLocation(program, "uTextureScale"),
        threshold: gl.getUniformLocation(program, "uThreshold"),
        softness: gl.getUniformLocation(program, "uSoftness"),
        opacity: gl.getUniformLocation(program, "uOpacity"),
        exposure: gl.getUniformLocation(program, "uExposure"),
        gamma: gl.getUniformLocation(program, "uGamma"),
        steps: gl.getUniformLocation(program, "uSteps"),
        mode: gl.getUniformLocation(program, "uMode"),
        toneMap: gl.getUniformLocation(program, "uToneMap"),
      };

      const maxDimension = Math.max(...film.shape);
      const boxScale = film.shape.map((extent) => extent / maxDimension) as [
        number,
        number,
        number,
      ];

      const resize = () => {
        const ratio = Math.min(window.devicePixelRatio || 1, 2);
        const width = Math.max(1, Math.floor(canvas.clientWidth * ratio));
        const height = Math.max(1, Math.floor(canvas.clientHeight * ratio));
        if (canvas.width !== width || canvas.height !== height) {
          canvas.width = width;
          canvas.height = height;
        }
      };

      const render = () => {
        resize();
        gl.viewport(0, 0, canvas.width, canvas.height);
        gl.clearColor(0, 0, 0, 0);
        gl.clear(gl.COLOR_BUFFER_BIT);
        gl.enable(gl.BLEND);
        gl.blendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA);
        gl.useProgram(program);

        const camera = [
          distance * Math.cos(pitch) * Math.sin(yaw),
          distance * Math.sin(pitch),
          distance * Math.cos(pitch) * Math.cos(yaw),
        ] as [number, number, number];
        const forward = normalize([-camera[0], -camera[1], -camera[2]]);
        const right = cross(forward, [0, 1, 0]);
        const up = cross(right, forward);
        const current = controlsRef.current;

        gl.activeTexture(gl.TEXTURE0);
        gl.bindTexture(gl.TEXTURE_3D, texture);
        const filter = current.linearFilter ? gl.LINEAR : gl.NEAREST;
        gl.texParameteri(gl.TEXTURE_3D, gl.TEXTURE_MIN_FILTER, filter);
        gl.texParameteri(gl.TEXTURE_3D, gl.TEXTURE_MAG_FILTER, filter);
        gl.uniform2f(uniforms.resolution, canvas.width, canvas.height);
        gl.uniform3fv(uniforms.camera, camera);
        gl.uniform3fv(uniforms.forward, forward);
        gl.uniform3fv(uniforms.right, right);
        gl.uniform3fv(uniforms.up, up);
        gl.uniform3fv(uniforms.boxScale, boxScale);
        gl.uniform3f(uniforms.dims, film.shape[0], film.shape[1], film.shape[2]);
        gl.uniform1f(uniforms.textureScale, film.textureScale);
        gl.uniform1f(uniforms.threshold, current.threshold);
        gl.uniform1f(uniforms.softness, current.softness);
        gl.uniform1f(uniforms.opacity, current.opacity);
        gl.uniform1f(uniforms.exposure, current.exposure);
        gl.uniform1f(uniforms.gamma, current.gamma);
        gl.uniform1i(uniforms.steps, current.steps);
        gl.uniform1i(uniforms.mode, current.mode === "dvr" ? 0 : current.mode === "mip" ? 1 : 2);
        gl.uniform1i(uniforms.toneMap, current.toneMap === "linear" ? 0 : current.toneMap === "reinhard" ? 1 : 2);
        gl.drawArrays(gl.TRIANGLES, 0, 3);
        animation = requestAnimationFrame(render);
      };

      const pointerDown = (event: PointerEvent) => {
        dragging = true;
        lastX = event.clientX;
        lastY = event.clientY;
        canvas.setPointerCapture(event.pointerId);
      };
      const pointerMove = (event: PointerEvent) => {
        if (!dragging) return;
        yaw -= (event.clientX - lastX) * 0.008;
        pitch = Math.max(-1.35, Math.min(1.35, pitch + (event.clientY - lastY) * 0.008));
        lastX = event.clientX;
        lastY = event.clientY;
      };
      const pointerUp = (event: PointerEvent) => {
        dragging = false;
        if (canvas.hasPointerCapture(event.pointerId)) canvas.releasePointerCapture(event.pointerId);
      };
      const wheel = (event: WheelEvent) => {
        event.preventDefault();
        distance = Math.max(1.05, Math.min(4.5, distance * Math.exp(event.deltaY * 0.001)));
      };
      canvas.addEventListener("pointerdown", pointerDown);
      canvas.addEventListener("pointermove", pointerMove);
      canvas.addEventListener("pointerup", pointerUp);
      canvas.addEventListener("pointercancel", pointerUp);
      canvas.addEventListener("wheel", wheel, { passive: false });
      queueMicrotask(() => setWebglReady(true));
      render();

      return () => {
        cancelAnimationFrame(animation);
        canvas.removeEventListener("pointerdown", pointerDown);
        canvas.removeEventListener("pointermove", pointerMove);
        canvas.removeEventListener("pointerup", pointerUp);
        canvas.removeEventListener("pointercancel", pointerUp);
        canvas.removeEventListener("wheel", wheel);
        gl.deleteTexture(texture);
        gl.deleteShader(vertex);
        gl.deleteShader(fragment);
        gl.deleteProgram(program);
      };
    } catch (error) {
      const message = error instanceof Error ? error.message : "未知 WebGL 错误";
      setWebglReady(false);
      onError(`三维视图初始化失败：${message}`);
    }
  }, [film, onError]);

  return (
    <div className="volume-stage">
      <canvas
        ref={canvasRef}
        className="volume-canvas"
        aria-label="可旋转的三维 film 张量体绘制"
      />
      <div className="axis-label axis-x">X</div>
      <div className="axis-label axis-y">Y</div>
      <div className="axis-label axis-z">Z</div>
      <div className="viewport-hint">拖动旋转 · 滚轮缩放</div>
      {!film && <div className="viewport-empty">等待 film 数据</div>}
      {!webglReady && <div className="viewport-empty">WebGL2 不可用</div>}
    </div>
  );
}
