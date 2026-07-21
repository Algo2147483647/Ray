package adapt

import (
	"errors"
	"time"
)

func AdaptScript(script *studioScript, source []string, dimension int) (*intermediateScript, error) {
	if script == nil {
		return nil, errors.New("script is nil")
	}

	objects, err := flattenObjects(script.Objects, newRootContext(dimension), dimension)
	if err != nil {
		return nil, err
	}
	cameras, err := adaptCameras(script.Cameras, dimension)
	if err != nil {
		return nil, err
	}

	return &intermediateScript{
		Studio: studioMetadata{
			Version:     "0.1",
			Source:      append([]string(nil), source...),
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Dimension:   dimension,
		},
		Materials: cloneMapSlice(script.Materials),
		Media:     cloneNestedStringMap(script.Media),
		Objects:   objects,
		Cameras:   cameras,
		Render:    renderToMap(script.Render),
		Geometry:  cloneMap(script.Geometry),
		Renders:   rendersToMaps(script.Renders),
	}, nil
}

func renderToMap(render studioRenderScript) map[string]interface{} {
	result := map[string]interface{}{}
	if render.Dimension > 0 {
		result["dimension"] = render.Dimension
	}
	if render.Samples > 0 {
		result["samples"] = render.Samples
	}
	if render.ThreadNum > 0 {
		result["thread_num"] = render.ThreadNum
	}
	if render.CameraIndexSet {
		result["camera_index"] = render.CameraIndex
	}
	if render.Width > 0 {
		result["width"] = render.Width
	}
	if render.Height > 0 {
		result["height"] = render.Height
	}
	if render.OutputFilm != "" {
		result["output_film"] = render.OutputFilm
	}
	if render.Exposure > 0 {
		result["exposure"] = render.Exposure
	}
	if render.ToneMapping != "" {
		result["tone_mapping"] = render.ToneMapping
	}
	if render.Gamma > 0 {
		result["gamma"] = render.Gamma
	}
	if render.SpectrumMode != "" {
		result["spectrum_mode"] = render.SpectrumMode
	}
	if render.WavelengthSamples > 0 {
		result["wavelength_samples"] = render.WavelengthSamples
	}
	if render.ColorSpace != "" {
		result["color_space"] = render.ColorSpace
	}
	if render.FilmColorSpace != "" {
		result["working_space"] = render.FilmColorSpace
	}
	return result
}

func rendersToMaps(renders []studioRenderScript) []map[string]interface{} {
	if len(renders) == 0 {
		return nil
	}
	result := make([]map[string]interface{}, len(renders))
	for i, render := range renders {
		result[i] = renderToMap(render)
	}
	return result
}
