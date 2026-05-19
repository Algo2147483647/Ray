import type { SceneObject, ShapeType } from "../types/scene";

export type FieldKind = "number" | "vector3" | "numberList";

export interface ShapeField {
  key: keyof SceneObject;
  label: string;
  kind: FieldKind;
  hint: string;
}

export const shapeSchemas: Record<ShapeType, ShapeField[]> = {
  cuboid: [
    { key: "position", label: "Position", kind: "vector3", hint: "Center point" },
    { key: "size", label: "Size", kind: "vector3", hint: "Width, height, depth" }
  ],
  sphere: [
    { key: "position", label: "Position", kind: "vector3", hint: "Sphere center" },
    { key: "r", label: "Radius", kind: "number", hint: "Sphere radius" }
  ],
  triangle: [
    { key: "p1", label: "P1", kind: "vector3", hint: "First vertex" },
    { key: "p2", label: "P2", kind: "vector3", hint: "Second vertex" },
    { key: "p3", label: "P3", kind: "vector3", hint: "Third vertex" }
  ],
  plane: [
    { key: "A", label: "Normal", kind: "vector3", hint: "Plane normal coefficients" },
    { key: "b", label: "Offset", kind: "number", hint: "Equation constant in A·x + b = 0" }
  ],
  "quadratic equation": [
    { key: "position", label: "Anchor", kind: "vector3", hint: "Preview anchor point" },
    { key: "a", label: "Matrix A", kind: "numberList", hint: "Nine values, row-major" },
    { key: "b", label: "Vector b", kind: "vector3", hint: "Linear term" },
    { key: "c", label: "Scalar c", kind: "number", hint: "Constant term" }
  ],
  "four-order equation": [
    { key: "position", label: "Anchor", kind: "vector3", hint: "Preview anchor point" },
    { key: "a", label: "Coefficient list", kind: "numberList", hint: "Flat coefficient array" }
  ]
};

export const shapeLabels: Record<ShapeType, string> = {
  cuboid: "Cuboid",
  sphere: "Sphere",
  triangle: "Triangle",
  plane: "Plane",
  "quadratic equation": "Quadratic Surface",
  "four-order equation": "Fourth-order Surface"
};

export function createObjectTemplate(
  shape: ShapeType,
  index: number,
  materialId: string
): SceneObject {
  const idBase = shape.replace(/\s+/g, "-");

  switch (shape) {
    case "cuboid":
      return {
        id: `${idBase}-${index}`,
        shape,
        position: [0, 0, 0],
        size: [500, 500, 500],
        material_id: materialId
      };
    case "sphere":
      return {
        id: `${idBase}-${index}`,
        shape,
        position: [0, 0, 0],
        r: 250,
        material_id: materialId
      };
    case "triangle":
      return {
        id: `${idBase}-${index}`,
        shape,
        p1: [-300, -200, 0],
        p2: [300, -200, 0],
        p3: [0, 280, 0],
        material_id: materialId
      };
    case "plane":
      return {
        id: `${idBase}-${index}`,
        shape,
        A: [0, 1, 0],
        b: 0,
        material_id: materialId
      };
    case "quadratic equation":
      return {
        id: `${idBase}-${index}`,
        shape,
        position: [0, 0, 0],
        a: [1, 0, 0, 0, 1, 0, 0, 0, 1],
        b: [0, 0, 0],
        c: -90000,
        material_id: materialId
      };
    case "four-order equation":
      return {
        id: `${idBase}-${index}`,
        shape,
        position: [0, 0, 0],
        a: [1, 0, 0, 0, 1, 0, 0, 1],
        material_id: materialId
      };
  }
}
