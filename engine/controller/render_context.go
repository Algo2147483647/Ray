package controller

import (
	"fmt"
	"runtime"

	"github.com/Algo2147483647/ray/engine/controller/parser"
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
	OutputFilm        string
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
	}
	if resolved.Integrator == "" {
		resolved.Integrator = "path"
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

	var exists bool
	h.Camera, exists = h.Scene.Cameras[context.CameraID]
	if !exists {
		h.err = fmt.Errorf("camera %q does not exist", context.CameraID)
		return h
	}

	film := h.Camera.GetFilm()
	context.OutputFilm = film.OutputFilm
	if context.OutputFilm == "" {
		h.err = fmt.Errorf("camera %q Film requires output_film", context.CameraID)
		return h
	}
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
