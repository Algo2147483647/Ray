"use client";

import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type DragEvent,
} from "react";
import { loadRayFilm, parseRayFilm, type RayFilm, type ToneMap } from "./film";
import { SliceCanvas } from "./SliceCanvas";
import { VolumeViewport, type VolumeMode } from "./VolumeViewport";

const DEFAULT_CONTROLS = {
  mode: "dvr" as VolumeMode,
  threshold: 0.018,
  softness: 0.04,
  opacity: 5.5,
  steps: 256,
  exposure: 1,
  toneMap: "linear" as ToneMap,
  gamma: 1,
  linearFilter: true,
};

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

function formatNumber(value: number) {
  if (!Number.isFinite(value)) return "—";
  if (Math.abs(value) >= 1000 || (Math.abs(value) > 0 && Math.abs(value) < 0.001)) {
    return value.toExponential(3);
  }
  return value.toFixed(4);
}

function RangeControl({
  label,
  value,
  min,
  max,
  step,
  display,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  step: number;
  display?: string;
  onChange: (value: number) => void;
}) {
  const controlId = useId();
  return (
    <div className="range-control">
      <span>
        <label htmlFor={controlId}>{label}</label>
        <output>{display ?? value.toFixed(2)}</output>
      </span>
      <input
        id={controlId}
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
      />
    </div>
  );
}

