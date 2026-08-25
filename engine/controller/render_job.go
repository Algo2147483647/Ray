package controller

import (
	"fmt"
	"runtime"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/film"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/ray_tracing"
)

const (
	defaultSamples               = int64(20)
	defaultWavelengthSampleCount = 1
)

// ResolveRenderJob is the only render-default and scene-binding boundary.
// The returned job is complete and can be passed unchanged to ray tracing.
func ResolveRenderJob(render parser.RenderScript, scene *model.Scene) (ray_tracing.RenderJob, error) {
	if scene == nil {
		return ray_tracing.RenderJob{}, fmt.Errorf("render scene is nil")
	}
	if render.Film == nil {
		return ray_tracing.RenderJob{}, fmt.Errorf("render film is required")
	}

	integratorName := render.Integrator
	if integratorName == "" {
		integratorName = string(ray_tracing.IntegratorPathTracing)
	}
	integrator, err := ray_tracing.ParseIntegratorKind(integratorName)
	if err != nil {
		return ray_tracing.RenderJob{}, err
	}
	samples := defaultSamples
	if render.Samples != nil {
		samples = *render.Samples
	}
	threadNum := runtime.NumCPU()
	if render.ThreadNum != nil {
		threadNum = *render.ThreadNum
	}
	wavelengthSamples := defaultWavelengthSampleCount
	if render.WavelengthSamples != nil {
		wavelengthSamples = *render.WavelengthSamples
	}

	renderFilm := &film.Film{Shape: append([]int(nil), render.Film.Shape...)}
	for _, window := range render.Film.PixelWindows {
		renderFilm.PixelWindows = append(renderFilm.PixelWindows, film.PixelWindow{
			Min: append([]int(nil), window.Min...), Max: append([]int(nil), window.Max...),
		})
	}
	if renderFilm.ElementCount() == 0 {
		return ray_tracing.RenderJob{}, fmt.Errorf("render film shape must contain positive extents")
	}
	windows, err := film.NormalizePixelWindows(renderFilm.PixelWindows, renderFilm.Shape)
	if err != nil {
		return ray_tracing.RenderJob{}, err
	}
	renderFilm.PixelWindows = windows
	binCount := render.Film.SpectralBinCount
	if binCount == 0 {
		binCount = film.DefaultSpectralBinCount
	}
	if binCount < 1 || binCount > film.MaxSpectralBinCount {
		return ray_tracing.RenderJob{}, fmt.Errorf("film spectral_bin_count must be between 1 and %d", film.MaxSpectralBinCount)
	}
	renderFilm.InitSpectralBins(binCount, optics.WavelengthMin, optics.WavelengthMax)

	if render.CameraID == "" {
		return ray_tracing.RenderJob{}, fmt.Errorf("render camera_id %q does not exist", render.CameraID)
	}
	renderCamera, exists := scene.Cameras[render.CameraID]
	if !exists {
		return ray_tracing.RenderJob{}, fmt.Errorf("camera %q does not exist", render.CameraID)
	}
	if renderCamera.RasterDimension() != len(renderFilm.Shape) {
		return ray_tracing.RenderJob{}, fmt.Errorf(
			"camera %q expects a %dD film, got %dD",
			render.CameraID, renderCamera.RasterDimension(), len(renderFilm.Shape),
		)
	}

	return ray_tracing.NewRenderJob(
		integrator, renderCamera, renderFilm, render.Output, scene.ObjectTree,
		samples, threadNum, wavelengthSamples,
	)
}
