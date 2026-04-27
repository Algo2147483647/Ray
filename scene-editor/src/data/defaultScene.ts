import type { SceneDocument } from "../types/scene";

export const defaultScene: SceneDocument = {
  materials: [
    {
      id: "Paper",
      color: [1, 1, 1],
      diffuse_loss: 1,
      reflect_loss: 0,
      refract_loss: 0,
      refractivity: 0
    },
    {
      id: "Glass",
      color: [1, 1, 1],
      diffuse_loss: 0,
      reflect_loss: 1,
      refract_loss: 0,
      refractivity: 1.7
    },
    {
      id: "Metal",
      color: [1, 0.9, 0.35],
      diffuse_loss: 0.5,
      reflect_loss: 0.5,
      refract_loss: 0,
      reflectivity: 0.75,
      refractivity: 0
    },
    {
      id: "Light",
      color: [10, 10, 10],
      radiate: 1
    }
  ],
  objects: [
    {
      id: "box1",
      shape: "cuboid",
      position: [0, 0, 0],
      size: [2000, 2000, 2000],
      material_id: "Paper"
    },
    {
      id: "glass_panel1",
      shape: "cuboid",
      position: [850, 1320, 0],
      size: [1150, 1350, 300],
      material_id: "Glass"
    },
    {
      id: "glass_panel2",
      shape: "cuboid",
      position: [850, 900, 0],
      size: [1150, 930, 300],
      material_id: "Glass"
    },
    {
      id: "light_source",
      shape: "sphere",
      position: [1000, 1000, 1600],
      r: 400,
      material_id: "Light"
    }
  ],
  cameras: [
    {
      position: [0, 0, 3000],
      direction: [0, 0, -1],
      up: [0, 1, 0],
      width: 800,
      height: 600,
      field_of_view: 90
    }
  ]
};
