import { createObjectTemplate } from "../data/shapeSchemas";
import type { SceneDocument, SceneMaterial, SceneObject, ShapeType } from "../types/scene";
import { createDefaultMaterial, serializeScene } from "./scene-utils";

export type InspectorTab = "object" | "material" | "camera" | "json";

export interface EditorState {
  scene: SceneDocument;
  committedJson: string;
  jsonDraft: string;
  jsonDirty: boolean;
  jsonError: string | null;
  selectedObjectId: string | null;
  selectedMaterialId: string | null;
  selectedTab: InspectorTab;
}

export type EditorAction =
  | { type: "select-object"; objectId: string | null }
  | { type: "select-material"; materialId: string | null }
  | { type: "select-tab"; tab: InspectorTab }
  | { type: "set-json-draft"; value: string }
  | { type: "set-json-error"; error: string | null }
  | { type: "discard-json-draft" }
  | { type: "apply-json-scene"; scene: SceneDocument }
  | { type: "reset-scene"; scene: SceneDocument }
  | { type: "add-object"; shape: ShapeType }
  | { type: "remove-object"; objectId: string }
  | { type: "move-object"; objectId: string; direction: -1 | 1 }
  | { type: "update-object"; objectId: string; patch: Partial<SceneObject> }
  | { type: "change-object-shape"; objectId: string; shape: ShapeType }
  | { type: "add-material" }
  | { type: "remove-material"; materialId: string }
  | { type: "update-material"; materialId: string; patch: Partial<SceneMaterial> }
  | { type: "update-camera"; patch: Partial<SceneDocument["cameras"][number]> };

export function createInitialState(scene: SceneDocument): EditorState {
  const committedJson = serializeScene(scene);

  return {
    scene,
    committedJson,
    jsonDraft: committedJson,
    jsonDirty: false,
    jsonError: null,
    selectedObjectId: scene.objects[0]?.id ?? null,
    selectedMaterialId: scene.materials[0]?.id ?? null,
    selectedTab: "object"
  };
}

export function editorReducer(state: EditorState, action: EditorAction): EditorState {
  switch (action.type) {
    case "select-object":
      return {
        ...state,
        selectedObjectId: action.objectId,
        selectedTab: action.objectId ? "object" : state.selectedTab
      };
    case "select-material":
      return {
        ...state,
        selectedMaterialId: action.materialId,
        selectedTab: action.materialId ? "material" : state.selectedTab
      };
    case "select-tab":
      return { ...state, selectedTab: action.tab };
    case "set-json-draft":
      return {
        ...state,
        jsonDraft: action.value,
        jsonDirty: action.value !== state.committedJson,
        jsonError: null
      };
    case "set-json-error":
      return {
        ...state,
        jsonError: action.error
      };
    case "discard-json-draft":
      return {
        ...state,
        jsonDraft: state.committedJson,
        jsonDirty: false,
        jsonError: null
      };
    case "apply-json-scene":
      return syncSceneState(state, action.scene, {
        selectedObjectId: action.scene.objects[0]?.id ?? null,
        selectedMaterialId: action.scene.materials[0]?.id ?? null
      });
    case "reset-scene":
      return syncSceneState(state, action.scene, {
        selectedObjectId: action.scene.objects[0]?.id ?? null,
        selectedMaterialId: action.scene.materials[0]?.id ?? null,
        selectedTab: "object"
      });
    case "add-object": {
      const nextScene = structuredClone(state.scene);
      const materialId = nextScene.materials[0]?.id ?? "Material-1";
      const nextObject = createObjectTemplate(
        action.shape,
        nextScene.objects.length + 1,
        materialId
      );
      nextScene.objects.push(nextObject);

      return syncSceneState(state, nextScene, {
        selectedObjectId: nextObject.id,
        selectedTab: "object"
      });
    }
    case "remove-object": {
      const nextScene = structuredClone(state.scene);
      nextScene.objects = nextScene.objects.filter((object) => object.id !== action.objectId);
      const selectedObjectId =
        state.selectedObjectId === action.objectId
          ? nextScene.objects[0]?.id ?? null
          : state.selectedObjectId;

      return syncSceneState(state, nextScene, { selectedObjectId });
    }
    case "move-object": {
      const nextScene = structuredClone(state.scene);
      const index = nextScene.objects.findIndex((object) => object.id === action.objectId);
      const target = index + action.direction;

      if (index < 0 || target < 0 || target >= nextScene.objects.length) {
        return state;
      }

      const [movedObject] = nextScene.objects.splice(index, 1);
      nextScene.objects.splice(target, 0, movedObject);
      return syncSceneState(state, nextScene);
    }
    case "update-object": {
      const nextScene = structuredClone(state.scene);
      nextScene.objects = nextScene.objects.map((object) =>
        object.id === action.objectId ? { ...object, ...action.patch } : object
      );
      return syncSceneState(state, nextScene);
    }
    case "change-object-shape": {
      const nextScene = structuredClone(state.scene);
      nextScene.objects = nextScene.objects.map((object, index) => {
        if (object.id !== action.objectId) {
          return object;
        }

        const template = createObjectTemplate(
          action.shape,
          index + 1,
          object.material_id
        );
        return {
          ...template,
          id: object.id,
          material_id: object.material_id
        };
      });
      return syncSceneState(state, nextScene);
    }
    case "add-material": {
      const nextScene = structuredClone(state.scene);
      const nextMaterial = createDefaultMaterial(nextScene.materials.length + 1);
      nextScene.materials.push(nextMaterial);

      return syncSceneState(state, nextScene, {
        selectedMaterialId: nextMaterial.id,
        selectedTab: "material"
      });
    }
    case "remove-material": {
      const nextScene = structuredClone(state.scene);
      if (nextScene.materials.length <= 1) {
        return state;
      }

      nextScene.materials = nextScene.materials.filter(
        (material) => material.id !== action.materialId
      );
      const fallbackMaterialId = nextScene.materials[0].id;
      nextScene.objects = nextScene.objects.map((object) =>
        object.material_id === action.materialId
          ? { ...object, material_id: fallbackMaterialId }
          : object
      );

      const selectedMaterialId =
        state.selectedMaterialId === action.materialId
          ? fallbackMaterialId
          : state.selectedMaterialId;

      return syncSceneState(state, nextScene, { selectedMaterialId });
    }
    case "update-material": {
      const nextScene = structuredClone(state.scene);
      nextScene.materials = nextScene.materials.map((material) =>
        material.id === action.materialId ? { ...material, ...action.patch } : material
      );
      return syncSceneState(state, nextScene);
    }
    case "update-camera": {
      const nextScene = structuredClone(state.scene);
      nextScene.cameras[0] = {
        ...nextScene.cameras[0],
        ...action.patch
      };
      return syncSceneState(state, nextScene);
    }
  }
}

function syncSceneState(
  state: EditorState,
  scene: SceneDocument,
  overrides: Partial<EditorState> = {}
): EditorState {
  const committedJson = serializeScene(scene);

  return {
    ...state,
    scene,
    committedJson,
    jsonDraft: committedJson,
    jsonDirty: false,
    jsonError: null,
    ...overrides
  };
}
