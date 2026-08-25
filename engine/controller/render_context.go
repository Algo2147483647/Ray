package controller

import (
	"fmt"
	"runtime"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/ray_tracing"
)

const (
	defaultSamples                = int64(20)
	defaultSampledWavelengthCount = 4
)

type RenderContext struct {
	Integrator        string
	CameraID          string
	ThreadNum         int
	Samples           int64
	Output            string
	Film              *camera.Film
	SpectrumMode      string
	WavelengthSamples int
}

// ResolveRenderSpec is the single business-default boundary for an Engine
// render job. Runtime objects receive a complete context and never infer
// missing render semantics.
func ResolveRenderSpec(render parser.RenderScript) (RenderContext, error) {
	resolved := RenderContext{
		Integrator:        render.Integrator,
		CameraID:          render.CameraID,
		ThreadNum:         render.ThreadNum,
		Samples:           render.Samples,
		SpectrumMode:      render.SpectrumMode,
		WavelengthSamples: render.WavelengthSamples,
		Output:            render.Output,
	}
	if resolved.Integrator == "" {
		resolved.Integrator = "path"
	}
	if render.Film == nil {
		return RenderContext{}, fmt.Errorf("render film is required")
	}
	resolved.Film = &camera.Film{
		Shape:            append([]int(nil), render.Film.Shape...),
		SpectralBinCount: render.Film.SpectralBinCount,
	}
	for _, window := range render.Film.PixelWindows {
		resolved.Film.PixelWindows = append(resolved.Film.PixelWindows, camera.PixelWindow{
			Min: append([]int(nil), window.Min...), Max: append([]int(nil), window.Max...),
		})
	}
	if resolved.Film.ElementCount() == 0 {
		return RenderContext{}, fmt.Errorf("render film shape must contain positive extents")
	}
	windows, err := camera.NormalizePixelWindows(resolved.Film.PixelWindows, resolved.Film.Shape)
	if err != nil {
		return RenderContext{}, err
	}
	resolved.Film.PixelWindows = windows
	binCount := resolved.Film.SpectralBinCount
	if binCount == 0 {
		binCount = camera.DefaultSpectralBinCount
	}
	if binCount < 0 || binCount > camera.MaxSpectralBinCount {
		return RenderContext{}, fmt.Errorf("film spectral_bin_count must be between 1 and %d", camera.MaxSpectralBinCount)
	}
	if !resolved.Film.HasSpectralBins() {
		resolved.Film.InitSpectralBins(binCount, optics.WavelengthMin, optics.WavelengthMax)
	} else if len(resolved.Film.SpectralBins) != binCount {
		return RenderContext{}, fmt.Errorf("film spectral_bin_count %d does not match %d supplied spectral bins", binCount, len(resolved.Film.SpectralBins))
	}
	if resolved.Output == "" {
		return RenderContext{}, fmt.Errorf("render output is required")
	}
	if _, err := ray_tracing.ParseIntegratorKind(resolved.Integrator); err != nil {
		return RenderContext{}, err
	}
	if resolved.ThreadNum <= 0 {
		resolved.ThreadNum = runtime.NumCPU()
	}
	if resolved.Samples <= 0 {
		resolved.Samples = defaultSamples
	}
	if resolved.SpectrumMode == "" {
		resolved.SpectrumMode = "hero_wavelength"
	}
	switch resolved.SpectrumMode {
	case "hero_wavelength":
		resolved.WavelengthSamples = 1
	case "sampled":
		if resolved.WavelengthSamples <= 0 {
			resolved.WavelengthSamples = defaultSampledWavelengthCount
		}
	default:
		return RenderContext{}, fmt.Errorf("unsupported spectrum_mode %q", resolved.SpectrumMode)
	}
	return resolved, nil
}

func (h *Handler) ConfigureRenderContext(context RenderContext) *Handler {
	if h.err != nil {
		return h
	}
	if context.CameraID == "" {
		h.err = fmt.Errorf("render camera_id %q does not exist", context.CameraID)
		return h
	}
	if context.Film == nil {
		h.err = fmt.Errorf("render film is not resolved")
		return h
	}

	var exists bool
	renderCamera, exists := h.Scene.Cameras[context.CameraID]
	if !exists {
		h.err = fmt.Errorf("camera %q does not exist", context.CameraID)
		return h
	}

	if renderCamera.RasterDimension() != len(context.Film.Shape) {
		h.err = fmt.Errorf("camera %q expects a %dD film, got %dD", context.CameraID, renderCamera.RasterDimension(), len(context.Film.Shape))
		return h
	}
	h.Target = model.RenderTarget{Camera: renderCamera, Film: context.Film, Output: context.Output}
	h.Context = context
	return h
}

func renderSpectrumMode(value string) optics.SpectrumMode {
	switch value {
	case "sampled":
		return optics.SpectrumModeSampledWavelengths
	case "hero_wavelength":
		return optics.SpectrumModeHeroWavelength
	default:
		panic("unresolved spectrum mode")
	}
}
