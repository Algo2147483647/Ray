package factory

import (
	"fmt"
	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
	"strings"
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
	if utils.Dimension != 3 {
		return nil, fmt.Errorf("camera type %q requires render dimension 3, got %d", "3d", utils.Dimension)
	}

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

func BuildCameraNDimFromScript(def CameraScript) (*modelcamera.CameraNDim, error) {
	if len(def.Position) == 0 {
		return nil, fmt.Errorf("position is required for n_dim camera")
	}
	position, err := vectorFromScript("position", def.Position)
	if err != nil {
		return nil, err
	}
	if len(def.Widths) == 0 {
		return nil, fmt.Errorf("widths is required for n_dim camera")
	}
	widths := append([]int(nil), def.Widths...)
	for i, width := range widths {
		if width <= 0 {
			return nil, fmt.Errorf("widths[%d] must be > 0", i)
		}
	}

	fieldOfViews := append([]float64(nil), def.FieldOfViews...)
	if len(fieldOfViews) == 0 && def.FieldOfView > 0 {
		fieldOfViews = make([]float64, len(widths))
		for i := range fieldOfViews {
			fieldOfViews[i] = def.FieldOfView
		}
	}
	if len(fieldOfViews) != len(widths) {
		return nil, fmt.Errorf("field_of_views count %d must match widths count %d", len(fieldOfViews), len(widths))
	}
	for i, fov := range fieldOfViews {
		if fov <= 0 {
			return nil, fmt.Errorf("field_of_views[%d] must be > 0", i)
		}
	}
	if len(def.Coordinates) != len(widths)+1 {
		return nil, fmt.Errorf("coordinates count %d must equal widths count + 1 (%d)", len(def.Coordinates), len(widths)+1)
	}

	coordinates := make([]*mat.VecDense, len(def.Coordinates))
	for i, values := range def.Coordinates {
		vec, err := vectorFromScript(fmt.Sprintf("coordinates[%d]", i), values)
		if err != nil {
			return nil, err
		}
		if mat.Norm(vec, 2) == 0 {
			return nil, fmt.Errorf("coordinates[%d] must not be zero", i)
		}
		coordinates[i] = vec
	}

	cameraNDim := &modelcamera.CameraNDim{
		Position:    position,
		Coordinates: coordinates,
		Width:       widths,
		FieldOfView: fieldOfViews,
		Ortho:       def.Ortho,
	}
	if err := cameraNDim.Prepare(); err != nil {
		return nil, err
	}
	return cameraNDim, nil
}

func BuildCameraFromScript(def CameraScript) (modelcamera.Camera, error) {
	switch strings.ToLower(def.Type) {
	case "", "3d", "camera3d":
		return BuildCamera3DFromScript(def)
	case "n_dim", "ndim", "n-dimensional":
		return BuildCameraNDimFromScript(def)
	default:
		return nil, fmt.Errorf("unsupported camera type %q", def.Type)
	}
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
