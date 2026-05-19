interface StatsPanelProps {
  totalObjects: number;
  totalMaterials: number;
  lightSources: number;
  exactPreviewable: number;
  proxyPreviewable: number;
  shapeCounts: Record<string, number>;
}

export function StatsPanel({
  totalObjects,
  totalMaterials,
  lightSources,
  exactPreviewable,
  proxyPreviewable,
  shapeCounts
}: StatsPanelProps) {
  return (
    <div className="panel-card stats-card">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">Summary</p>
          <h2>Reliable scene stats</h2>
        </div>
      </div>

      <div className="stats-grid">
        <article>
          <span>Total objects</span>
          <strong>{totalObjects}</strong>
        </article>
        <article>
          <span>Materials</span>
          <strong>{totalMaterials}</strong>
        </article>
        <article>
          <span>Light sources</span>
          <strong>{lightSources}</strong>
        </article>
        <article>
          <span>Exact preview</span>
          <strong>{exactPreviewable}</strong>
        </article>
        <article>
          <span>Proxy preview</span>
          <strong>{proxyPreviewable}</strong>
        </article>
      </div>

      <div className="shape-breakdown">
        {Object.entries(shapeCounts).map(([shape, count]) => (
          <span key={shape} className="shape-chip">
            {shape}: {count}
          </span>
        ))}
        {Object.keys(shapeCounts).length === 0 && (
          <span className="empty-state">No objects in the current scene.</span>
        )}
      </div>
    </div>
  );
}
