package factory

import (
	"errors"
	"fmt"

	"github.com/Algo2147483647/ray/engine/go/internal/model"
	"github.com/Algo2147483647/ray/engine/go/internal/model/camera"
	"github.com/Algo2147483647/ray/engine/go/internal/model/object"
	"github.com/Algo2147483647/ray/engine/go/internal/sceneio/schema"
)

func LoadSceneFromScript(script *schema.Script, scene *model.Scene) error {
	if script == nil {
		return errors.New("script is nil")
	}
	if scene == nil {
		return errors.New("scene is nil")
	}

	scene.ObjectTree = &object.ObjectTree{}
	scene.Cameras = nil

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
		if objectID, ok, err := optionalStringField(item, "id"); err == nil && ok && objectID != "" {
			objectLabel = fmt.Sprintf("object[%d] id=%q", idx, objectID)
		} else if err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", objectLabel, err))
			continue
		}

		materialID, err := requiredStringField(item, "material_id")
		if err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", objectLabel, err))
			continue
		}
		material, exists := materials[materialID]
		if !exists {
			parseErrors = append(parseErrors, fmt.Errorf("%s: undefined material %q", objectLabel, materialID))
			continue
		}

		shapes, err := ParseShape(item)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", objectLabel, err))
			continue
		}
		if len(shapes) == 0 {
			parseErrors = append(parseErrors, fmt.Errorf("%s: shape parser produced no geometry", objectLabel))
			continue
		}

		mediumBoundary, err := parseMediumBoundary(item, mediaRegistry)
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
	}

	if len(parseErrors) > 0 {
		return errors.Join(parseErrors...)
	}
	scene.Cameras = append(scene.Cameras, cameras...)
	scene.ObjectTree.Build()
	return nil
}

func ParseCameras(script *schema.Script) ([]camera.Camera, error) {
	cameraDefs := script.Cameras
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
