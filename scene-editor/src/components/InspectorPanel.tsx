import { shapeSchemas, shapeLabels } from "../data/shapeSchemas";
import type { SceneCamera, SceneDocument, SceneMaterial, SceneObject, ShapeType } from "../types/scene";
import {
  formatNumberList,
  parseNumberList,
  getPreviewSupport,
  toNumber,
  toVec3
} from "../lib/scene-utils";
import type { InspectorTab } from "../lib/scene-editor-state";

interface InspectorPanelProps {
  scene: SceneDocument;
  selectedTab: InspectorTab;
  selectedObject: SceneObject | null;
  selectedMaterial: SceneMaterial | null;
  jsonDraft: string;
  jsonDirty: boolean;
  jsonError: string | null;
  collapsed: boolean;
  disableMutations: boolean;
  onSelectTab: (tab: InspectorTab) => void;
  onUpdateObject: (objectId: string, patch: Partial<SceneObject>) => void;
  onChangeShape: (objectId: string, shape: ShapeType) => void;
  onUpdateMaterial: (materialId: string, patch: Partial<SceneMaterial>) => void;
  onUpdateCamera: (patch: Partial<SceneCamera>) => void;
  onJsonDraftChange: (value: string) => void;
  onApplyJson: () => void;
  onDiscardJson: () => void;
  onToggleCollapsed: () => void;
}

export function InspectorPanel({
  scene,
  selectedTab,
  selectedObject,
  selectedMaterial,
  jsonDraft,
  jsonDirty,
  jsonError,
  collapsed,
  disableMutations,
  onSelectTab,
  onUpdateObject,
  onChangeShape,
  onUpdateMaterial,
  onUpdateCamera,
  onJsonDraftChange,
  onApplyJson,
  onDiscardJson,
  onToggleCollapsed
}: InspectorPanelProps) {
  const camera = scene.cameras[0];

  if (collapsed) {
    return (
      <div className="panel-card inspector-card collapsed-card">
        <div className="collapsed-rail right">
          <button className="collapse-toggle" type="button" onClick={onToggleCollapsed}>
            «
          </button>
          <span className="rail-label">Inspector</span>
        </div>
      </div>
    );
  }

  return (
    <div className="panel-card inspector-card">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">Inspector</p>
        </div>
        <button className="collapse-toggle" type="button" onClick={onToggleCollapsed}>
          »
        </button>
      </div>

      <div className="tab-strip">
        <button
          type="button"
          className={selectedTab === "object" ? "active" : ""}
          onClick={() => onSelectTab("object")}
        >
          Object
        </button>
        <button
          type="button"
          className={selectedTab === "material" ? "active" : ""}
          onClick={() => onSelectTab("material")}
        >
          Material
        </button>
        <button
          type="button"
          className={selectedTab === "camera" ? "active" : ""}
          onClick={() => onSelectTab("camera")}
        >
          Camera
        </button>
        <button
          type="button"
          className={selectedTab === "json" ? "active" : ""}
          onClick={() => onSelectTab("json")}
        >
          JSON
        </button>
      </div>

      <div className="inspector-body">
        {selectedTab === "object" && (
          <ObjectTab
            scene={scene}
            selectedObject={selectedObject}
            disableMutations={disableMutations}
            onUpdateObject={onUpdateObject}
            onChangeShape={onChangeShape}
          />
        )}
        {selectedTab === "material" && (
          <MaterialTab
            selectedMaterial={selectedMaterial}
            disableMutations={disableMutations}
            onUpdateMaterial={onUpdateMaterial}
          />
        )}
        {selectedTab === "camera" && (
          <CameraTab
            camera={camera}
            disableMutations={disableMutations}
            onUpdateCamera={onUpdateCamera}
          />
        )}
        {selectedTab === "json" && (
          <JsonTab
            jsonDraft={jsonDraft}
            jsonDirty={jsonDirty}
            jsonError={jsonError}
            onJsonDraftChange={onJsonDraftChange}
            onApplyJson={onApplyJson}
            onDiscardJson={onDiscardJson}
          />
        )}
      </div>
    </div>
  );
}

