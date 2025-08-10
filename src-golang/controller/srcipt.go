package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Script struct {
	Materials []ScriptMaterial `json:"materials"`
	Objects   []ScriptObject   `json:"objects"`
}

type ScriptMaterial struct {
	Key          string  `json:"key"`
	Color        []int   `json:"color"`
	Diffuse      float64 `json:"diffuse,omitempty"`
	Reflect      float64 `json:"reflect,omitempty"`
	Refractivity float64 `json:"refractivity,omitempty"`
	Radiate      int     `json:"radiate,omitempty"`
}

type ScriptObject struct {
	Key      string `json:"key"`
	Shape    string `json:"shape"`
	Position []int  `json:"position"`
	Size     []int  `json:"size,omitempty"`
	Material string `json:"material"`
	Radius   int    `json:"radius,omitempty"`
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
