import type {
  SceneCamera,
  SceneDocument,
  SceneMaterial,
  SceneObject,
  ShapeType
} from "../types/scene";

const shapeTypes: ShapeType[] = [
  "cuboid",
  "sphere",
  "triangle",
  "plane",
  "quadratic equation",
  "four-order equation"
];

export function serializeScene(scene: SceneDocument): string {
  return JSON.stringify(scene, null, 2);
}

export function parseSceneText(text: string): SceneDocument {
  return normalizeScene(JSON.parse(text));
}

export function normalizeScene(input: unknown): SceneDocument {
  const source = isRecord(input) ? input : {};
  const materials = Array.isArray(source.materials)
    ? source.materials.map((material, index) => normalizeMaterial(material, index))
    : [normalizeMaterial({}, 0)];
  const fallbackMaterialId = materials[0]?.id ?? "Material-1";
  const objects = Array.isArray(source.objects)
    ? source.objects.map((object, index) =>
        normalizeObject(object, index, fallbackMaterialId)
      )
    : [];
  const cameras = Array.isArray(source.cameras)
    ? source.cameras.map((camera) => normalizeCamera(camera))
    : [normalizeCamera({})];

  return { materials, objects, cameras };
}

export function materialEmitsLight(material?: SceneMaterial | null): boolean {
  if (!material) {
    return false;
  }

  return Boolean(material.radiation || (material.radiate ?? 0) > 0);
}

export function getMaterialForObject(
  scene: SceneDocument,
  object: SceneObject
): SceneMaterial | undefined {
  return scene.materials.find((material) => material.id === object.material_id);
}

export function getPreviewSupport(shape: ShapeType): "exact" | "proxy" {
  if (
    shape === "quadratic equation" ||
    shape === "four-order equation"
  ) {
    return "proxy";
  }

  return "exact";
}

export function createDefaultMaterial(index: number): SceneMaterial {
  return {
    id: `Material-${index}`,
    color: [0.82, 0.72, 0.52],
    diffuse_loss: 1,
    reflect_loss: 0,
    refract_loss: 0,
    refractivity: 0
  };
}

export function getSceneStats(scene: SceneDocument) {
  const shapeCounts = scene.objects.reduce<Record<string, number>>((acc, object) => {
    acc[object.shape] = (acc[object.shape] ?? 0) + 1;
    return acc;
  }, {});

  const exactPreviewable = scene.objects.filter(
    (object) => getPreviewSupport(object.shape) === "exact"
  ).length;
  const proxyPreviewable = scene.objects.length - exactPreviewable;
  const lightSources = scene.objects.filter((object) =>
    materialEmitsLight(getMaterialForObject(scene, object))
  ).length;

  return {
    totalObjects: scene.objects.length,
    totalMaterials: scene.materials.length,
    lightSources,
    exactPreviewable,
    proxyPreviewable,
    shapeCounts
  };
}

export function estimateObjectCenter(object: SceneObject): [number, number, number] {
  if (object.position && object.position.length >= 3) {
    return toVec3(object.position);
  }

  if (object.shape === "triangle" && object.p1 && object.p2 && object.p3) {
    return [
      (toNumber(object.p1[0]) + toNumber(object.p2[0]) + toNumber(object.p3[0])) / 3,
      (toNumber(object.p1[1]) + toNumber(object.p2[1]) + toNumber(object.p3[1])) / 3,
      (toNumber(object.p1[2]) + toNumber(object.p2[2]) + toNumber(object.p3[2])) / 3
    ];
  }

  if (object.shape === "plane" && Array.isArray(object.A)) {
    const normal = toVec3(object.A);
    const squaredLength =
      normal[0] * normal[0] + normal[1] * normal[1] + normal[2] * normal[2];
    const offset = Array.isArray(object.b)
      ? toNumber(object.b[0])
      : toNumber(object.b);

    if (squaredLength > 0) {
      const scale = -offset / squaredLength;
      return [normal[0] * scale, normal[1] * scale, normal[2] * scale];
    }
  }

  return [0, 0, 0];
}

