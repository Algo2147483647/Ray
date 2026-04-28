package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"src-golang/model"
	"src-golang/model/camera"
	"src-golang/model/object"
)

type CameraScript struct {
	ID          string    `json:"id"`
	Position    []float64 `json:"position"`
	LookAt      []float64 `json:"look_at"`
	Direction   []float64 `json:"direction"`
	Up          []float64 `json:"up"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	FieldOfView float64   `json:"field_of_view"`
	AspectRatio float64   `json:"aspect_ratio"`
	Ortho       bool      `json:"ortho"`
}

type RenderScript struct {
	Samples     int64  `json:"samples"`
	CameraIndex int    `json:"camera_index"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	OutputImage string `json:"output_image"`
	OutputFilm  string `json:"output_film"`
	DebugOutput string `json:"debug_output"`
}

type Script struct {
	Materials     []map[string]interface{} `json:"materials"`
	Objects       []map[string]interface{} `json:"objects"`
	Cameras       []CameraScript           `json:"cameras"`
	LegacyCameras []CameraScript           `json:"camera"`
	Render        RenderScript             `json:"render"`
}

func (s *Script) GetCameras() []CameraScript {
	if len(s.Cameras) > 0 {
		return s.Cameras
	}
	return s.LegacyCameras
}

func ReadScriptFile(filepath string) (*Script, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("open script %q: %w", filepath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read script %q: %w", filepath, err)
	}

	var script Script
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, fmt.Errorf("parse script %q: %w", filepath, err)
	}

	return &script, nil
}

func LoadSceneFromScript(script *Script, scene *model.Scene) error {
	if script == nil {
		return errors.New("script is nil")
	}

	scene.ObjectTree = &object.ObjectTree{}
	scene.Cameras = nil

	materials := ParseMaterials(script)

	for _, item := range script.Objects {
		materialID, _ := item["material_id"].(string)
		material, exists := materials[materialID]
		if !exists {
			continue
		}

		shapes := ParseShape(item)
		if len(shapes) == 0 {
			continue
		}

		for _, shape := range shapes {
			scene.ObjectTree.AddObject(&object.Object{
				Shape:    shape,
				Material: material,
			})
		}
	}

	cameras, err := ParseCameras(script)
	if err != nil {
		return err
	}
	scene.Cameras = append(scene.Cameras, cameras...)
	scene.ObjectTree.Build()
	return nil
}

func ParseCameras(script *Script) ([]camera.Camera, error) {
	cameraDefs := script.GetCameras()
	if len(cameraDefs) == 0 {
		return nil, nil
	}

	cameras := make([]camera.Camera, 0, len(cameraDefs))
	for idx, cameraDef := range cameraDefs {
		camera3D, err := BuildCamera3DFromScript(cameraDef)
		if err != nil {
			return nil, fmt.Errorf("parse camera[%d]: %w", idx, err)
		}
		cameras = append(cameras, camera3D)
	}

	return cameras, nil
}
