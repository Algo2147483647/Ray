package controller

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
	"math"
	modelcamera "src-golang/model/camera"
	"src-golang/utils"
)

var (
	defaultCameraPosition = []float64{-1.7, 0.1, 0.5}
	defaultCameraLookAt   = []float64{2, 0, 0}
	defaultCameraUp       = []float64{0, 0, 1}
	defaultFieldOfView    = 100.0
)

func DefaultCameraScript() CameraScript {
	return CameraScript{
		Position:    append([]float64(nil), defaultCameraPosition...),
		LookAt:      append([]float64(nil), defaultCameraLookAt...),
		Up:          append([]float64(nil), defaultCameraUp...),
		FieldOfView: defaultFieldOfView,
		AspectRatio: 1,
	}
}

func BuildCamera3DFromScript(def CameraScript) (*modelcamera.Camera3D, error) {
	defaults := DefaultCameraScript()

	position, err := vectorFromScript("position", firstNonEmptyFloat64s(def.Position, defaults.Position))
	if err != nil {
		return nil, err
	}
	up, err := vectorFromScript("up", firstNonEmptyFloat64s(def.Up, defaults.Up))
	if err != nil {
		return nil, err
	}

	width := def.Width
	height := def.Height
	aspectRatio := def.AspectRatio
	if aspectRatio <= 0 && width > 0 && height > 0 {
		aspectRatio = float64(width) / float64(height)
	}
	if aspectRatio <= 0 {
		aspectRatio = defaults.AspectRatio
	}

	camera3D := &modelcamera.Camera3D{
		Position:    position,
		Up:          up,
		Width:       width,
		Height:      height,
		AspectRatio: aspectRatio,
		FieldOfView: positiveOrDefault(def.FieldOfView, defaults.FieldOfView),
		Ortho:       def.Ortho,
	}

	if len(def.Direction) > 0 {
		direction, err := vectorFromScript("direction", def.Direction)
		if err != nil {
			return nil, err
		}
		if mat.Norm(direction, 2) == 0 {
			return nil, fmt.Errorf("direction must not be zero")
		}
		camera3D.Direction = direction
		camera3D.Direction.ScaleVec(1.0/mat.Norm(direction, 2), camera3D.Direction)
		return camera3D, nil
	}

	lookAt, err := vectorFromScript("look_at", firstNonEmptyFloat64s(def.LookAt, defaults.LookAt))
	if err != nil {
		return nil, err
	}
	if distance(position, lookAt) == 0 {
		return nil, fmt.Errorf("look_at must differ from position")
	}

	camera3D.SetLookAt(lookAt)
	return camera3D, nil
}

func vectorFromScript(name string, values []float64) (*mat.VecDense, error) {
	if len(values) != utils.Dimension {
		return nil, fmt.Errorf("%s must contain %d values, got %d", name, utils.Dimension, len(values))
	}

	data := make([]float64, len(values))
	copy(data, values)
	return mat.NewVecDense(len(data), data), nil
}

func firstNonEmptyFloat64s(primary, fallback []float64) []float64 {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func positiveOrDefault(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

func distance(a, b *mat.VecDense) float64 {
	var sum float64
	for i := 0; i < a.Len(); i++ {
		diff := a.AtVec(i) - b.AtVec(i)
		sum += diff * diff
	}
	return math.Sqrt(sum)
}
