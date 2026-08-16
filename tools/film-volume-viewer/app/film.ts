export type FilmColorSpace = "linear_srgb" | "xyz" | "acescg";

export type FilmStats = {
  min: number;
  max: number;
  finiteValues: number;
  nonFiniteValues: number;
  occupiedVoxels: number;
};

export type RayFilm = {
  name: string;
  samples: bigint;
  shape: [number, number, number];
  sourceRank: number;
  sourceShape: number[];
  voxelCount: number;
  colorSpace: FilmColorSpace;
  channels: [Float32Array, Float32Array, Float32Array];
  textureData: Uint8Array;
  textureScale: number;
  stats: FilmStats;
  spectralBins: number;
  spectralRange: [number, number] | null;
};

const MAX_VOXELS = 64 * 1024 * 1024;
const UTF8 = new TextDecoder();
const FILM_MAGIC = [0x52, 0x41, 0x59, 0x46, 0x49, 0x4c, 0x4d, 0x00];
const FILM_VERSION = 2;

class FilmReader {
  private readonly view: DataView;
  offset = 0;

  constructor(buffer: ArrayBuffer) {
    this.view = new DataView(buffer);
  }

  get remaining() {
    return this.view.byteLength - this.offset;
  }

  private require(bytes: number, label: string) {
    if (bytes < 0 || this.remaining < bytes) {
      throw new Error(`文件在读取 ${label} 时提前结束。`);
    }
  }

  uint32(label: string) {
    this.require(4, label);
    const value = this.view.getUint32(this.offset, true);
    this.offset += 4;
    return value;
  }

  uint64(label: string) {
    this.require(8, label);
    const value = this.view.getBigUint64(this.offset, true);
    this.offset += 8;
    return value;
  }

  int64(label: string) {
    this.require(8, label);
    const value = this.view.getBigInt64(this.offset, true);
    this.offset += 8;
    return value;
  }

  float64(label: string) {
    this.require(8, label);
    const value = this.view.getFloat64(this.offset, true);
    this.offset += 8;
    return value;
  }

  bytes(length: number, label: string) {
    this.require(length, label);
    const value = new Uint8Array(
      this.view.buffer,
      this.view.byteOffset + this.offset,
      length,
    );
    this.offset += length;
    return value;
  }
}

function checkedProduct(shape: number[]) {
  let total = 1;
  for (const extent of shape) {
    if (!Number.isInteger(extent) || extent <= 0) {
      throw new Error(`张量维度必须是正整数，收到 ${extent}。`);
    }
    total *= extent;
    if (!Number.isSafeInteger(total) || total > MAX_VOXELS) {
      throw new Error("张量过大；当前查看器最多接受 67,108,864 个体素。");
    }
  }
  return total;
}

function toLinearSRGB(
  a: number,
  b: number,
  c: number,
  colorSpace: FilmColorSpace,
) {
  if (colorSpace === "xyz") {
    return [
      3.2404542 * a - 1.5371385 * b - 0.4985314 * c,
      -0.969266 * a + 1.8760108 * b + 0.041556 * c,
      0.0556434 * a - 0.2040259 * b + 1.0572252 * c,
    ] as const;
  }
  if (colorSpace === "acescg") {
    const x = 0.6624541811 * a + 0.1340042065 * b + 0.156187687 * c;
    const y = 0.2722287168 * a + 0.6740817658 * b + 0.0536895174 * c;
    const z = -0.0055746495 * a + 0.0040607335 * b + 1.0103391003 * c;
    return [
      3.2404542 * x - 1.5371385 * y - 0.4985314 * z,
      -0.969266 * x + 1.8760108 * y + 0.041556 * z,
      0.0556434 * x - 0.2040259 * y + 1.0572252 * z,
    ] as const;
  }
  return [a, b, c] as const;
}

function normalizeRank(shape: number[]) {
  if (shape.length === 2) {
    return [shape[0], shape[1], 1] as [number, number, number];
  }
  if (shape.length !== 3) {
    throw new Error(
      `当前三维查看器接受 rank 2 或 rank 3 film；文件 rank 为 ${shape.length}。`,
    );
  }
  return [shape[0], shape[1], shape[2]] as [number, number, number];
}