function ObjectTab({
  scene,
  selectedObject,
  disableMutations,
  onUpdateObject,
  onChangeShape
}: {
  scene: SceneDocument;
  selectedObject: SceneObject | null;
  disableMutations: boolean;
  onUpdateObject: (objectId: string, patch: Partial<SceneObject>) => void;
  onChangeShape: (objectId: string, shape: ShapeType) => void;
}) {
  if (!selectedObject) {
    return <p className="empty-state">Select an object from the scene graph.</p>;
  }

  const fields = shapeSchemas[selectedObject.shape];

  return (
    <div className="detail-stack">
      <div className="preview-note">
        <strong>{shapeLabels[selectedObject.shape]}</strong>
        <span className={`support-badge ${getPreviewSupport(selectedObject.shape)}`}>
          {getPreviewSupport(selectedObject.shape)} preview
        </span>
      </div>

      <label className="field">
        <span>ID</span>
        <input
          type="text"
          value={selectedObject.id}
          disabled={disableMutations}
          onChange={(event) =>
            onUpdateObject(selectedObject.id, { id: event.target.value })
          }
        />
      </label>

      <label className="field">
        <span>Shape</span>
        <select
          value={selectedObject.shape}
          disabled={disableMutations}
          onChange={(event) =>
            onChangeShape(selectedObject.id, event.target.value as ShapeType)
          }
        >
          {Object.entries(shapeLabels).map(([shape, label]) => (
            <option key={shape} value={shape}>
              {label}
            </option>
          ))}
        </select>
      </label>

      <label className="field">
        <span>Material</span>
        <select
          value={selectedObject.material_id}
          disabled={disableMutations}
          onChange={(event) =>
            onUpdateObject(selectedObject.id, { material_id: event.target.value })
          }
        >
          {scene.materials.map((material) => (
            <option key={material.id} value={material.id}>
              {material.id}
            </option>
          ))}
        </select>
      </label>

      {fields.map((field) => {
        if (field.kind === "number") {
          const rawValue = selectedObject[field.key];
          const value = Array.isArray(rawValue) ? toNumber(rawValue[0]) : toNumber(rawValue);
          return (
            <label key={String(field.key)} className="field">
              <span>{field.label}</span>
              <small>{field.hint}</small>
              <input
                type="number"
                step="0.1"
                value={value}
                disabled={disableMutations}
                onChange={(event) =>
                  onUpdateObject(selectedObject.id, {
                    [field.key]: Number(event.target.value)
                  })
                }
              />
            </label>
          );
        }

        if (field.kind === "vector3") {
          const [x, y, z] = toVec3(selectedObject[field.key]);
          return (
            <div key={String(field.key)} className="field">
              <span>{field.label}</span>
              <small>{field.hint}</small>
              <div className="vector-grid">
                {[x, y, z].map((value, index) => (
                  <input
                    key={index}
                    type="number"
                    step="0.1"
                    value={value}
                    disabled={disableMutations}
                    onChange={(event) => {
                      const nextValue = Number(event.target.value);
                      const nextVector = [x, y, z];
                      nextVector[index] = nextValue;
                      onUpdateObject(selectedObject.id, {
                        [field.key]: nextVector
                      });
                    }}
                  />
                ))}
              </div>
            </div>
          );
        }

        return (
          <label key={String(field.key)} className="field">
            <span>{field.label}</span>
            <small>{field.hint}</small>
            <textarea
              key={`${selectedObject.id}-${String(field.key)}`}
              defaultValue={formatNumberList(selectedObject[field.key])}
              rows={4}
              disabled={disableMutations}
              onBlur={(event) => {
                const parsed = parseNumberList(event.target.value);
                if (parsed === null) {
                  window.alert(`${field.label} expects a comma or space separated number list.`);
                  event.target.value = formatNumberList(selectedObject[field.key]);
                  return;
                }

                onUpdateObject(selectedObject.id, {
                  [field.key]: parsed
                });
              }}
            />
          </label>
        );
      })}
    </div>
  );
}

function MaterialTab({
  selectedMaterial,
  disableMutations,
  onUpdateMaterial
}: {
  selectedMaterial: SceneMaterial | null;
  disableMutations: boolean;
  onUpdateMaterial: (materialId: string, patch: Partial<SceneMaterial>) => void;
}) {
  if (!selectedMaterial) {
    return <p className="empty-state">Select a material from the scene graph.</p>;
  }

  const color = selectedMaterial.color.slice(0, 3);
  while (color.length < 3) {
    color.push(1);
  }

  return (
    <div className="detail-stack">
      <label className="field">
        <span>ID</span>
        <input
          type="text"
          value={selectedMaterial.id}
          disabled={disableMutations}
          onChange={(event) =>
            onUpdateMaterial(selectedMaterial.id, { id: event.target.value })
          }
        />
      </label>

      <div className="field">
        <span>Color</span>
        <small>Use scene-linear RGB values. Lights can exceed 1.</small>
        <div className="vector-grid">
          {color.map((value, index) => (
            <input
              key={index}
              type="number"
              step="0.1"
              value={value}
              disabled={disableMutations}
              onChange={(event) => {
                const nextColor = [...color];
                nextColor[index] = Number(event.target.value);
                onUpdateMaterial(selectedMaterial.id, { color: nextColor });
              }}
            />
          ))}
        </div>
      </div>

      {(
        [
          ["diffuse_loss", "Diffuse loss"],
          ["reflect_loss", "Reflect loss"],
          ["refract_loss", "Refract loss"],
          ["reflectivity", "Reflectivity"],
          ["refractivity", "Refractivity"]
        ] as const
      ).map(([key, label]) => (
        <label key={key} className="field">
          <span>{label}</span>
          <input
            type="number"
            step="0.05"
            value={toNumber(selectedMaterial[key])}
            disabled={disableMutations}
            onChange={(event) =>
              onUpdateMaterial(selectedMaterial.id, {
                [key]: Number(event.target.value)
              })
            }
          />
        </label>
      ))}

      <label className="field">
        <span>Radiance flag</span>
        <input
          type="number"
          step="1"
          value={toNumber(selectedMaterial.radiate)}
          disabled={disableMutations}
          onChange={(event) =>
            onUpdateMaterial(selectedMaterial.id, {
              radiate: Number(event.target.value)
            })
          }
        />
      </label>
    </div>
  );
}

