package controller

import (
	"flag"
	"fmt"
	"io"
	"runtime"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model/camera"
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
	PixelWindows      []camera.PixelWindow
}

func (h *Handler) ParseArgs(args []string) *Handler {
	if h.err != nil {
		return h
	}

	scriptPaths := stringListFlag{}

	flagSet := flag.NewFlagSet("ray", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.Var(&scriptPaths, "script", "path to a canonical scene script")

	if err := flagSet.Parse(args); err != nil {
		h.err = err
		return h
	}

	scriptPaths = append(scriptPaths, flagSet.Args()...)
	if len(scriptPaths) == 0 {
		scriptPaths = append(scriptPaths, defaultScriptPath)
	}
	if len(scriptPaths) != 1 {
		h.err = fmt.Errorf("engine accepts exactly one --script; use studio to merge multiple scripts")
		return h
	}
	h.ScriptPath = scriptPaths[0]
	return h
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

type stringListFlag []string

func (s *stringListFlag) String() string {
	return fmt.Sprint([]string(*s))
}

func (s *stringListFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}
