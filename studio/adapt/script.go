package adapt

import (
	"errors"
	"time"

	"github.com/Algo2147483647/ray/studio/schema"
)

func AdaptScript(script *schema.StudioScript, source []string, dimension int) (*schema.IntermediateScript, error) {
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

	return &schema.IntermediateScript{
		Studio: schema.StudioMetadata{
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

func renderToMap(render schema.StudioRenderScript) map[string]interface{} {
	result := map[string]interface{}{}
	if render.Integrator != "" {
		result["integrator"] = render.Integrator
	}
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
	if len(render.PixelWindows) > 0 {
		result["pixel_windows"] = clonePixelWindows(render.PixelWindows)
	}
	return result
}

func rendersToMaps(renders []schema.StudioRenderScript) []map[string]interface{} {
	if len(renders) == 0 {
		return nil
	}
	result := make([]map[string]interface{}, len(renders))
	for i, render := range renders {
		result[i] = renderToMap(render)
	}
	return result
}

func clonePixelWindows(windows []schema.PixelWindowScript) []schema.PixelWindowScript {
	if len(windows) == 0 {
		return nil
	}
	cloned := make([]schema.PixelWindowScript, len(windows))
	for i, window := range windows {
		cloned[i] = schema.PixelWindowScript{
			Min: append([]int(nil), window.Min...),
			Max: append([]int(nil), window.Max...),
		}
	}
	return cloned
}
