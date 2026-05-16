import { startTransition, useMemo, useReducer, useState } from "react";
import { SceneOutline } from "./components/SceneOutline";
import { InspectorPanel } from "./components/InspectorPanel";
import { SceneViewport } from "./components/SceneViewport";
import { defaultScene } from "./data/defaultScene";
import { createInitialState, editorReducer } from "./lib/scene-editor-state";
import { parseSceneText } from "./lib/scene-utils";
import type { ShapeType } from "./types/scene";

export default function App() {
  const [state, dispatch] = useReducer(
    editorReducer,
    defaultScene,
    createInitialState
  );
  const [copyState, setCopyState] = useState<"idle" | "copied" | "failed">("idle");
  const [leftCollapsed, setLeftCollapsed] = useState(false);
  const [rightCollapsed, setRightCollapsed] = useState(false);

  const selectedObject = useMemo(
    () => state.scene.objects.find((object) => object.id === state.selectedObjectId) ?? null,
    [state.scene.objects, state.selectedObjectId]
  );
  const selectedMaterial = useMemo(
    () =>
      state.scene.materials.find((material) => material.id === state.selectedMaterialId) ?? null,
    [state.scene.materials, state.selectedMaterialId]
  );
  const handleApplyJson = () => {
    try {
      const nextScene = parseSceneText(state.jsonDraft);
      startTransition(() => {
        dispatch({ type: "apply-json-scene", scene: nextScene });
      });
    } catch (error) {
      dispatch({
        type: "set-json-error",
        error: error instanceof Error ? error.message : "Failed to parse JSON."
      });
    }
  };

  const handleCopyJson = async () => {
    try {
      await navigator.clipboard.writeText(state.committedJson);
      setCopyState("copied");
      window.setTimeout(() => setCopyState("idle"), 1800);
    } catch {
      setCopyState("failed");
    }
  };

  const handleAddObject = (shape: ShapeType) => {
    dispatch({ type: "add-object", shape });
  };

  return (
    <div className="app-shell">
      <header className="toolbar">
        <h1>Scene Editor</h1>
        <div className="hero-actions">
          <button
            className="button primary"
            type="button"
            onClick={() => dispatch({ type: "select-tab", tab: "json" })}
          >
            Open JSON
          </button>
          <button className="button ghost" type="button" onClick={handleCopyJson}>
            {copyState === "copied"
              ? "Copied"
              : copyState === "failed"
                ? "Copy failed"
                : "Copy JSON"}
          </button>
          <button
            className="button ghost"
            type="button"
            onClick={() => dispatch({ type: "reset-scene", scene: defaultScene })}
          >
            Reset sample
          </button>
        </div>
      </header>

      {state.jsonDirty && (
        <div className="status-banner">
          <strong>Raw JSON has unapplied changes.</strong>
          <span>
            Inspector mutations are temporarily locked so the scene never drifts
            out of sync. Apply or discard the JSON draft to continue.
          </span>
          <div className="action-row">
            <button className="button primary" type="button" onClick={handleApplyJson}>
              Apply JSON
            </button>
            <button
              className="button ghost"
              type="button"
              onClick={() => dispatch({ type: "discard-json-draft" })}
            >
              Discard draft
            </button>
          </div>
        </div>
      )}

      <main
        className={`workspace ${leftCollapsed ? "left-collapsed" : ""} ${
          rightCollapsed ? "right-collapsed" : ""
        }`}
      >
        <aside className={`navigator-pane ${leftCollapsed ? "collapsed" : ""}`}>
          <SceneOutline
            scene={state.scene}
            selectedObjectId={state.selectedObjectId}
            selectedMaterialId={state.selectedMaterialId}
            collapsed={leftCollapsed}
            disableMutations={state.jsonDirty}
            onSelectObject={(objectId) => dispatch({ type: "select-object", objectId })}
            onSelectMaterial={(materialId) =>
              dispatch({ type: "select-material", materialId })
            }
            onAddObject={handleAddObject}
            onAddMaterial={() => dispatch({ type: "add-material" })}
            onMoveObject={(objectId, direction) =>
              dispatch({ type: "move-object", objectId, direction })
            }
            onRemoveObject={(objectId) => dispatch({ type: "remove-object", objectId })}
            onRemoveMaterial={(materialId) =>
              dispatch({ type: "remove-material", materialId })
            }
            onToggleCollapsed={() => setLeftCollapsed((value) => !value)}
          />
        </aside>

        <section className="viewport-pane">
          <SceneViewport
            scene={state.scene}
            selectedObjectId={state.selectedObjectId}
            leftCollapsed={leftCollapsed}
            rightCollapsed={rightCollapsed}
          />
        </section>

        <section className={`inspector-pane ${rightCollapsed ? "collapsed" : ""}`}>
          <InspectorPanel
            scene={state.scene}
            selectedTab={state.selectedTab}
            selectedObject={selectedObject}
            selectedMaterial={selectedMaterial}
            jsonDraft={state.jsonDraft}
            jsonDirty={state.jsonDirty}
            jsonError={state.jsonError}
            collapsed={rightCollapsed}
            disableMutations={state.jsonDirty}
            onSelectTab={(tab) => dispatch({ type: "select-tab", tab })}
            onUpdateObject={(objectId, patch) =>
              dispatch({ type: "update-object", objectId, patch })
            }
            onChangeShape={(objectId, shape) =>
              dispatch({ type: "change-object-shape", objectId, shape })
            }
            onUpdateMaterial={(materialId, patch) =>
              dispatch({ type: "update-material", materialId, patch })
            }
            onUpdateCamera={(patch) => dispatch({ type: "update-camera", patch })}
            onJsonDraftChange={(value) =>
              dispatch({ type: "set-json-draft", value })
            }
            onApplyJson={handleApplyJson}
            onDiscardJson={() => dispatch({ type: "discard-json-draft" })}
            onToggleCollapsed={() => setRightCollapsed((value) => !value)}
          />
        </section>
      </main>
    </div>
  );
}
