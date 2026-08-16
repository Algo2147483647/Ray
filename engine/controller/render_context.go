package controller

import (
	"fmt"
	"runtime"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

const (
	defaultScriptPath = "../../examples/scenes/default.json"
	defaultSamples    = int64(20)
	defaultOutputFilm = "../../outputs/img.bin"
)

type RenderContext struct {
	Integrator        string
	Dimension         int
	CameraID          string
	ThreadNum         int
	Samples           int64
	OutputFilm        string
	SpectrumMode      string
	WavelengthSamples int
}

func defaultRenderContext() RenderContext {
	return RenderContext{
		Integrator:        "path",
		Dimension:         3,
		ThreadNum:         runtime.NumCPU(),
		Samples:           defaultSamples,
		SpectrumMode:      "hero_wavelength",
		WavelengthSamples: 1,
	}
}

func renderScriptContext(render parser.RenderScript) RenderContext {
	return RenderContext{
		Integrator:        render.Integrator,
		Dimension:         render.Dimension,
		CameraID:          render.CameraID,
		ThreadNum:         render.ThreadNum,
		Samples:           render.Samples,
		SpectrumMode:      render.SpectrumMode,
		WavelengthSamples: render.WavelengthSamples,
	}
}

// mergeRenderContext applies non-zero override values to base. A zero value
// means "not specified" for render configuration fields.
func mergeRenderContext(base, override RenderContext) RenderContext {
	if override.Integrator != "" {
		base.Integrator = override.Integrator
	}
	if override.CameraID != "" {
		base.CameraID = override.CameraID
	}
	if override.Dimension > 0 {
		base.Dimension = override.Dimension
	}
	if override.ThreadNum > 0 {
		base.ThreadNum = override.ThreadNum
	}
	if override.Samples > 0 {
		base.Samples = override.Samples
	}
	if override.SpectrumMode != "" {
		base.SpectrumMode = override.SpectrumMode
	}
	if override.WavelengthSamples > 0 {
		base.WavelengthSamples = override.WavelengthSamples
	}
	return base
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
		context.OutputFilm = defaultOutputFilm
	}
	h.Context = context
	return h
}

func renderSpectrumMode(value string) optics.SpectrumMode {
	switch value {
	case "sampled":
		return optics.SpectrumModeSampledWavelengths
	default:
		return optics.SpectrumModeHeroWavelength
	}
}
