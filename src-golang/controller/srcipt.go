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
	"src-golang/model/object/shape"
)

type Script struct {
	Materials []map[string]interface{} `json:"materials"`
	Objects   []map[string]interface{} `json:"objects"`
	Cameras   []map[string]interface{} `json:"camera"`
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

	for _, item := range script.Objects { // 解析物体
		material, exists := materials[cast.ToString(item["material_id"])]
		if !exists {
			continue // 跳过未定义材质的物体
		}

		shape := ParseShape(item)
		if shape == nil {
			continue
		}

		scene.ObjectTree.AddObject(&object.Object{
			Shape:    shape,
			Material: material,
		})
	}

	scene.ObjectTree.Build()
	return nil
}

func ParseShape(objDef map[string]interface{}) shape.Shape {
	switch objDef["shape"] {
	case "cuboid":
		if _, ok := objDef["position"]; ok {
			position := mat.NewVecDense(3, cast.ToFloat64Slice(objDef["position"]))
			halfSize := math_lib.ScaleVec2(0.5, mat.NewVecDense(3, cast.ToFloat64Slice(objDef["size"])))
			pmax := mat.NewVecDense(3, nil)
			pmin := mat.NewVecDense(3, nil)
			pmax.AddVec(position, halfSize)
			pmin.SubVec(position, halfSize)
			return shape.NewCuboid(pmin, pmax)
		}

		if _, ok := objDef["pmax"]; ok {
			return shape.NewCuboid(
				mat.NewVecDense(3, cast.ToFloat64Slice(objDef["pmin"])),
				mat.NewVecDense(3, cast.ToFloat64Slice(objDef["pmax"])),
			)
		}

	case "sphere":
		return shape.NewSphere(
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["position"])),
			cast.ToFloat64(objDef["r"]),
		)

	case "triangle":
		return shape.NewTriangle(
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p1"])),
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p2"])),
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p3"])),
		)

	case "plane":
	case "cylinder":
	}
	return nil
}

func ParseMaterials(script *Script) map[string]*object.Material {
	materials := make(map[string]*object.Material)

	for _, matDef := range script.Materials {
		material := object.NewMaterial(mat.NewVecDense(3, cast.ToFloat64Slice(matDef["color"])))

		if val, ok := matDef["radiate"]; ok {
			material.Radiation = cast.ToBool(val)
		}
		if val, ok := matDef["reflectivity"]; ok {
			material.ReflectLoss = cast.ToFloat64(val)
		}
		if val, ok := matDef["refractivity"]; ok {
			material.RefractLoss = cast.ToFloat64(val)
		}
		if val, ok := matDef["refractive_index"]; ok {
			material.RefractiveIndex = cast.ToFloat64(val)
		}
		if val, ok := matDef["diffuse_loss"]; ok {
			material.DiffuseLoss = cast.ToFloat64(val)
		}
		if val, ok := matDef["reflect_loss"]; ok {
			material.ReflectLoss = cast.ToFloat64(val)
		}
		if val, ok := matDef["refract_loss"]; ok {
			material.RefractLoss = cast.ToFloat64(val)
		}
		materials[cast.ToString(matDef["id"])] = material
	}

	return materials
}
