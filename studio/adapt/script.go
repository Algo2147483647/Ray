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

	render, err := renderToMap(script, script.Render, cameraIDs)
	if err != nil {
		return nil, err
	}
	renders, err := rendersToMaps(script, cameraIDs)
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
		Render:    render,
		Geometry:  cloneMap(script.Geometry),
		Renders:   renders,
	}, nil
}

func renderToMap(script *schema.StudioScript, render schema.StudioRenderScript, cameraIDs map[string]string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	filmID := render.FilmID
	if filmID == "" {
		filmID = script.Render.FilmID
	}
	film, err := selectedFilm(script, filmID)
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
	if render.WavelengthSamples > 0 {
		result["wavelength_samples"] = render.WavelengthSamples
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
	add(script.Render.FilmID)
	if len(script.Renders) > 0 {
		for _, render := range script.Renders {
			id := render.FilmID
			if id == "" {
				id = script.Render.FilmID
			}
			add(id)
		}
	}
	return result
}

func rendersToMaps(script *schema.StudioScript, cameraIDs map[string]string) ([]map[string]interface{}, error) {
	if len(script.Renders) == 0 {
		return nil, nil
	}
	result := make([]map[string]interface{}, len(script.Renders))
	for i, render := range script.Renders {
		mapped, err := renderToMap(script, render, cameraIDs)
		if err != nil {
			return nil, err
		}
		result[i] = mapped
	}
	return result, nil
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
