package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type studioScript struct {
	Includes  []string                          `json:"includes"`
	Materials []map[string]interface{}          `json:"materials"`
	Media     map[string]map[string]interface{} `json:"media"`
	Objects   []map[string]interface{}          `json:"objects"`
	Cameras   []studioCameraScript              `json:"cameras"`
	Render    studioRenderScript                `json:"render"`
	Geometry  map[string]interface{}            `json:"geometry"`
	Renders   []studioRenderScript              `json:"renders"`
}

type studioRenderScript struct {
	Dimension         int     `json:"dimension"`
	Samples           int64   `json:"samples"`
	ThreadNum         int     `json:"thread_num"`
	CameraIndex       int     `json:"camera_index"`
	CameraIndexSet    bool    `json:"-"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	OutputImage       string  `json:"output_image"`
	OutputFilm        string  `json:"output_film"`
	ResumeFilm        string  `json:"resume_film"`
	Exposure          float64 `json:"exposure"`
	ToneMapping       string  `json:"tone_mapping"`
	Gamma             float64 `json:"gamma"`
	SpectrumMode      string  `json:"spectrum_mode"`
	WavelengthSamples int     `json:"wavelength_samples"`
	ColorSpace        string  `json:"color_space"`
	FilmColorSpace    string  `json:"working_space"`
}

func (r *studioRenderScript) UnmarshalJSON(data []byte) error {
	type renderScript studioRenderScript
	var decoded renderScript
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decoded.CameraIndexSet = raw["camera_index"] != nil
	*r = studioRenderScript(decoded)
	return nil
}

func readStudioScriptFiles(paths []string) (*studioScript, error) {
	if len(paths) == 0 {
		return nil, errors.New("no script files provided")
	}

	merged := &studioScript{}
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

func readStudioScriptFile(path string) (*studioScript, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve script path %q: %w", path, err)
	}
	return readStudioScriptFileRecursive(filepath.Clean(absolute), map[string]bool{})
}

func readStudioScriptFileRecursive(path string, stack map[string]bool) (*studioScript, error) {
	if stack[path] {
		return nil, fmt.Errorf("include cycle detected at %q", path)
	}
	stack[path] = true
	defer delete(stack, path)

	script, err := readStudioScriptFileRaw(path)
	if err != nil {
		return nil, err
	}

	merged := &studioScript{}
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

func readStudioScriptFileRaw(path string) (*studioScript, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open script %q: %w", path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read script %q: %w", path, err)
	}

	var script studioScript
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, fmt.Errorf("parse script %q: %w", path, err)
	}
	return &script, nil
}

func mergeStudioScripts(dst, src *studioScript, source string) error {
	if dst == nil || src == nil {
		return errors.New("cannot merge nil script")
	}
	if err := mergeStudioMedia(dst, src, source); err != nil {
		return err
	}
	if err := appendUniqueStudioIDMaps(&dst.Materials, src.Materials, "material", source); err != nil {
		return err
	}
	if err := appendUniqueStudioIDMaps(&dst.Objects, src.Objects, "object", source); err != nil {
		return err
	}
	if err := appendUniqueStudioCameras(&dst.Cameras, src.Cameras, source); err != nil {
		return err
	}
	dst.Render = mergeStudioRenderScript(dst.Render, src.Render)
	if len(src.Geometry) > 0 {
		dst.Geometry = cloneMap(src.Geometry)
	}
	dst.Renders = append(dst.Renders, src.Renders...)
	return nil
}

func mergeStudioMedia(dst, src *studioScript, source string) error {
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
		dst.Media[id] = cloneMap(medium)
	}
	return nil
}

func appendUniqueStudioIDMaps(dst *[]map[string]interface{}, src []map[string]interface{}, label, source string) error {
	ids := map[string]bool{}
	for _, item := range *dst {
		if id, ok := stringField(item, "id"); ok {
			ids[id] = true
		}
	}
	for _, item := range src {
		id, ok := stringField(item, "id")
		if !ok {
			*dst = append(*dst, cloneMap(item))
			continue
		}
		if ids[id] {
			return fmt.Errorf("duplicate %s id %q while merging %s", label, id, source)
		}
		ids[id] = true
		*dst = append(*dst, cloneMap(item))
	}
	return nil
}

func appendUniqueStudioCameras(dst *[]studioCameraScript, src []studioCameraScript, source string) error {
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
		*dst = append(*dst, cloneStudioCamera(camera))
	}
	return nil
}

func mergeStudioRenderScript(base, override studioRenderScript) studioRenderScript {
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
