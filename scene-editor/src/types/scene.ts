export type Vec3 = [number, number, number];

export type ShapeType =
  | "cuboid"
  | "sphere"
  | "triangle"
  | "plane"
  | "quadratic equation"
  | "four-order equation";

export interface SceneMaterial {
  id: string;
  color: number[];
  diffuse_loss?: number;
  reflect_loss?: number;
  refract_loss?: number;
  reflectivity?: number;
  refractivity?: number;
  refractive_index?: number[];
  radiate?: number;
  radiation?: boolean;
  radiation_type?: string;
}

export interface SceneObject {
  id: string;
  shape: ShapeType;
  material_id: string;
  position?: number[];
  size?: number[];
  r?: number;
  p1?: number[];
  p2?: number[];
  p3?: number[];
  A?: number[];
  b?: number[] | number;
  a?: number[] | string;
  c?: number;
  [key: string]: unknown;
}

export interface SceneCamera {
  position: number[];
  direction: number[];
  up: number[];
  width: number;
  height: number;
  field_of_view: number;
}

export interface SceneDocument {
  materials: SceneMaterial[];
  objects: SceneObject[];
  cameras: SceneCamera[];
}
