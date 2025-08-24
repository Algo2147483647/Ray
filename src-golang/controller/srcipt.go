package controller

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"io"
	"os"
	"src-golang/model"
	"src-golang/model/object"
)

type Script struct {
	Materials []map[string]interface{} `json:"materials"`
	Objects   []map[string]interface{} `json:"objects"`
	Cameras   []map[string]interface{} `json:"camera"`
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

		shapes := ParseShape(item)
		if shapes == nil || len(shapes) == 0 {
			continue
		}

		for _, shape := range shapes {
			scene.ObjectTree.AddObject(&object.Object{
				Shape:    *shape,
				Material: material,
			})
		}
	}

	scene.ObjectTree.Build()
	return nil
}