export function estimateProxyRadius(object: SceneObject): number {
  if (object.shape === "quadratic equation") {
    const scalar = Math.abs(toNumber(object.c, -90000));
    return Math.min(900, Math.max(180, Math.sqrt(Math.max(scalar, 1))));
  }

  if (object.shape === "four-order equation") {
    const coefficientCount = Array.isArray(object.a) ? object.a.length : 1;
    return Math.min(720, Math.max(220, 160 + coefficientCount * 12));
  }

  return 260;
}

export function formatNumberList(value: unknown): string {
  if (!Array.isArray(value)) {
    return "";
  }

  return value.join(", ");
}

export function parseNumberList(text: string): number[] | null {
  const trimmed = text.trim();
  if (!trimmed) {
    return [];
  }

  const values = trimmed
    .split(/[\s,\n]+/)
    .filter(Boolean)
    .map((item) => Number(item));

  return values.every((value) => Number.isFinite(value)) ? values : null;
}

export function toVec3(values: unknown): [number, number, number] {
  if (!Array.isArray(values)) {
    return [0, 0, 0];
  }

  return [
    toNumber(values[0]),
    toNumber(values[1]),
    toNumber(values[2])
  ];
}

export function toNumber(value: unknown, fallback = 0): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

function normalizeMaterial(input: unknown, index: number): SceneMaterial {
  const source = isRecord(input) ? input : {};
  const color = Array.isArray(source.color)
    ? source.color.map((value) => toNumber(value, 1))
    : [1, 1, 1];
  const refractive_index = Array.isArray(source.refractive_index)
    ? source.refractive_index.map((value) => toNumber(value, 1))
    : undefined;

  return {
    id: typeof source.id === "string" && source.id.trim() ? source.id : `Material-${index + 1}`,
    color,
    diffuse_loss: toOptionalNumber(source.diffuse_loss),
    reflect_loss: toOptionalNumber(source.reflect_loss),
    refract_loss: toOptionalNumber(source.refract_loss),
    reflectivity: toOptionalNumber(source.reflectivity),
    refractivity: toOptionalNumber(source.refractivity),
    refractive_index,
    radiate: toOptionalNumber(source.radiate),
    radiation: typeof source.radiation === "boolean" ? source.radiation : undefined,
    radiation_type:
      typeof source.radiation_type === "string" ? source.radiation_type : undefined
  };
}

function normalizeObject(
  input: unknown,
  index: number,
  fallbackMaterialId: string
): SceneObject {
  const source = isRecord(input) ? { ...input } : {};
  const shape = isShape(source.shape) ? source.shape : "cuboid";
  const object: SceneObject = {
    id:
      typeof source.id === "string" && source.id.trim()
        ? source.id
        : `${shape.replace(/\s+/g, "-")}-${index + 1}`,
    shape,
    material_id:
      typeof source.material_id === "string" && source.material_id.trim()
        ? source.material_id
        : fallbackMaterialId
  };

  for (const [key, value] of Object.entries(source)) {
    if (key === "id" || key === "shape" || key === "material_id") {
      continue;
    }

    object[key] = normalizeLooseValue(value);
  }

  return object;
}

function normalizeCamera(input: unknown): SceneCamera {
  const source = isRecord(input) ? input : {};

  return {
    position: toNumberArray(source.position, [0, 0, 3000]),
    direction: toNumberArray(source.direction, [0, 0, -1]),
    up: toNumberArray(source.up, [0, 1, 0]),
    width: toNumber(source.width, 800),
    height: toNumber(source.height, 600),
    field_of_view: toNumber(source.field_of_view, 90)
  };
}

function normalizeLooseValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map((entry) =>
      typeof entry === "number" && Number.isFinite(entry) ? entry : entry
    );
  }

  if (typeof value === "number") {
    return Number.isFinite(value) ? value : 0;
  }

  return value;
}

function toOptionalNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function toNumberArray(value: unknown, fallback: number[]): number[] {
  if (!Array.isArray(value)) {
    return [...fallback];
  }

  return [0, 1, 2].map((index) => toNumber(value[index], fallback[index]));
}

function isShape(value: unknown): value is ShapeType {
  return typeof value === "string" && shapeTypes.includes(value as ShapeType);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
