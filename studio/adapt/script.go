package adapt

import (
	"errors"
	"fmt"
	"strings"
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
	baseCameras, err := adaptCameras(script.Cameras, dimension)
	if err != nil {
		return nil, err
	}
	renders, err := rendersToMaps(script, baseCameras)
	if err != nil {
		return nil, err
	}
	geometry, err := adaptGeometry(script.Geometry)
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
		Dimension: dimension,
		Materials: adaptMaterials(script.Materials),
		Media:     cloneNestedStringMap(script.Media),
		Objects:   objects,
		Cameras:   baseCameras,
		Geometry:  geometry,
		Renders:   renders,
	}, nil
}

func renderToMap(script *schema.StudioScript, render schema.StudioRenderScript, cameraIDs map[string]bool) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	film, err := selectedFilm(script, render.FilmID)
	if err != nil {
		return nil, err
	}
	if render.Integrator != "" {
		integrator := render.Integrator
		if integrator == "light_trace" {
			integrator = "light_tracing"
		}
		result["integrator"] = integrator
	}
	if render.Samples > 0 {
		result["samples"] = render.Samples
	}
	if render.ThreadNum > 0 {
		result["thread_num"] = render.ThreadNum
	}
	if film.CameraID == "" {
		return nil, fmt.Errorf("film %q must specify camera_id", film.ID)
	}
	if !cameraIDs[film.CameraID] {
		return nil, fmt.Errorf("film camera_id %q does not exist", film.CameraID)
	}
	if len(film.Shape) == 0 {
		return nil, fmt.Errorf("film %q shape is required", film.ID)
	}
	for axis, extent := range film.Shape {
		if extent <= 0 {
			return nil, fmt.Errorf("film %q shape[%d] must be > 0", film.ID, axis)
		}
	}
	result["camera_id"] = film.CameraID
	result["film"] = schema.EngineFilmScript{
		Shape:            append([]int(nil), film.Shape...),
		SpectralBinCount: film.SpectralBinCount,
		PixelWindows:     clonePixelWindows(film.PixelWindows),
	}
	result["output"] = film.OutputFilm
	if render.WavelengthSamples > 0 {
		result["wavelength_samples"] = render.WavelengthSamples
	}
	return result, nil
}

func adaptGeometry(source map[string]interface{}) (map[string]interface{}, error) {
	if len(source) == 0 {
		return nil, nil
	}
	result := cloneMap(source)
	kind, _ := stringField(result, "type")
	switch strings.ToLower(kind) {
	case "", "euclidean":
		result["type"] = "euclidean"
	case "klein", "hyperbolic":
		result["type"] = "klein"
	case "spherical", "sphere":
		result["type"] = "spherical"
	default:
		return nil, fmt.Errorf("unsupported geometry type %q", kind)
	}
	return result, nil
}

func adaptMaterials(source []map[string]interface{}) []map[string]interface{} {
	result := cloneMapSlice(source)
	for _, material := range result {
		if surface, ok := material["surface"].(map[string]interface{}); ok {
			canonicalizeSurface(surface)
		}
	}
	return result
}

func canonicalizeSurface(surface map[string]interface{}) {
	kind, _ := stringField(surface, "type")
	if strings.EqualFold(kind, "wire_mesh") {
		surface["type"] = "cylindrical_grid_cutout"
	}
	if line, ok := surface["line_surface"].(map[string]interface{}); ok {
		canonicalizeSurface(line)
	}
	if components, ok := surface["components"].([]interface{}); ok {
		for _, raw := range components {
			component, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			if nested, ok := component["surface"].(map[string]interface{}); ok {
				canonicalizeSurface(nested)
			}
		}
	}
}

func rendersToMaps(script *schema.StudioScript, cameras []schema.EngineCameraScript) ([]map[string]interface{}, error) {
	cameraIDs := make(map[string]bool, len(cameras))
	for _, camera := range cameras {
		cameraIDs[camera.ID] = true
	}
	renders := resolvedRenderScripts(script)
	result := make([]map[string]interface{}, len(renders))
	for i, render := range renders {
		mapped, err := renderToMap(script, render, cameraIDs)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
	}
	return result, nil
}

// resolvedRenderScripts upgrades Studio's legacy render defaults into complete
// Engine render jobs. The Engine schema only exposes renders.
func resolvedRenderScripts(script *schema.StudioScript) []schema.StudioRenderScript {
	if len(script.Renders) == 0 {
		return []schema.StudioRenderScript{script.Render}
	}
	renders := make([]schema.StudioRenderScript, len(script.Renders))
	for i, render := range script.Renders {
		renders[i] = schema.MergeRenderScripts(script.Render, render)
	}
	return renders
}

func selectedFilm(script *schema.StudioScript, id string) (schema.StudioFilmScript, error) {
	if id == "" {
		return schema.StudioFilmScript{}, errors.New("render must specify film_id")
	}
	for _, film := range script.Films {
		if film.ID == id {
			return film, nil
		}
	}
	return schema.StudioFilmScript{}, fmt.Errorf("film %q does not exist", id)
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