export function parseRayFilm(buffer: ArrayBuffer, name = "film.bin"): RayFilm {
  const reader = new FilmReader(buffer);

  const magic = reader.bytes(FILM_MAGIC.length, "Film magic");
  if (!FILM_MAGIC.every((value, index) => magic[index] === value)) {
    throw new Error("不支持的 Film 文件：magic 无效；旧版无头格式不再兼容。");
  }
  const version = reader.uint32("Film 版本");
  if (version !== FILM_VERSION) {
    throw new Error(`不支持的 Film 版本 ${version}；当前版本为 ${FILM_VERSION}。`);
  }

  const samples = reader.int64("采样数");
  if (samples < BigInt(0)) throw new Error("采样数不能为负数。");

  const rank = reader.uint32("张量阶数");
  if (rank < 1 || rank > 16) {
    throw new Error(`无效的张量阶数 ${rank}；这可能不是 Ray film 文件。`);
  }

  const sourceShape = Array.from({ length: rank }, (_, index) => {
    const extent = reader.uint64(`shape[${index}]`);
    if (extent > BigInt(Number.MAX_SAFE_INTEGER)) {
      throw new Error(`shape[${index}] 超出 JavaScript 安全整数范围。`);
    }
    return Number(extent);
  });
  const voxelCount = checkedProduct(sourceShape);
  const shape = normalizeRank(sourceShape);

  const colorSpaceLength = reader.uint32("颜色空间长度");
  if (colorSpaceLength < 1 || colorSpaceLength > 64) {
    throw new Error("颜色空间字段长度无效。");
  }
  const parsedColorSpace = UTF8.decode(reader.bytes(colorSpaceLength, "颜色空间"));
  if (parsedColorSpace !== "xyz" && parsedColorSpace !== "acescg" && parsedColorSpace !== "linear_srgb") {
    throw new Error(`不支持的 Film 颜色空间 ${parsedColorSpace}。`);
  }
  const colorSpace: FilmColorSpace = parsedColorSpace;

  const spectralBins = reader.uint32("光谱 bin 数");
  if (spectralBins > 4096) throw new Error("光谱诊断元数据无效。");
  const minNM = reader.float64("光谱下界");
  const maxNM = reader.float64("光谱上界");
  let spectralRange: [number, number] | null = null;
  if (spectralBins > 0) {
    if (!(minNM > 0) || !(maxNM > minNM)) throw new Error("光谱诊断元数据无效。");
    spectralRange = [minNM, maxNM];
  }

  const requiredPayloadBytes = (3 + spectralBins) * voxelCount * 8;
  if (reader.remaining !== requiredPayloadBytes) {
    throw new Error(`Film payload 长度为 ${reader.remaining}，预期 ${requiredPayloadBytes}。`);
  }

  const source = [
    new Float64Array(voxelCount),
    new Float64Array(voxelCount),
    new Float64Array(voxelCount),
  ] as const;
  for (let channel = 0; channel < 3; channel += 1) {
    for (let index = 0; index < voxelCount; index += 1) {
      source[channel][index] = reader.float64(`颜色通道 ${channel}`);
    }
  }
  if (spectralBins > 0) {
    reader.bytes(spectralBins * voxelCount * 8, "光谱诊断数据");
  }

  const channels = [
    new Float32Array(voxelCount),
    new Float32Array(voxelCount),
    new Float32Array(voxelCount),
  ] as [Float32Array, Float32Array, Float32Array];

  let min = Number.POSITIVE_INFINITY;
  let max = 0;
  let finiteValues = 0;
  let nonFiniteValues = 0;
  let occupiedVoxels = 0;
  for (let index = 0; index < voxelCount; index += 1) {
    const rgb = toLinearSRGB(
      source[0][index],
      source[1][index],
      source[2][index],
      colorSpace,
    );
    let occupied = false;
    for (let channel = 0; channel < 3; channel += 1) {
      const value = rgb[channel];
      if (!Number.isFinite(value)) {
        channels[channel][index] = 0;
        nonFiniteValues += 1;
        continue;
      }
      channels[channel][index] = value;
      min = Math.min(min, value);
      max = Math.max(max, value);
      finiteValues += 1;
      occupied ||= value > 0;
    }
    if (occupied) occupiedVoxels += 1;
  }
  if (!Number.isFinite(min)) min = 0;
  const textureScale = max > 0 ? max : 1;
  const textureData = new Uint8Array(voxelCount * 4);
  for (let index = 0; index < voxelCount; index += 1) {
    let density = 0;
    for (let channel = 0; channel < 3; channel += 1) {
      const normalized = Math.max(0, Math.min(1, channels[channel][index] / textureScale));
      textureData[index * 4 + channel] = Math.round(normalized * 255);
      density = Math.max(density, normalized);
    }
    textureData[index * 4 + 3] = Math.round(density * 255);
  }

  return {
    name,
    samples,
    shape,
    sourceRank: rank,
    sourceShape,
    voxelCount,
    colorSpace,
    channels,
    textureData,
    textureScale,
    stats: { min, max, finiteValues, nonFiniteValues, occupiedVoxels },
    spectralBins,
    spectralRange,
  };
}

export async function loadRayFilm(url: string, name?: string) {
  const response = await fetch(url);
  if (!response.ok) throw new Error(`无法加载示例 film（HTTP ${response.status}）。`);
  return parseRayFilm(await response.arrayBuffer(), name ?? url.split("/").pop());
}

export type ToneMap = "linear" | "reinhard" | "aces";

export function displayChannel(
  value: number,
  exposure: number,
  toneMap: ToneMap,
  gamma: number,
) {
  let output = Math.max(0, value * exposure);
  if (toneMap === "reinhard") output /= 1 + output;
  if (toneMap === "aces") {
    output =
      (output * (2.51 * output + 0.03)) /
      (output * (2.43 * output + 0.59) + 0.14);
  }
  output = Math.max(0, Math.min(1, output));
  return Math.pow(output, 1 / Math.max(0.1, gamma));
}
