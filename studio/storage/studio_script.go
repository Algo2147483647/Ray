package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Algo2147483647/ray/studio/adapt"
	"github.com/Algo2147483647/ray/studio/schema"
)

func ReadStudioScriptFiles(paths []string) (*schema.StudioScript, error) {
	if len(paths) == 0 {
		return nil, errors.New("no script files provided")
	}

	merged := &schema.StudioScript{}
	for _, path := range paths {
		script, err := readStudioScriptFile(path)
		if err != nil {
			return nil, err
		}
		if err := mergeStudioScripts(merged, script, path); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func readStudioScriptFile(path string) (*schema.StudioScript, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve script path %q: %w", path, err)
	}
	return readStudioScriptFileRecursive(filepath.Clean(absolute), map[string]bool{})
}

func readStudioScriptFileRecursive(path string, stack map[string]bool) (*schema.StudioScript, error) {
	if stack[path] {
		return nil, fmt.Errorf("include cycle detected at %q", path)
	}
	stack[path] = true
	defer delete(stack, path)

	script, err := readStudioScriptFileRaw(path)
	if err != nil {
		return nil, err
	}

	merged := &schema.StudioScript{}
	for _, include := range script.Includes {
		includePath := include
		if !filepath.IsAbs(includePath) {
			includePath = filepath.Join(filepath.Dir(path), includePath)
		}
		includePath, err = filepath.Abs(includePath)
		if err != nil {
			return nil, fmt.Errorf("resolve script path %q: %w", include, err)
		}
		included, err := readStudioScriptFileRecursive(filepath.Clean(includePath), stack)
		if err != nil {
			return nil, err
		}
		if err := mergeStudioScripts(merged, included, includePath); err != nil {
			return nil, err
		}
	}

	script.Includes = nil
	if err := mergeStudioScripts(merged, script, path); err != nil {
		return nil, err
	}
	return merged, nil
}

func readStudioScriptFileRaw(path string) (*schema.StudioScript, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open script %q: %w", path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read script %q: %w", path, err)
	}

	var script schema.StudioScript
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, fmt.Errorf("parse script %q: %w", path, err)
	}
	return &script, nil
}

func mergeStudioScripts(dst, src *schema.StudioScript, source string) error {
	if dst == nil || src == nil {
		return errors.New("cannot merge nil script")
	}
	if err := mergeStudioMedia(dst, src, source); err != nil {
		return err
	}
	if err := appendUniqueStudioIDMaps(&dst.Materials, src.Materials, "material", source); err != nil {
		return err
	}
	if err := appendOrMergeStudioObjects(&dst.Objects, src.Objects, source); err != nil {
		return err
	}
	if err := appendUniqueStudioCameras(&dst.Cameras, src.Cameras, source); err != nil {
		return err
	}
	dst.Render = mergeStudioRenderScript(dst.Render, src.Render)
	if len(src.Geometry) > 0 {
		dst.Geometry = adapt.CloneMap(src.Geometry)
	}
	dst.Renders = append(dst.Renders, src.Renders...)
	return nil
}

func mergeStudioMedia(dst, src *schema.StudioScript, source string) error {
	if len(src.Media) == 0 {
		return nil
	}
	if dst.Media == nil {
		dst.Media = map[string]map[string]interface{}{}
	}
	for id, medium := range src.Media {
		if _, exists := dst.Media[id]; exists {
			return fmt.Errorf("duplicate medium id %q while merging %s", id, source)
		}
		dst.Media[id] = adapt.CloneMap(medium)
	}
	return nil
}

func appendUniqueStudioIDMaps(dst *[]map[string]interface{}, src []map[string]interface{}, label, source string) error {
	ids := map[string]bool{}
	for _, item := range *dst {
		if id, ok := adapt.StringField(item, "id"); ok {
			ids[id] = true
		}
	}
	for _, item := range src {
		id, ok := adapt.StringField(item, "id")
		if !ok {
			*dst = append(*dst, adapt.CloneMap(item))
			continue
		}
		if ids[id] {
			return fmt.Errorf("duplicate %s id %q while merging %s", label, id, source)
		}
		ids[id] = true
		*dst = append(*dst, adapt.CloneMap(item))
	}
	return nil
}

