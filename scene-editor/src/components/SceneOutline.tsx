import { useState } from "react";
import { shapeLabels } from "../data/shapeSchemas";
import type { SceneDocument, ShapeType } from "../types/scene";
import { getPreviewSupport } from "../lib/scene-utils";

interface SceneOutlineProps {
  scene: SceneDocument;
  selectedObjectId: string | null;
  selectedMaterialId: string | null;
  collapsed: boolean;
  disableMutations: boolean;
  onSelectObject: (objectId: string) => void;
  onSelectMaterial: (materialId: string) => void;
  onAddObject: (shape: ShapeType) => void;
  onAddMaterial: () => void;
  onMoveObject: (objectId: string, direction: -1 | 1) => void;
  onRemoveObject: (objectId: string) => void;
  onRemoveMaterial: (materialId: string) => void;
  onToggleCollapsed: () => void;
}

export function SceneOutline({
  scene,
  selectedObjectId,
  selectedMaterialId,
  collapsed,
  disableMutations,
  onSelectObject,
  onSelectMaterial,
  onAddObject,
  onAddMaterial,
  onMoveObject,
  onRemoveObject,
  onRemoveMaterial,
  onToggleCollapsed
}: SceneOutlineProps) {
  const [nextShape, setNextShape] = useState<ShapeType>("cuboid");

  if (collapsed) {
    return (
      <div className="panel-card navigator-card collapsed-card">
        <div className="collapsed-rail">
          <button className="collapse-toggle" type="button" onClick={onToggleCollapsed}>
            »
          </button>
          <span className="rail-label">Objects</span>
        </div>
      </div>
    );
  }

  return (
    <div className="panel-card navigator-card">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">Objects & Materials</p>
        </div>
        <button className="collapse-toggle" type="button" onClick={onToggleCollapsed}>
          «
        </button>
      </div>

      <section className="outline-section">
        <div className="section-title-row">
          <h3>Objects</h3>
          <div className="action-cluster add-shape-cluster">
            <select
              value={nextShape}
              onChange={(event) => setNextShape(event.target.value as ShapeType)}
              disabled={disableMutations}
            >
              {Object.entries(shapeLabels).map(([shape, label]) => (
                <option key={shape} value={shape}>
                  {label}
                </option>
              ))}
            </select>
            <button
              className="button ghost"
              type="button"
              onClick={() => onAddObject(nextShape)}
              disabled={disableMutations}
            >
              Add object
            </button>
          </div>
        </div>
        <div className="list-stack">
          {scene.objects.map((object, index) => (
            <div
              key={object.id}
              className={`outline-item ${selectedObjectId === object.id ? "selected" : ""}`}
            >
              <button
                className="outline-main"
                type="button"
                onClick={() => onSelectObject(object.id)}
              >
                <span className="outline-copy">
                  <strong>{object.id}</strong>
                  <span>{shapeLabels[object.shape]}</span>
                </span>
                <span
                  className={`support-badge ${getPreviewSupport(object.shape)}`}
                >
                  {getPreviewSupport(object.shape)}
                </span>
              </button>
              <div className="outline-actions">
                <button
                  className="icon-button"
                  type="button"
                  onClick={() => onMoveObject(object.id, -1)}
                  disabled={disableMutations || index === 0}
                  aria-label={`Move ${object.id} up`}
                >
                  ↑
                </button>
                <button
                  className="icon-button"
                  type="button"
                  onClick={() => onMoveObject(object.id, 1)}
                  disabled={disableMutations || index === scene.objects.length - 1}
                  aria-label={`Move ${object.id} down`}
                >
                  ↓
                </button>
                <button
                  className="icon-button danger"
                  type="button"
                  onClick={() => onRemoveObject(object.id)}
                  disabled={disableMutations}
                  aria-label={`Remove ${object.id}`}
                >
                  ×
                </button>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="outline-section">
        <div className="section-title-row">
          <h3>Materials</h3>
          <button
            className="button ghost"
            type="button"
            onClick={onAddMaterial}
            disabled={disableMutations}
          >
            Add material
          </button>
        </div>
        <div className="list-stack">
          {scene.materials.map((material) => (
            <div
              key={material.id}
              className={`outline-item ${selectedMaterialId === material.id ? "selected" : ""}`}
            >
              <button
                className="outline-main"
                type="button"
                onClick={() => onSelectMaterial(material.id)}
              >
                <span
                  className="material-dot"
                  style={{
                    background: `rgb(${material.color
                      .slice(0, 3)
                      .map((value) => Math.min(255, Math.max(0, value * 255)))
                      .join(",")})`
                  }}
                />
                <span className="outline-copy">
                  <strong>{material.id}</strong>
                  <span>
                    {(material.radiate ?? 0) > 0 || material.radiation ? "Light source" : "Surface"}
                  </span>
                </span>
              </button>
              <div className="outline-actions">
                <button
                  className="icon-button danger"
                  type="button"
                  onClick={() => onRemoveMaterial(material.id)}
                  disabled={disableMutations || scene.materials.length <= 1}
                  aria-label={`Remove ${material.id}`}
                >
                  ×
                </button>
              </div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
