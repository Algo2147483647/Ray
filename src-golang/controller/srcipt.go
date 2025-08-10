package controller

import (
	"encoding/json"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"io"
	"os"
	"src-golang/model"
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
	ID           string    `json:"id"`
	Color        []float64 `json:"color"`
	Diffuse      float64   `json:"diffuse,omitempty"`
	Reflect      float64   `json:"reflect,omitempty"`
	Refractivity float64   `json:"refractivity,omitempty"`
	Radiate      int       `json:"radiate,omitempty"`
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
func LoadSceneFromScript(script *Script, scene *model.Scene) error {
	materials := ParseMaterials(script) // 解析材质映射表

	// 解析物体
	for _, objDef := range script.Objects {
		position := mat.NewVecDense(3, objDef.Position)
		material, exists := materials[objDef.MaterialID]
		if !exists {
			continue // 跳过未定义材质的物体
		}

		switch objDef.Shape {
		case "cuboid":
			if len(objDef.Size) < 3 {
				continue
			}

			// 计算长方体对角点
			size := mat.NewVecDense(3, objDef.Size)
			pmax := mat.NewVecDense(3, nil)
			pmax.AddVec(position, size)
			pmin := mat.NewVecDense(3, nil)
			pmin.SubVec(position, size)

			scene.ObjectTree.AddObject(&object.Object{
				Shape:    object.NewCuboid(pmin, pmax),
				Material: material,
			})

		case "sphere":
			if len(objDef.Size) < 1 {
				continue
			}

			scene.ObjectTree.AddObject(&object.Object{
				Shape:    object.NewSphere(position, float64(objDef.Size[0])),
				Material: material,
			})

		case "plane":
		case "cylinder":
		}
	}
	scene.ObjectTree.Build()
	return nil
}

func ParseMaterials(script *Script) map[string]*object.Material {
	materials := make(map[string]*object.Material)

	for _, matDef := range script.Materials {
		r := float64(matDef.Color[0])
		g := float64(matDef.Color[1])
		b := float64(matDef.Color[2])

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

	return materials
}