export default function Home() {
  const [film, setFilm] = useState<RayFilm | null>(null);
  const [fileSize, setFileSize] = useState(0);
  const [loading, setLoading] = useState(true);
  const [dragging, setDragging] = useState(false);
  const [error, setError] = useState("");
  const [controls, setControls] = useState(DEFAULT_CONTROLS);
  const [slices, setSlices] = useState<[number, number, number]>([0, 0, 0]);
  const inputRef = useRef<HTMLInputElement>(null);

  const reportError = useCallback((message: string) => setError(message), []);

  const installFilm = useCallback((next: RayFilm, bytes: number) => {
    setFilm(next);
    setFileSize(bytes);
    setSlices(next.shape.map((extent) => Math.floor((extent - 1) / 2)) as [number, number, number]);
    setError("");
  }, []);

  const loadFile = useCallback(async (file: File) => {
    setLoading(true);
    try {
      if (file.size > 1024 * 1024 * 1024) {
        throw new Error("浏览器查看器拒绝超过 1 GB 的文件。");
      }
      installFilm(parseRayFilm(await file.arrayBuffer(), file.name), file.size);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "无法解析该 film 文件。");
    } finally {
      setLoading(false);
    }
  }, [installFilm]);

  useEffect(() => {
    let active = true;
    loadRayFilm("/sample-klein.bin", "klein-bottle-4d-showcase-96x36x36-128spp.bin")
      .then((sample) => {
        if (active) installFilm(sample, 2986023);
      })
      .catch((cause) => {
        if (active) setError(cause instanceof Error ? cause.message : "示例加载失败。");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [installFilm]);

  const dropFile = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setDragging(false);
    const file = event.dataTransfer.files.item(0);
    if (file) void loadFile(file);
  };

  const updateControl = <Key extends keyof typeof controls>(
    key: Key,
    value: (typeof controls)[Key],
  ) => setControls((current) => ({ ...current, [key]: value }));

  const occupancy = film
    ? (film.stats.occupiedVoxels / Math.max(1, film.voxelCount)) * 100
    : 0;

  return (
    <main className="app-shell">
      <header className="topbar">
        <div className="brand-lockup">
          <span className="brand-mark"><i />RAY</span>
          <span className="brand-divider" />
          <div>
            <strong>FILM LAB</strong>
            <span>LOCAL TENSOR INSPECTION</span>
          </div>
        </div>
        <div className="topbar-actions">
          <span className="privacy-pill"><i />仅在本机解析</span>
          <input
            ref={inputRef}
            className="visually-hidden"
            type="file"
            accept=".bin,application/octet-stream"
            onChange={(event) => {
              const file = event.target.files?.item(0);
              if (file) void loadFile(file);
              event.target.value = "";
            }}
          />
          <button className="load-button" type="button" onClick={() => inputRef.current?.click()}>
            <span>＋</span> 选择 .bin
          </button>
        </div>
      </header>

      <section className="data-ribbon" aria-label="film 文件摘要">
        <div className="file-identity">
          <span className="eyebrow">ACTIVE FILM</span>
          <strong>{loading ? "正在读取…" : film?.name ?? "未加载"}</strong>
        </div>
        <div className="ribbon-stat"><span>SHAPE</span><strong>{film?.sourceShape.join(" × ") ?? "—"}</strong></div>
        <div className="ribbon-stat"><span>VOXELS</span><strong>{film?.voxelCount.toLocaleString() ?? "—"}</strong></div>
        <div className="ribbon-stat"><span>SAMPLES</span><strong>{film?.samples.toLocaleString() ?? "—"}</strong></div>
        <div className="ribbon-stat"><span>SPACE</span><strong>{film?.colorSpace ?? "—"}</strong></div>
        <div className="ribbon-stat"><span>FILE</span><strong>{fileSize ? formatBytes(fileSize) : "—"}</strong></div>
        <div className="status-indicator"><i />{film ? "READY" : "IDLE"}</div>
      </section>

      {error && (
        <div className="error-banner" role="alert">
          <strong>检查数据</strong><span>{error}</span>
          <button type="button" onClick={() => setError("")} aria-label="关闭错误提示">×</button>
        </div>
      )}

      <div
        className={`workspace ${dragging ? "is-dragging" : ""}`}
        onDragEnter={(event) => { event.preventDefault(); setDragging(true); }}
        onDragOver={(event) => event.preventDefault()}
        onDragLeave={(event) => {
          if (event.currentTarget === event.target) setDragging(false);
        }}
        onDrop={dropFile}
      >
        <section className="visual-column">
          <div className="panel volume-panel">
            <div className="panel-heading">
              <div>
                <span className="eyebrow">VOLUME / INDEX SPACE</span>
                <h1>三维张量场</h1>
              </div>
              <div className="mode-switcher" aria-label="体绘制模式">
                {(["dvr", "mip", "iso"] as VolumeMode[]).map((mode) => (
                  <button
                    key={mode}
                    type="button"
                    className={controls.mode === mode ? "active" : ""}
                    onClick={() => updateControl("mode", mode)}
                  >
                    {mode.toUpperCase()}
                  </button>
                ))}
              </div>
            </div>
            <VolumeViewport film={film} controls={controls} onError={reportError} />
            <div className="viewport-legend">
              <span><i className="legend-cyan" />线性 RGB</span>
              <span><i className="legend-purple" />体素透明度</span>
              <span>盒体比例按张量尺寸归一化</span>
            </div>
          </div>

          {film && (
            <div className="slice-grid">
              <SliceCanvas film={film} plane="xy" index={slices[2]} exposure={controls.exposure} toneMap={controls.toneMap} gamma={controls.gamma} />
              <SliceCanvas film={film} plane="xz" index={slices[1]} exposure={controls.exposure} toneMap={controls.toneMap} gamma={controls.gamma} />
              <SliceCanvas film={film} plane="yz" index={slices[0]} exposure={controls.exposure} toneMap={controls.toneMap} gamma={controls.gamma} />
            </div>
          )}
        </section>

        <aside className="control-column">
          <section className="panel control-panel">
            <div className="panel-heading compact">
              <div><span className="eyebrow">TRANSFER FUNCTION</span><h2>体绘制</h2></div>
              <button type="button" className="text-button" onClick={() => setControls(DEFAULT_CONTROLS)}>重置</button>
            </div>
            <RangeControl label="背景阈值" value={controls.threshold} min={0} max={0.4} step={0.002} display={controls.threshold.toFixed(3)} onChange={(value) => updateControl("threshold", value)} />
            <RangeControl label="阈值柔度" value={controls.softness} min={0.002} max={0.3} step={0.002} display={controls.softness.toFixed(3)} onChange={(value) => updateControl("softness", value)} />
            <RangeControl label="不透明度" value={controls.opacity} min={0.25} max={14} step={0.25} onChange={(value) => updateControl("opacity", value)} />
            <RangeControl label="采样步数" value={controls.steps} min={64} max={512} step={16} display={String(controls.steps)} onChange={(value) => updateControl("steps", value)} />
            <div className="toggle-row">
              <span><strong>三线性过滤</strong><small>关闭可检查原始体素边界</small></span>
              <button
                className="toggle-switch"
                type="button"
                role="switch"
                aria-checked={controls.linearFilter}
                aria-label="三线性过滤"
                onClick={() => updateControl("linearFilter", !controls.linearFilter)}
              ><i /></button>
            </div>
          </section>

          <section className="panel control-panel">
            <div className="panel-heading compact"><div><span className="eyebrow">DISPLAY PIPELINE</span><h2>颜色</h2></div></div>
            <RangeControl label="曝光" value={controls.exposure} min={0.05} max={4} step={0.05} display={`${controls.exposure.toFixed(2)}×`} onChange={(value) => updateControl("exposure", value)} />
            <RangeControl label="Gamma" value={controls.gamma} min={0.5} max={3} step={0.05} display={controls.gamma.toFixed(2)} onChange={(value) => updateControl("gamma", value)} />
            <label className="select-control" htmlFor="tone-map"><span>色调映射</span>
              <select id="tone-map" value={controls.toneMap} onChange={(event) => updateControl("toneMap", event.target.value as ToneMap)}>
                <option value="linear">Linear</option><option value="reinhard">Reinhard</option><option value="aces">ACES</option>
              </select>
            </label>
          </section>

          {film && (
            <section className="panel control-panel">
              <div className="panel-heading compact"><div><span className="eyebrow">ORTHOGONAL PLANES</span><h2>切片位置</h2></div></div>
              <RangeControl label="X / YZ" value={slices[0]} min={0} max={film.shape[0] - 1} step={1} display={`${slices[0]} / ${film.shape[0] - 1}`} onChange={(value) => setSlices((current) => [value, current[1], current[2]])} />
              <RangeControl label="Y / XZ" value={slices[1]} min={0} max={film.shape[1] - 1} step={1} display={`${slices[1]} / ${film.shape[1] - 1}`} onChange={(value) => setSlices((current) => [current[0], value, current[2]])} />
              <RangeControl label="Z / XY" value={slices[2]} min={0} max={film.shape[2] - 1} step={1} display={`${slices[2]} / ${film.shape[2] - 1}`} onChange={(value) => setSlices((current) => [current[0], current[1], value])} />
            </section>
          )}

          <section className="panel diagnostics-panel">
            <div className="panel-heading compact"><div><span className="eyebrow">DIAGNOSTICS</span><h2>数据质量</h2></div></div>
            <dl>
              <div><dt>有效范围</dt><dd>{film ? `${formatNumber(film.stats.min)} — ${formatNumber(film.stats.max)}` : "—"}</dd></div>
              <div><dt>非有限值</dt><dd className={film?.stats.nonFiniteValues ? "warn" : ""}>{film?.stats.nonFiniteValues.toLocaleString() ?? "—"}</dd></div>
              <div><dt>非黑体素</dt><dd>{film ? `${occupancy.toFixed(2)}%` : "—"}</dd></div>
              <div><dt>光谱 bins</dt><dd>{film?.spectralBins || "—"}</dd></div>
              <div><dt>显示语义</dt><dd>film index space</dd></div>
            </dl>
            <p>RGB film 是投影后的三维辐射张量，不等同于完整四维几何。精确表面重建仍需要 depth / hit-position AOV。</p>
          </section>
        </aside>

        {dragging && (
          <div className="drop-overlay">
            <div><span>↓</span><strong>释放以检查 film</strong><small>文件始终留在此设备</small></div>
          </div>
        )}
      </div>

      <footer>
        <span>Ray Film Lab · WebGL2 volume inspection</span>
        <span>Little-endian float64 · RGB planar channels</span>
      </footer>
    </main>
  );
}
