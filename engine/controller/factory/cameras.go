package factory

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

func ParseCameras(script *parser.Script) ([]modelcamera.Camera, error) {
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
	if err := validateCameraCoordinates(def.Coordinates, 3); err != nil {
		return nil, err
	}
	return &modelcamera.Camera3D{
		Position:     utils.NewVec(def.Position),
		Coordinates:  buildCameraCoordinates(def.Coordinates, 3),
		FieldOfViews: append([]float64(nil), def.FieldOfViews...),
		Ortho:        def.Ortho,
	}, nil
}

func BuildCameraNDimFromScript(def parser.CameraScript) (*modelcamera.CameraNDim, error) {
	fieldOfViews := append([]float64(nil), def.FieldOfViews...)
	if len(fieldOfViews) == 0 {
		return nil, fmt.Errorf("field_of_views is required for n_dim camera")
	}
	for i, fov := range fieldOfViews {
		if fov <= 0 {
			return nil, fmt.Errorf("field_of_views[%d] must be > 0", i)
		}
	}
	if len(def.Coordinates) != len(fieldOfViews)+1 {
		return nil, fmt.Errorf("coordinates count %d must equal field_of_views count + 1 (%d)", len(def.Coordinates), len(fieldOfViews)+1)
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
	if err := validateCameraCoordinates(def.Coordinates, 4); err != nil {
		return nil, err
	}
	cam := &modelcamera.SphericalCamera{
		Position:     utils.NewVec(def.Position),
		Coordinates:  buildCameraCoordinates(def.Coordinates, 4),
		FieldOfViews: append([]float64(nil), def.FieldOfViews...),
	}
	if err := cam.Prepare(); err != nil {
		return nil, err
	}
	return cam, nil
}

func buildCameraCoordinates(values [][]float64, dimension int) []*mat.VecDense {
	coordinates := make([]*mat.VecDense, len(values))
	for i, value := range values {
		coordinates[i] = utils.NewVec(value)
	}
	return coordinates
}

func validateCameraCoordinates(coordinates [][]float64, dimension int) error {
	if len(coordinates) != 3 {
		return fmt.Errorf("coordinates must contain forward, right, and up vectors")
	}
	for i, values := range coordinates {
		if len(values) != dimension {
			return fmt.Errorf("coordinates[%d] must contain %d values, got %d", i, dimension, len(values))
		}
		if mat.Norm(utils.NewVec(values), 2) == 0 {
			return fmt.Errorf("coordinates[%d] must not be zero", i)
		}
	}
	return nil
}
