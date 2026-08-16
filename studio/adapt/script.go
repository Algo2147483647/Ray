package adapt

import (
	"errors"
	"fmt"
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
	cameras, cameraIDs, err := attachFilms(baseCameras, script)
	if err != nil {
		return nil, err
	}

	renders, err := rendersToMaps(script, cameraIDs, dimension)
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
		Geometry:  cloneMap(script.Geometry),
		Renders:   renders,
	}, nil
}

func renderToMap(script *schema.StudioScript, render schema.StudioRenderScript, cameraIDs map[string]string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	film, err := selectedFilm(script, render.FilmID)
	if err != nil {
		return nil, err
	}
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
	cameraID, exists := cameraIDs[film.ID]
	if !exists {
		return nil, fmt.Errorf("film %q is not attached to an Engine camera", film.ID)
	}
	result["camera_id"] = cameraID
	if render.SpectrumMode != "" {
		result["spectrum_mode"] = render.SpectrumMode
	}
	wavelengthSamples := schema.NormalizeWavelengthSamples(render.SpectrumMode, render.WavelengthSamples)
	if wavelengthSamples > 0 {
		result["wavelength_samples"] = wavelengthSamples
	}
	return result, nil
}

func attachFilms(baseCameras []schema.EngineCameraScript, script *schema.StudioScript) ([]schema.EngineCameraScript, map[string]string, error) {
	filmIDs := activeFilmIDs(script)
	films := make(map[string]schema.StudioFilmScript, len(script.Films))
	for _, film := range script.Films {
		films[film.ID] = film
	}
	counts := make(map[string]int)
	for _, filmID := range filmIDs {
		film, exists := films[filmID]
		if !exists {
			return nil, nil, fmt.Errorf("film %q does not exist", filmID)
		}
		counts[film.CameraID]++
	}

	cameras := make([]schema.EngineCameraScript, 0, len(filmIDs))
	cameraIDs := make(map[string]string, len(filmIDs))
	for _, filmID := range filmIDs {
		film := films[filmID]
		if film.CameraID == "" {
			return nil, nil, fmt.Errorf("film %q must specify camera_id", film.ID)
		}
		index := -1
		for i := range baseCameras {
			if baseCameras[i].ID == film.CameraID {
				index = i
				break
			}
		}
		if index < 0 {
			return nil, nil, fmt.Errorf("film camera_id %q does not exist", film.CameraID)
		}
		if len(film.Shape) == 0 {
			return nil, nil, fmt.Errorf("film %q shape is required", film.ID)
		}
		for axis, extent := range film.Shape {
			if extent <= 0 {
				return nil, nil, fmt.Errorf("film %q shape[%d] must be > 0", film.ID, axis)
			}
		}
		camera := baseCameras[index]
		if counts[film.CameraID] > 1 {
			camera.ID = film.CameraID + "@" + film.ID
		}
		camera.Film = schema.EngineFilmScript{
			Shape:        append([]int(nil), film.Shape...),
			OutputFilm:   film.OutputFilm,
			PixelWindows: clonePixelWindows(film.PixelWindows),
		}
		cameras = append(cameras, camera)
		cameraIDs[film.ID] = camera.ID
	}
	return cameras, cameraIDs, nil
}

func activeFilmIDs(script *schema.StudioScript) []string {
	seen := map[string]bool{}
	result := []string{}
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		result = append(result, id)
	}
	for _, render := range resolvedRenderScripts(script) {
		add(render.FilmID)
	}
	return result
}

func rendersToMaps(script *schema.StudioScript, cameraIDs map[string]string, dimension int) ([]map[string]interface{}, error) {
	renders := resolvedRenderScripts(script)
	result := make([]map[string]interface{}, len(renders))
	for i, render := range renders {
		render.Dimension = dimension
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
