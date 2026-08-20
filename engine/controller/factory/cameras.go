package factory

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils"
)

func ParseCameras(script *parser.Script) (map[string]modelcamera.RayCamera, error) {
	cameras := make(map[string]modelcamera.RayCamera, len(script.Cameras))
	for index, def := range script.Cameras {
		if def.ID == "" {
			return nil, fmt.Errorf("parse camera[%d]: id is required", index)
		}
		if _, exists := cameras[def.ID]; exists {
			return nil, fmt.Errorf("parse camera[%d]: duplicate id %q", index, def.ID)
		}
		if def.Film == nil {
			return nil, fmt.Errorf("parse camera[%d] %q: film is required", index, def.ID)
		}
		if len(def.Film.Shape) != len(def.FieldOfViews) || def.Film.ElementCount() == 0 {
			return nil, fmt.Errorf("parse camera[%d] %q: film shape must match field_of_views", index, def.ID)
		}
		if _, err := modelcamera.NormalizePixelWindows(def.Film.PixelWindows, def.Film.Shape); err != nil {
			return nil, fmt.Errorf("parse camera[%d] %q: %w", index, def.ID, err)
		}
		binCount := def.Film.SpectralBinCount
		if binCount < 0 || binCount > modelcamera.MaxSpectralBinCount {
			return nil, fmt.Errorf("parse camera[%d] %q: film spectral_bin_count must be between 0 and %d", index, def.ID, modelcamera.MaxSpectralBinCount)
		}
		if !def.Film.HasSpectralBins() {
			if binCount == 0 {
				binCount = modelcamera.DefaultSpectralBinCount
			}
			def.Film.InitSpectralBins(binCount, optics.WavelengthMin, optics.WavelengthMax)
		} else if binCount > 0 && binCount != len(def.Film.SpectralBins) {
			return nil, fmt.Errorf("parse camera[%d] %q: film spectral_bin_count %d does not match %d supplied spectral bins", index, def.ID, binCount, len(def.Film.SpectralBins))
		}

		parsed, err := BuildCameraFromScript(def)
		if err != nil {
			return nil, fmt.Errorf("parse camera[%d] %q: %w", index, def.ID, err)
		}
		prepared, ok := parsed.(interface{ Prepare() error })
		if !ok {
			return nil, fmt.Errorf("parse camera[%d] %q: camera does not support preparation", index, def.ID)
		}
		if err := prepared.Prepare(); err != nil {
			return nil, fmt.Errorf("parse camera[%d] %q: %w", index, def.ID, err)
		}
		cameras[def.ID] = parsed
	}
	return cameras, nil
}

func BuildCameraFromScript(def parser.CameraScript) (modelcamera.RayCamera, error) {
	coordinates := utils.NewVecs(def.Coordinates)
	film := modelcamera.Camera{Film: def.Film}

	switch def.Type {
	case "", modelcamera.CameraType3D:
		return &modelcamera.Camera3D{
			Camera:       film,
			Position:     utils.NewVec(def.Position),
			Coordinates:  coordinates,
			FieldOfViews: append([]float64(nil), def.FieldOfViews...),
			Ortho:        def.Ortho,
		}, nil

	case modelcamera.CameraTypeNDim:
		return &modelcamera.CameraNDim{
			Camera:       film,
			Position:     utils.NewVec(def.Position),
			Coordinates:  coordinates,
			FieldOfViews: append([]float64(nil), def.FieldOfViews...),
			Ortho:        def.Ortho,
		}, nil

	case modelcamera.CameraTypeHyperbolic:
		return &modelcamera.HyperbolicCamera{Camera3D: modelcamera.Camera3D{
			Camera:       film,
			Position:     utils.NewVec(def.Position),
			Coordinates:  coordinates,
			FieldOfViews: append([]float64(nil), def.FieldOfViews...),
			Ortho:        def.Ortho,
		}}, nil

	case modelcamera.CameraTypeSpherical:
		return &modelcamera.SphericalCamera{
			Camera:       film,
			Position:     utils.NewVec(def.Position),
			Coordinates:  coordinates,
			FieldOfViews: append([]float64(nil), def.FieldOfViews...),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported camera type %q", def.Type)
	}
}
