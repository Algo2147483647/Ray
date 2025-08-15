package controller

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"gonum.org/v1/gonum/mat"
	"io"
	"os"
	"src-golang/math_lib"
	"src-golang/model"
	"src-golang/model/object"
	"src-golang/model/object/optics"
	"src-golang/model/object/shape"
)

type Script struct {
	Materials []ScriptMaterial         `json:"materials"`
	Objects   []map[string]interface{} `json:"objects"`
	Cameras   []*optics.Camera         `json:"camera"`
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
		material, exists := materials[cast.ToString(objDef["material_id"])]
		if !exists {
			continue // 跳过未定义材质的物体
		}

		switch objDef["shape"] {
		case "cuboid":
			if len(cast.ToIntSlice(objDef["size"])) < 3 {
				continue
			}

			position := mat.NewVecDense(3, cast.ToFloat64Slice(objDef["position"]))
			halfSize := math_lib.ScaleVec2(0.5, mat.NewVecDense(3, cast.ToFloat64Slice(objDef["size"])))
			pmax := mat.NewVecDense(3, nil)
			pmin := mat.NewVecDense(3, nil)
			pmax.AddVec(position, halfSize)
			pmin.SubVec(position, halfSize)

			scene.ObjectTree.AddObject(&object.Object{
				Shape:    shape.NewCuboid(pmin, pmax),
				Material: material,
			})

		case "sphere":
			scene.ObjectTree.AddObject(&object.Object{
				Shape: shape.NewSphere(
					mat.NewVecDense(3, cast.ToFloat64Slice(objDef["position"])),
					cast.ToFloat64(objDef["r"]),
				),
				Material: material,
			})

		case "triangle":
			scene.ObjectTree.AddObject(&object.Object{
				Shape: shape.NewTriangle(
					mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p1"])),
					mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p2"])),
					mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p3"])),
				),
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
