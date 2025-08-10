package controller

import (
	"encoding/json"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"io"
	"os"
	"src-golang/model/object"
)

type Script struct {
	Materials []ScriptMaterial `json:"materials"`
	Objects   []ScriptObject   `json:"objects"`
	Cameras   []ScriptCamera   `json:"camera"`
}

type ScriptCamera struct {
	ID        string    `json:"key"`
	Position  []float64 `json:"position"`
	Direction []float64 `json:"direction"`
	Up        []float64 `json:"up"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
}

type ScriptMaterial struct {
	ID           string  `json:"id"`
	Color        []int   `json:"color"`
	Diffuse      float64 `json:"diffuse,omitempty"`
	Reflect      float64 `json:"reflect,omitempty"`
	Refractivity float64 `json:"refractivity,omitempty"`
	Radiate      int     `json:"radiate,omitempty"`
}

type ScriptObject struct {
	ID         string    `json:"id"`
	Shape      string    `json:"shape"`
	Position   []float64 `json:"position"`
	Size       []float64 `json:"size,omitempty"`
	MaterialID string    `json:"material_id"`
}

func ReadScriptFile(filepath string) *Script {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return nil
	}
	defer file.Close()

	// 读取整个文件内容
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return nil
	}

	// 解析JSON到Script结构体
	var script Script
	if err := json.Unmarshal(data, &script); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return nil
	}

	return &script
}

// LoadSceneFromScript 从脚本加载场景
func LoadSceneFromScript(script *Script, objTree *object.ObjectTree) error {
	// 创建材质映射表
	materials := make(map[string]*object.Material)

	// 解析材质
	for _, matDef := range script.Materials {
		// 将颜色从 [0-255] 转换为 [0.0-1.0]
		r := float64(matDef.Color[0]) / 255.0
		g := float64(matDef.Color[1]) / 255.0
		b := float64(matDef.Color[2]) / 255.0

		material := object.NewMaterial(mat.NewVecDense(3, []float64{r, g, b}))

		// 设置材质属性
		if matDef.Diffuse != 0 {
			material.DiffuseLoss = matDef.Diffuse
		}
		if matDef.Reflect != 0 {
			material.Reflectivity = matDef.Reflect
		}
		if matDef.Refractivity != 0 {
			material.Refractivity = matDef.Refractivity
		}
		if matDef.Radiate != 0 {
			material.Radiation = matDef.Radiate != 0
		}

		materials[matDef.ID] = material
	}

	// 解析物体
	for _, objDef := range script.Objects {
		material, exists := materials[objDef.MaterialID]
		if !exists {
			continue // 跳过未定义材质的物体
		}

		position := mat.NewVecDense(3, []float64{
			float64(objDef.Position[0]),
			float64(objDef.Position[1]),
			float64(objDef.Position[2]),
		})

		switch objDef.Shape {
		case "cuboid":
			if len(objDef.Size) < 3 {
				continue
			}

			// 计算长方体对角点
			size := mat.NewVecDense(3, []float64{
				float64(objDef.Size[0]),
				float64(objDef.Size[1]),
				float64(objDef.Size[2]),
			})

			p2 := mat.NewVecDense(3, nil)
			p2.AddVec(position, size)

			objTree.AddObject(&object.Object{
				Shape:    object.NewCuboid(position, p2),
				Material: material,
			})

		case "sphere":
			radius := float64(objDef.Size[0])
			if radius == 0 {
				radius = 1.0 // 默认半径
			}

			objTree.AddObject(&object.Object{
				Shape:    object.NewSphere(position, radius),
				Material: material,
			})

		case "Plane":
			// 平面需要法线方向，这里使用位置向量作为法线
			//normal := position
			//if normal.Norm() == 0 {
			//	normal = mat.NewVecDense(3, []float64{0, 1, 0}) // 默认法线向上
			//} else {
			//	normal.ScaleVec(1/normal.Norm(), normal) // 归一化
			//}

			// XZH t.Add(NewPlane(position, normal), material)

		case "Cylinder":
			//height := float64(objDef.Size[0])
			//radius := float64(objDef.Size[1])
			//if height == 0 {
			//	height = 1.0
			//}
			//if radius == 0 {
			//	radius = 0.5
			//}
			//
			//// 圆柱体方向，默认为Y轴
			//axis := mat.NewVecDense(3, []float64{0, 1, 0})
			//if len(objDef.Size) > 2 {
			//	axis = mat.NewVecDense(3, []float64{
			//		float64(objDef.Size[0]),
			//		float64(objDef.Size[1]),
			//		float64(objDef.Size[2]),
			//	})
			//}
			//
			//t.Add(NewCylinder(position, axis, height, radius), material)
		}
	}
	objTree.Build()
	return nil
}