func appendOrMergeStudioObjects(dst *[]map[string]interface{}, src []map[string]interface{}, source string) error {
	ids := map[string]int{}
	for index, item := range *dst {
		if id, ok := adapt.StringField(item, "id"); ok {
			ids[id] = index
		}
	}
	for _, item := range src {
		id, ok := adapt.StringField(item, "id")
		if !ok {
			*dst = append(*dst, adapt.CloneMap(item))
			continue
		}
		if existingIndex, exists := ids[id]; exists {
			merged, err := mergeStudioObject((*dst)[existingIndex], item, source)
			if err != nil {
				return err
			}
			(*dst)[existingIndex] = merged
			continue
		}
		ids[id] = len(*dst)
		*dst = append(*dst, adapt.CloneMap(item))
	}
	return nil
}

func mergeStudioObject(base, override map[string]interface{}, source string) (map[string]interface{}, error) {
	baseShape, _ := adapt.StringField(base, "shape")
	overrideShape, _ := adapt.StringField(override, "shape")
	baseShape = strings.ToLower(baseShape)
	overrideShape = strings.ToLower(overrideShape)
	if !mergeableContainerShape(baseShape) || !mergeableContainerShape(overrideShape) || baseShape != overrideShape {
		id, _ := adapt.StringField(base, "id")
		return nil, fmt.Errorf("duplicate object id %q while merging %s", id, source)
	}

	merged := adapt.CloneMap(base)
	overrideClone := adapt.CloneMap(override)
	for key, value := range overrideClone {
		if key == "objects" {
			continue
		}
		merged[key] = value
	}

	switch baseShape {
	case "group":
		if objects, ok, err := mergeOptionalStudioObjectLists(merged["objects"], override["objects"], source); err != nil {
			return nil, err
		} else if ok {
			merged["objects"] = objects
		}
	case "array":
		if objects, ok, err := mergeOptionalStudioArrayObjects(merged["objects"], override["objects"], source); err != nil {
			return nil, err
		} else if ok {
			merged["objects"] = objects
		}
	}
	return merged, nil
}

func mergeableContainerShape(shape string) bool {
	return shape == "group" || shape == "array"
}

func mergeStudioObjectLists(baseRaw, overrideRaw interface{}, source string) ([]interface{}, error) {
	baseItems, err := objectListRaw(baseRaw, "objects")
	if err != nil {
		return nil, err
	}
	overrideItems, err := objectListRaw(overrideRaw, "objects")
	if err != nil {
		return nil, err
	}

	merged := cloneInterfaceSlice(baseItems)
	ids := map[string]int{}
	for index, item := range merged {
		object, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := adapt.StringField(object, "id"); ok {
			ids[id] = index
		}
	}
	for _, item := range overrideItems {
		object, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field %q: expected object, got %T", "objects", item)
		}
		id, hasID := adapt.StringField(object, "id")
		if !hasID {
			merged = append(merged, cloneInterfaceValue(item))
			continue
		}
		if existingIndex, exists := ids[id]; exists {
			existing, ok := merged[existingIndex].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("duplicate object id %q while merging %s", id, source)
			}
			nested, err := mergeStudioObject(existing, object, source)
			if err != nil {
				return nil, err
			}
			merged[existingIndex] = nested
			continue
		}
		ids[id] = len(merged)
		merged = append(merged, cloneInterfaceValue(item))
	}
	return merged, nil
}

func mergeOptionalStudioObjectLists(baseRaw, overrideRaw interface{}, source string) ([]interface{}, bool, error) {
	if baseRaw == nil && overrideRaw == nil {
		return nil, false, nil
	}
	if baseRaw == nil {
		overrideItems, err := objectListRaw(overrideRaw, "objects")
		if err != nil {
			return nil, false, err
		}
		return cloneInterfaceSlice(overrideItems), true, nil
	}
	if overrideRaw == nil {
		baseItems, err := objectListRaw(baseRaw, "objects")
		if err != nil {
			return nil, false, err
		}
		return cloneInterfaceSlice(baseItems), true, nil
	}
	merged, err := mergeStudioObjectLists(baseRaw, overrideRaw, source)
	return merged, err == nil, err
}

