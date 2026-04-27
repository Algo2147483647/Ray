import { useState } from "react";
import { shapeLabels } from "../data/shapeSchemas";
import { getPreviewSupport } from "../lib/scene-utils";
import type { ShapeType, SceneDocument } from "../types/scene";

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
          <button className="collapse-toggle text" type="button" onClick={onToggleCollapsed}>
            Open
          </button>
          <span className="rail-label">Outline</span>
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
        <button className="collapse-toggle text" type="button" onClick={onToggleCollapsed}>
          Collapse
        </button>
      </div>

      <section className="outline-section">
        <div className="section-title-row">
          <h3>Objects</h3>
          <div className="action-cluster add-shape-cluster compact">
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
              className="button ghost compact-button"
              type="button"
              onClick={() => onAddObject(nextShape)}
              disabled={disableMutations}
            >
              Add
            </button>
          </div>
        </div>

        <div className="list-stack">
          {scene.objects.map((object, index) => (
            <div
              key={object.id}
              className={`outline-item ${selectedObjectId === object.id ? "selected" : ""}`}
            >
              <div className="outline-item-body">
                <button
                  className="outline-main"
                  type="button"
                  onClick={() => onSelectObject(object.id)}
                >
                  <span className="outline-copy">
                    <span className="outline-title-row">
                      <strong>{object.id}</strong>
                      <span className={`support-badge ${getPreviewSupport(object.shape)}`}>
                        {getPreviewSupport(object.shape)}
                      </span>
                    </span>
                    <span className="outline-meta-row">
                      <span className="shape-tag">{shapeLabels[object.shape]}</span>
                      <span className="material-link">{object.material_id}</span>
                    </span>
                  </span>
                </button>

                <div className="outline-inline-actions">
                  <button
                    className="action-chip"
                    type="button"
                    onClick={() => onMoveObject(object.id, -1)}
                    disabled={disableMutations || index === 0}
                    aria-label={`Move ${object.id} up`}
                  >
                    Up
                  </button>

                  <button
                    className="action-chip"
                    type="button"
                    onClick={() => onMoveObject(object.id, 1)}
                    disabled={disableMutations || index === scene.objects.length - 1}
                    aria-label={`Move ${object.id} down`}
                  >
                    Dn
                  </button>

                  <button
                    className="action-chip danger"
                    type="button"
                    onClick={() => onRemoveObject(object.id)}
                    disabled={disableMutations}
                    aria-label={`Remove ${object.id}`}
                  >
                    Del
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="outline-section">
        <div className="section-title-row">
          <h3>Materials</h3>
          <button
            className="button ghost compact-button"
            type="button"
            onClick={onAddMaterial}
            disabled={disableMutations}
          >
            Add
          </button>
        </div>

        <div className="list-stack">
          {scene.materials.map((material) => {
            const materialRole =
              (material.radiate ?? 0) > 0 || material.radiation
                ? "Light source"
                : "Surface";

            return (
              <div
                key={material.id}
                className={`outline-item ${selectedMaterialId === material.id ? "selected" : ""}`}
              >
                <div className="outline-item-body materials">
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
                      <span className="outline-title-row">
                        <strong>{material.id}</strong>
                      </span>
                      <span className="outline-meta-row">
                        <span className="material-role">{materialRole}</span>
                      </span>
                    </span>
                  </button>

                  <button
                    className="action-chip danger"
                    type="button"
                    onClick={() => onRemoveMaterial(material.id)}
                    disabled={disableMutations || scene.materials.length <= 1}
                    aria-label={`Remove ${material.id}`}
                  >
                    Remove
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      </section>
    </div>
  );
}
