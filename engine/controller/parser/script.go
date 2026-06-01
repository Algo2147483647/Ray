package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ReadScriptFile(path string) (*Script, error) {
	absolute, err := filepathAbs(path)
	if err != nil {
		return nil, err
	}
	return readScriptFileRecursive(absolute, map[string]bool{})
}

func ReadScriptFiles(paths []string) (*Script, error) {
	if len(paths) == 0 {
		return nil, errors.New("no script files provided")
	}

	merged := &Script{}
	for _, path := range paths {
		script, err := ReadScriptFile(path)
		if err != nil {
			return nil, err
		}
		if err := mergeScripts(merged, script, path); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func readScriptFileRecursive(path string, stack map[string]bool) (*Script, error) {
	if stack[path] {
		return nil, fmt.Errorf("include cycle detected at %q", path)
	}
	stack[path] = true
	defer delete(stack, path)

	script, err := readScriptFileRaw(path)
	if err != nil {
		return nil, err
	}

	merged := &Script{}
	for _, include := range script.Includes {
		includePath := include
		if !filepath.IsAbs(includePath) {
			includePath = filepath.Join(filepath.Dir(path), includePath)
		}
		includePath, err = filepathAbs(includePath)
		if err != nil {
			return nil, err
		}

		included, err := readScriptFileRecursive(includePath, stack)
		if err != nil {
			return nil, err
		}
		if err := mergeScripts(merged, included, includePath); err != nil {
			return nil, err
		}
	}

	script.Includes = nil
	if err := mergeScripts(merged, script, path); err != nil {
		return nil, err
	}
	return merged, nil
}

func readScriptFileRaw(path string) (*Script, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open script %q: %w", path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read script %q: %w", path, err)
	}

	var script Script
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, fmt.Errorf("parse script %q: %w", path, err)
	}

	return &script, nil
}

func mergeScripts(dst, src *Script, source string) error {
	if dst == nil || src == nil {
		return errors.New("cannot merge nil script")
	}

	if err := mergeMedia(dst, src, source); err != nil {
		return err
	}
	if err := appendUniqueIDMaps(&dst.Materials, src.Materials, "material", source); err != nil {
		return err
	}
	if err := appendUniqueIDMaps(&dst.Objects, src.Objects, "object", source); err != nil {
		return err
	}
	if err := appendUniqueCameras(&dst.Cameras, src.Cameras, source); err != nil {
		return err
	}

	dst.Render = mergeRenderScript(dst.Render, src.Render)
	if src.Geometry != nil {
		geometry := *src.Geometry
		dst.Geometry = &geometry
	}
	dst.Renders = append(dst.Renders, src.Renders...)
	return nil
}

func mergeMedia(dst, src *Script, source string) error {
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
		dst.Media[id] = medium
	}
	return nil
}

func appendUniqueIDMaps(dst *[]map[string]interface{}, src []map[string]interface{}, label, source string) error {
	ids := map[string]bool{}
	for _, item := range *dst {
		if id, ok := stringMapID(item); ok {
			ids[id] = true
		}
	}

	for _, item := range src {
		id, ok := stringMapID(item)
		if !ok {
			*dst = append(*dst, item)
			continue
		}
		if ids[id] {
			return fmt.Errorf("duplicate %s id %q while merging %s", label, id, source)
		}
		ids[id] = true
		*dst = append(*dst, item)
	}
	return nil
}

func appendUniqueCameras(dst *[]CameraScript, src []CameraScript, source string) error {
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
		*dst = append(*dst, camera)
	}
	return nil
}

func mergeRenderScript(base, override RenderScript) RenderScript {
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

func stringMapID(item map[string]interface{}) (string, bool) {
	raw, ok := item["id"]
	if !ok {
		return "", false
	}
	id, ok := raw.(string)
	return id, ok && id != ""
}

func filepathAbs(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve script path %q: %w", path, err)
	}
	return filepath.Clean(absolute), nil
}