func mergeStudioArrayObjects(baseRaw, overrideRaw interface{}, source string) (map[string]interface{}, error) {
	baseMap, err := objectMapRaw(baseRaw, "objects")
	if err != nil {
		return nil, err
	}
	overrideMap, err := objectMapRaw(overrideRaw, "objects")
	if err != nil {
		return nil, err
	}
	merged := cloneStringInterfaceMap(baseMap)
	for cell, overrideItems := range overrideMap {
		baseItems, exists := merged[cell]
		if !exists {
			merged[cell] = cloneInterfaceValue(overrideItems)
			continue
		}
		items, err := mergeStudioObjectLists(baseItems, overrideItems, source)
		if err != nil {
			return nil, err
		}
		merged[cell] = items
	}
	return merged, nil
}

func mergeOptionalStudioArrayObjects(baseRaw, overrideRaw interface{}, source string) (map[string]interface{}, bool, error) {
	if baseRaw == nil && overrideRaw == nil {
		return nil, false, nil
	}
	if baseRaw == nil {
		overrideMap, err := objectMapRaw(overrideRaw, "objects")
		if err != nil {
			return nil, false, err
		}
		return cloneStringInterfaceMap(overrideMap), true, nil
	}
	if overrideRaw == nil {
		baseMap, err := objectMapRaw(baseRaw, "objects")
		if err != nil {
			return nil, false, err
		}
		return cloneStringInterfaceMap(baseMap), true, nil
	}
	merged, err := mergeStudioArrayObjects(baseRaw, overrideRaw, source)
	return merged, err == nil, err
}

func objectListRaw(raw interface{}, key string) ([]interface{}, error) {
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected array, got %T", key, raw)
	}
	return items, nil
}

func objectMapRaw(raw interface{}, key string) (map[string]interface{}, error) {
	items, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected object, got %T", key, raw)
	}
	return items, nil
}

func cloneStringInterfaceMap(value map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(value))
	for key, item := range value {
		result[key] = cloneInterfaceValue(item)
	}
	return result
}

func cloneInterfaceSlice(value []interface{}) []interface{} {
	result := make([]interface{}, len(value))
	for i, item := range value {
		result[i] = cloneInterfaceValue(item)
	}
	return result
}

func cloneInterfaceValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return adapt.CloneMap(v)
	case []interface{}:
		return cloneInterfaceSlice(v)
	case []map[string]interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = adapt.CloneMap(item)
		}
		return result
	default:
		return value
	}
}

func appendUniqueStudioCameras(dst *[]schema.StudioCameraScript, src []schema.StudioCameraScript, source string) error {
	ids := map[string]bool{}
	for _, camera := range *dst {
		if camera.ID != "" {
			ids[camera.ID] = true
		}
	}
	for _, camera := range src {
		if camera.ID != "" {
			if ids[camera.ID] {
				return fmt.Errorf("duplicate camera id %q while merging %s", camera.ID, source)
			}
			ids[camera.ID] = true
		}
		*dst = append(*dst, adapt.CloneStudioCamera(camera))
	}
	return nil
}

func mergeStudioRenderScript(base, override schema.StudioRenderScript) schema.StudioRenderScript {
	result := base
	if override.Dimension > 0 {
		result.Dimension = override.Dimension
	}
	if override.Samples > 0 {
		result.Samples = override.Samples
	}
	if override.ThreadNum > 0 {
		result.ThreadNum = override.ThreadNum
	}
	if override.CameraIndexSet {
		result.CameraIndex = override.CameraIndex
		result.CameraIndexSet = true
	}
	if override.Width > 0 {
		result.Width = override.Width
	}
	if override.Height > 0 {
		result.Height = override.Height
	}
	if override.OutputImage != "" {
		result.OutputImage = override.OutputImage
	}
	if override.OutputFilm != "" {
		result.OutputFilm = override.OutputFilm
	}
	if override.ResumeFilm != "" {
		result.ResumeFilm = override.ResumeFilm
	}
	if override.Exposure > 0 {
		result.Exposure = override.Exposure
	}
	if override.ToneMapping != "" {
		result.ToneMapping = override.ToneMapping
	}
	if override.Gamma > 0 {
		result.Gamma = override.Gamma
	}
	if override.SpectrumMode != "" {
		result.SpectrumMode = override.SpectrumMode
	}
	if override.WavelengthSamples > 0 {
		result.WavelengthSamples = override.WavelengthSamples
	}
	if override.ColorSpace != "" {
		result.ColorSpace = override.ColorSpace
	}
	if override.FilmColorSpace != "" {
		result.FilmColorSpace = override.FilmColorSpace
	}
	return result
}
