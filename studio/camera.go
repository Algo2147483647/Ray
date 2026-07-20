package main

import (
	"fmt"
	"math"
	"strings"
)

var (
	defaultStudioCameraPosition  = []float64{0, 0, 0}
	defaultStudioCameraDirection = []float64{1, 0, 0}
	defaultStudioCameraUp        = []float64{0, 0, 1}
)

const (
	defaultStudioFieldOfView = 100.0
	defaultStudioAspectRatio = 1.0
)

type studioCameraScript struct {
	ID           string      `json:"id"`
	Type         string      `json:"type"`
	Position     []float64   `json:"position"`
	LookAt       []float64   `json:"look_at"`
	Direction    []float64   `json:"direction"`
	Up           []float64   `json:"up"`
	Widths       []int       `json:"widths"`
	FieldOfView  float64     `json:"field_of_view"`
	FieldOfViews []float64   `json:"field_of_views"`
	Coordinates  [][]float64 `json:"coordinates"`
	AspectRatio  float64     `json:"aspect_ratio"`
	Ortho        bool        `json:"ortho"`
}

type engineCameraScript struct {
	ID           string      `json:"id,omitempty"`
	Type         string      `json:"type,omitempty"`
	Position     []float64   `json:"position,omitempty"`
	Direction    []float64   `json:"direction,omitempty"`
	Up           []float64   `json:"up,omitempty"`
	Widths       []int       `json:"widths,omitempty"`
	FieldOfView  float64     `json:"field_of_view,omitempty"`
	FieldOfViews []float64   `json:"field_of_views,omitempty"`
	Coordinates  [][]float64 `json:"coordinates,omitempty"`
	AspectRatio  float64     `json:"aspect_ratio,omitempty"`
	Ortho        bool        `json:"ortho,omitempty"`
}

func adaptCameras(cameraDefs []studioCameraScript, dimension int) ([]engineCameraScript, error) {
	if len(cameraDefs) == 0 {
		if dimension != 3 {
			return nil, nil
		}
		camera, err := adaptCamera3D(studioCameraScript{}, dimension)
		if err != nil {
			return nil, err
		}
		return []engineCameraScript{camera}, nil
	}

	cameras := make([]engineCameraScript, len(cameraDefs))
	for idx, cameraDef := range cameraDefs {
		camera, err := adaptCamera(cameraDef, dimension)
		if err != nil {
			return nil, fmt.Errorf("adapt camera[%d]: %w", idx, err)
		}
		cameras[idx] = camera
	}
	return cameras, nil
}

func adaptCamera(def studioCameraScript, dimension int) (engineCameraScript, error) {
	switch strings.ToLower(def.Type) {
	case "", "3d", "camera3d", "hyperbolic", "klein":
		return adaptCamera3D(def, dimension)
	case "spherical", "s3":
		return adaptSphericalCamera(def, dimension)
	default:
		return cloneCamera(def), nil
	}
}

func adaptCamera3D(def studioCameraScript, dimension int) (engineCameraScript, error) {
	if dimension != 3 {
		return engineCameraScript{}, fmt.Errorf("camera type %q requires render dimension 3, got %d", displayCameraType(def.Type), dimension)
	}

	position, err := cameraVector("position", def.Position, defaultStudioCameraPosition, dimension)
	if err != nil {
		return engineCameraScript{}, err
	}
	up, err := cameraVector("up", def.Up, defaultStudioCameraUp, dimension)
	if err != nil {
		return engineCameraScript{}, err
	}

	direction := append([]float64(nil), def.Direction...)
	if len(direction) == 0 {
		if len(def.LookAt) > 0 {
			if len(def.LookAt) != dimension {
				return engineCameraScript{}, fmt.Errorf("field %q must contain %d values, got %d", "look_at", dimension, len(def.LookAt))
			}
			direction = subFloat64Slices(def.LookAt, position)
		} else {
			direction = append([]float64(nil), defaultStudioCameraDirection...)
		}
	} else if len(direction) != dimension {
		return engineCameraScript{}, fmt.Errorf("field %q must contain %d values, got %d", "direction", dimension, len(direction))
	}
	if vectorNorm(direction) == 0 {
		return engineCameraScript{}, fmt.Errorf("direction must not be zero")
	}

	camera := cloneCamera(def)
	if camera.Type == "" {
		camera.Type = "3d"
	}
	camera.Position = position
	camera.Direction = direction
	camera.Up = up
	camera.FieldOfView = positiveCameraValue(def.FieldOfView, defaultStudioFieldOfView)
	camera.AspectRatio = positiveCameraValue(def.AspectRatio, defaultStudioAspectRatio)
	return camera, nil
}

func adaptSphericalCamera(def studioCameraScript, dimension int) (engineCameraScript, error) {
	if dimension != 4 {
		return engineCameraScript{}, fmt.Errorf("spherical camera requires render dimension 4, got %d", dimension)
	}
	camera := cloneCamera(def)
	camera.FieldOfView = positiveCameraValue(def.FieldOfView, defaultStudioFieldOfView)
	camera.AspectRatio = positiveCameraValue(def.AspectRatio, defaultStudioAspectRatio)
	return camera, nil
}

func cameraVector(name string, values, fallback []float64, dimension int) ([]float64, error) {
	if len(values) == 0 {
		return append([]float64(nil), fallback...), nil
	}
	if len(values) != dimension {
		return nil, fmt.Errorf("field %q must contain %d values, got %d", name, dimension, len(values))
	}
	return append([]float64(nil), values...), nil
}

func positiveCameraValue(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

func subFloat64Slices(a, b []float64) []float64 {
	result := make([]float64, len(a))
	for i := range a {
		result[i] = a[i] - b[i]
	}
	return result
}

func vectorNorm(values []float64) float64 {
	sum := 0.0
	for _, value := range values {
		sum += value * value
	}
	return math.Sqrt(sum)
}

func cloneCamera(def studioCameraScript) engineCameraScript {
	camera := engineCameraScript{
		ID:          def.ID,
		Type:        def.Type,
		FieldOfView: def.FieldOfView,
		AspectRatio: def.AspectRatio,
		Ortho:       def.Ortho,
	}
	camera.Position = append([]float64(nil), def.Position...)
	camera.Direction = append([]float64(nil), def.Direction...)
	camera.Up = append([]float64(nil), def.Up...)
	camera.Widths = append([]int(nil), def.Widths...)
	camera.FieldOfViews = append([]float64(nil), def.FieldOfViews...)
	if len(def.Coordinates) > 0 {
		camera.Coordinates = make([][]float64, len(def.Coordinates))
		for i, coordinate := range def.Coordinates {
			camera.Coordinates[i] = append([]float64(nil), coordinate...)
		}
	}
	return camera
}

func cloneStudioCamera(def studioCameraScript) studioCameraScript {
	camera := def
	camera.Position = append([]float64(nil), def.Position...)
	camera.LookAt = append([]float64(nil), def.LookAt...)
	camera.Direction = append([]float64(nil), def.Direction...)
	camera.Up = append([]float64(nil), def.Up...)
	camera.Widths = append([]int(nil), def.Widths...)
	camera.FieldOfViews = append([]float64(nil), def.FieldOfViews...)
	if len(def.Coordinates) > 0 {
		camera.Coordinates = make([][]float64, len(def.Coordinates))
		for i, coordinate := range def.Coordinates {
			camera.Coordinates[i] = append([]float64(nil), coordinate...)
		}
	}
	return camera
}

func displayCameraType(value string) string {
	if value == "" {
		return "3d"
	}
	return value
}
