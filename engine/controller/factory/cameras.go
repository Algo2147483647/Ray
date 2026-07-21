package factory

import (
	"errors"
	"fmt"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths"
	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

func ParseCameras(script *parser.Script) ([]modelcamera.Camera, error) {
	if len(script.Cameras) == 0 {
		return nil, errors.New("no cameras")
	}

	cameras := make([]modelcamera.Camera, 0, len(script.Cameras))
	for idx, cameraDef := range script.Cameras {
		parsedCamera, err := BuildCameraFromScript(cameraDef)
		if err != nil {
			return nil, fmt.Errorf("parse camera[%d]: %w", idx, err)
		}
		cameras = append(cameras, parsedCamera)
	}

	return cameras, nil
}

func BuildCameraFromScript(def parser.CameraScript) (modelcamera.Camera, error) {
	switch def.Type {
	case "", modelcamera.CameraType3D:
		return BuildCamera3DFromScript(def)
	case modelcamera.CameraTypeNDim:
		return BuildCameraNDimFromScript(def)
	case modelcamera.CameraTypeHyperbolic:
		return BuildHyperbolicCameraFromScript(def)
	case modelcamera.CameraTypeSpherical:
		return BuildSphericalCameraFromScript(def)
	default:
		return nil, fmt.Errorf("unsupported camera type %q", def.Type)
	}
}

func BuildCamera3DFromScript(def parser.CameraScript) (*modelcamera.Camera3D, error) {
	return &modelcamera.Camera3D{
		Position:     utils.NewVec(def.Position),
		Direction:    maths.Normalize(utils.NewVec(def.Direction)),
		Up:           utils.NewVec(def.Up),
		FieldOfViews: append([]float64(nil), def.FieldOfViews...),
		Ortho:        def.Ortho,
	}, nil
}

func BuildCameraNDimFromScript(def parser.CameraScript) (*modelcamera.CameraNDim, error) {
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
		vec := utils.NewVec(values)
		if mat.Norm(vec, 2) == 0 {
			return nil, fmt.Errorf("coordinates[%d] must not be zero", i)
		}
		coordinates[i] = vec
	}

	cameraNDim := &modelcamera.CameraNDim{
		Position:     utils.NewVec(def.Position),
		Coordinates:  coordinates,
		Width:        widths,
		FieldOfViews: fieldOfViews,
		Ortho:        def.Ortho,
	}
	if err := cameraNDim.Prepare(); err != nil {
		return nil, err
	}
	return cameraNDim, nil
}

func BuildHyperbolicCameraFromScript(def parser.CameraScript) (*modelcamera.HyperbolicCamera, error) {
	base, err := BuildCamera3DFromScript(def)
	if err != nil {
		return nil, err
	}
	return &modelcamera.HyperbolicCamera{Camera3D: *base}, nil
}

func BuildSphericalCameraFromScript(def parser.CameraScript) (*modelcamera.SphericalCamera, error) {
	cam := &modelcamera.SphericalCamera{
		Position:     utils.NewVec(def.Position),
		Forward:      utils.NewVec(def.Direction),
		Up:           utils.NewVec(def.Up),
		FieldOfViews: append([]float64(nil), def.FieldOfViews...),
	}
	if err := cam.Prepare(); err != nil {
		return nil, err
	}
	return cam, nil
}