function CameraTab({
  camera,
  disableMutations,
  onUpdateCamera
}: {
  camera: SceneCamera;
  disableMutations: boolean;
  onUpdateCamera: (patch: Partial<SceneCamera>) => void;
}) {
  return (
    <div className="detail-stack">
      <VectorField
        label="Position"
        hint="Viewport camera origin"
        value={toVec3(camera.position)}
        disabled={disableMutations}
        onChange={(nextValue) => onUpdateCamera({ position: nextValue })}
      />
      <VectorField
        label="Direction"
        hint="Look direction"
        value={toVec3(camera.direction)}
        disabled={disableMutations}
        onChange={(nextValue) => onUpdateCamera({ direction: nextValue })}
      />
      <VectorField
        label="Up"
        hint="Camera up axis"
        value={toVec3(camera.up)}
        disabled={disableMutations}
        onChange={(nextValue) => onUpdateCamera({ up: nextValue })}
      />
      <label className="field">
        <span>Field of view</span>
        <input
          type="number"
          step="1"
          value={camera.field_of_view}
          disabled={disableMutations}
          onChange={(event) =>
            onUpdateCamera({ field_of_view: Number(event.target.value) })
          }
        />
      </label>
      <div className="vector-grid dual">
        <label className="field compact">
          <span>Width</span>
          <input
            type="number"
            step="1"
            value={camera.width}
            disabled={disableMutations}
            onChange={(event) =>
              onUpdateCamera({ width: Number(event.target.value) })
            }
          />
        </label>
        <label className="field compact">
          <span>Height</span>
          <input
            type="number"
            step="1"
            value={camera.height}
            disabled={disableMutations}
            onChange={(event) =>
              onUpdateCamera({ height: Number(event.target.value) })
            }
          />
        </label>
      </div>
    </div>
  );
}

function JsonTab({
  jsonDraft,
  jsonDirty,
  jsonError,
  onJsonDraftChange,
  onApplyJson,
  onDiscardJson
}: {
  jsonDraft: string;
  jsonDirty: boolean;
  jsonError: string | null;
  onJsonDraftChange: (value: string) => void;
  onApplyJson: () => void;
  onDiscardJson: () => void;
}) {
  return (
    <div className="detail-stack json-stack">
      <div className="preview-note">
        <strong>Raw scene JSON</strong>
        <span>{jsonDirty ? "Unapplied changes" : "In sync with inspector"}</span>
      </div>
      <textarea
        className="json-editor"
        value={jsonDraft}
        onChange={(event) => onJsonDraftChange(event.target.value)}
        spellCheck={false}
      />
      {jsonError && <p className="error-text">{jsonError}</p>}
      <div className="action-row">
        <button className="button primary" type="button" onClick={onApplyJson}>
          Apply JSON
        </button>
        <button
          className="button ghost"
          type="button"
          onClick={onDiscardJson}
          disabled={!jsonDirty}
        >
          Discard edits
        </button>
      </div>
    </div>
  );
}

function VectorField({
  label,
  hint,
  value,
  disabled,
  onChange
}: {
  label: string;
  hint: string;
  value: [number, number, number];
  disabled: boolean;
  onChange: (value: [number, number, number]) => void;
}) {
  return (
    <div className="field">
      <span>{label}</span>
      <small>{hint}</small>
      <div className="vector-grid">
        {value.map((entry, index) => (
          <input
            key={index}
            type="number"
            step="0.1"
            value={entry}
            disabled={disabled}
            onChange={(event) => {
              const next = [...value] as [number, number, number];
              next[index] = Number(event.target.value);
              onChange(next);
            }}
          />
        ))}
      </div>
    </div>
  );
}
