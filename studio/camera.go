package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/Algo2147483647/ray/engine/controller/parser"
)

var (
	defaultStudioCameraPosition = []float64{-1.7, 0.1, 0.5}
	defaultStudioCameraLookAt   = []float64{2, 0, 0}
	defaultStudioCameraUp       = []float64{0, 0, 1}
)

const (
	defaultStudioFieldOfView = 100.0
	defaultStudioAspectRatio = 1.0
)

func adaptCameras(cameraDefs []parser.CameraScript, dimension int) ([]parser.CameraScript, error) {
	if len(cameraDefs) == 0 {
		if dimension != 3 {
			return nil, nil
		}
		camera, err := adaptCamera3D(parser.CameraScript{}, dimension)
		if err != nil {
			return nil, err
		}
		return []parser.CameraScript{camera}, nil
	}

	cameras := make([]parser.CameraScript, len(cameraDefs))
	for idx, cameraDef := range cameraDefs {
		camera, err := adaptCamera(cameraDef, dimension)
		if err != nil {
			return nil, fmt.Errorf("adapt camera[%d]: %w", idx, err)
		}
		cameras[idx] = camera
	}
	return cameras, nil
}

func adaptCamera(def parser.CameraScript, dimension int) (parser.CameraScript, error) {
	switch strings.ToLower(def.Type) {
	case "", "3d", "camera3d", "hyperbolic", "klein":
		return adaptCamera3D(def, dimension)
	case "spherical", "s3":
		return adaptSphericalCamera(def, dimension)
	default:
		return cloneCamera(def), nil
	}
}

func adaptCamera3D(def parser.CameraScript, dimension int) (parser.CameraScript, error) {
	if dimension != 3 {
		return parser.CameraScript{}, fmt.Errorf("camera type %q requires render dimension 3, got %d", displayCameraType(def.Type), dimension)
	}

	position, err := cameraVector("position", def.Position, defaultStudioCameraPosition, dimension)
	if err != nil {
		return parser.CameraScript{}, err
	}
	up, err := cameraVector("up", def.Up, defaultStudioCameraUp, dimension)
	if err != nil {
		return parser.CameraScript{}, err
	}

	direction := append([]float64(nil), def.Direction...)
	if len(direction) == 0 {
		lookAt, err := cameraVector("look_at", def.LookAt, defaultStudioCameraLookAt, dimension)
		if err != nil {
			return parser.CameraScript{}, err
		}
		direction = subFloat64Slices(lookAt, position)
	} else if len(direction) != dimension {
		return parser.CameraScript{}, fmt.Errorf("field %q must contain %d values, got %d", "direction", dimension, len(direction))
	}
	if vectorNorm(direction) == 0 {
		return parser.CameraScript{}, fmt.Errorf("direction must not be zero")
	}

	camera := cloneCamera(def)
	if camera.Type == "" {
		camera.Type = "3d"
	}
	camera.Position = position
	camera.LookAt = nil
	camera.Direction = direction
	camera.Up = up
	camera.FieldOfView = positiveCameraValue(def.FieldOfView, defaultStudioFieldOfView)
	camera.AspectRatio = positiveCameraValue(def.AspectRatio, defaultStudioAspectRatio)
	return camera, nil
}

func adaptSphericalCamera(def parser.CameraScript, dimension int) (parser.CameraScript, error) {
	if dimension != 4 {
		return parser.CameraScript{}, fmt.Errorf("spherical camera requires render dimension 4, got %d", dimension)
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

func cloneCamera(def parser.CameraScript) parser.CameraScript {
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
