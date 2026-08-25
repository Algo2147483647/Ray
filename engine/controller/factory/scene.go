package factory

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model"
	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
)

func LoadSceneFromScript(script *parser.Script, scene *model.Scene) error {
	if script == nil {
		return errors.New("script is nil")
	}
	if scene == nil {
		return errors.New("scene is nil")
	}

	scene.ObjectTree = &object.ObjectTree{}
	scene.Cameras = make(map[string]modelcamera.RayCamera)
	scene.Space = geometry.DefaultSceneSpace()
	scene.MaxArc = 0

	dimension, err := sceneDimension(script)
	if err != nil {
		return err
	}
	if dimension < 2 {
		return fmt.Errorf("scene dimension must be >= 2, got %d", dimension)
	}

	// Geometry is always explicit. Euclidean owns the Scene's actual embedding
	// dimension instead of using nil as a process-wide compatibility sentinel.
	sceneGeometry := geometry.Euclidean(dimension)
	if script.Geometry != nil {
		switch strings.ToLower(script.Geometry.Type) {
		case "", "euclidean":
			sceneGeometry = geometry.Euclidean(dimension)
		case "klein", "hyperbolic":
			sceneGeometry = geometry.Klein()
		case "spherical", "sphere":
			sceneGeometry = geometry.Spherical()
		default:
			return fmt.Errorf("unsupported geometry type %q", script.Geometry.Type)
		}
		scene.MaxArc = script.Geometry.MaxArc
		if scene.MaxArc < 0 || math.IsNaN(scene.MaxArc) || math.IsInf(scene.MaxArc, 0) {
			return fmt.Errorf("geometry max_arc must be finite and >= 0, got %v", scene.MaxArc)
		}
		if scene.MaxArc == 0 && sceneGeometry == geometry.Spherical() {
			scene.MaxArc = 2 * math.Pi
		}
	}
	if dimension != sceneGeometry.Dimension() {
		return fmt.Errorf(
			"geometry %q requires scene dimension %d, got %d",
			sceneGeometry.Name(),
			sceneGeometry.Dimension(),
			dimension,
		)
	}
	scene.Space = geometry.NewSceneSpace(sceneGeometry, dimension)

	materials, err := ParseMaterials(script)
	if err != nil {
		return err
	}
	mediaRegistry, err := ParseMediaRegistry(script)
	if err != nil {
		return err
	}
	scene.ObjectTree.Media = mediaRegistry

	var parseErrors []error

	for idx, item := range script.Objects {
		objectLabel := fmt.Sprintf("object[%d]", idx)
		if item.ID != "" {
			objectLabel = fmt.Sprintf("object[%d] id=%q", idx, item.ID)
		}

		materialID := item.MaterialID
		material, exists := materials[materialID]
		if !exists {
			parseErrors = append(parseErrors, fmt.Errorf("%s: undefined material %q", objectLabel, materialID))
			continue
		}

		shapes, err := ParseObjectSpecInSpace(item, scene.Space)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", objectLabel, err))
			continue
		} else if len(shapes) == 0 {
			parseErrors = append(parseErrors, fmt.Errorf("%s: shape parser produced no geometry", objectLabel))
			continue
		}

		mediumBoundary, err := parseMediumBoundary(item.MediumBoundary, mediaRegistry)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s medium_boundary: %w", objectLabel, err))
			continue
		}

		for _, shape := range shapes {
			scene.ObjectTree.AddObject(&object.Object{
				Shape:          shape,
				Material:       material,
				MediumBoundary: mediumBoundary,
			})
		}
	}

	cameras, err := ParseCameras(script)
	if err != nil {
		parseErrors = append(parseErrors, err)
	} else if err := validateCamerasForGeometry(scene.Space.Geometry, cameras); err != nil {
		parseErrors = append(parseErrors, err)
	}

	if len(parseErrors) > 0 {
		return errors.Join(parseErrors...)
	}
	scene.Cameras = cameras
	scene.ObjectTree.Build()
	return nil
}

func sceneDimension(script *parser.Script) (int, error) {
	dimension := script.Dimension
	if dimension <= 0 {
		dimension = 3
	}
	return dimension, nil
}

func validateCamerasForGeometry(g geometry.Geometry, cameras map[string]modelcamera.RayCamera) error {
	for id, cam := range cameras {
		switch g.Kind() {
		case geometry.EuclideanKind:
			switch cam.(type) {
			case *modelcamera.HyperbolicCamera, *modelcamera.SphericalCamera:
				return fmt.Errorf("camera %q is non-euclidean but scene geometry is euclidean", id)
			}
		case geometry.KleinKind:
			if _, ok := cam.(*modelcamera.HyperbolicCamera); !ok {
				return fmt.Errorf("camera %q must use type %q for Klein geometry, got %T", id, modelcamera.CameraTypeHyperbolic, cam)
			}
		case geometry.SphericalKind:
			if _, ok := cam.(*modelcamera.SphericalCamera); !ok {
				return fmt.Errorf("camera %q must use type %q for spherical geometry, got %T", id, modelcamera.CameraTypeSpherical, cam)
			}
		}
	}
	return nil
}
