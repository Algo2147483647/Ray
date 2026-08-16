package controller

import (
	"flag"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/ray_tracing"
)

const (
	defaultScriptPath = "../../examples/scenes/default.json"
	defaultSamples    = int64(20)
	defaultOutputFilm = "../../outputs/img.bin"
)

type RenderContext struct {
	ScriptPath        string
	Integrator        string
	Dimension         int
	CameraID          string
	ThreadNum         int
	FilmShapeOverride []int
	Samples           int64
	OutputFilm        string
	SpectrumMode      string
	WavelengthSamples int
	PixelWindows      []camera.PixelWindow
}

func (h *Handler) ParseRenderArgs(args []string) *Handler {
	if h.err != nil {
		return h
	}

	context := RenderContext{}
	scriptPaths := stringListFlag{}
	pixelWindowFlags := stringListFlag{}

	flagSet := flag.NewFlagSet("ray", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.Var(&scriptPaths, "script", "path to a canonical scene script")
	flagSet.StringVar(&context.Integrator, "integrator", "", "light transport integrator: path, bdpt, light_tracing")
	flagSet.Var(&pixelWindowFlags, "pixel-window", "pixel render window, for example 100:150,600:650; repeat for multiple windows")
	flagSet.IntVar(&context.Dimension, "dimension", 0, "scene dimension")
	flagSet.StringVar(&context.CameraID, "camera-id", "", "camera ID to render")
	flagSet.IntVar(&context.ThreadNum, "threads", 0, "worker thread count")
	flagSet.Func("widths", "film dimensions, for example 1920,1080", func(value string) error {
		width, err := parseWidths(value)
		if err == nil {
			context.FilmShapeOverride = width
		}
		return err
	})
	flagSet.Int64Var(&context.Samples, "samples", 0, "samples per pixel")
	flagSet.StringVar(&context.OutputFilm, "output-film", "", "output film path")
	flagSet.StringVar(&context.SpectrumMode, "spectrum-mode", "", "spectral sampling mode: hero_wavelength, sampled")
	flagSet.IntVar(&context.WavelengthSamples, "wavelength-samples", 0, "wavelength samples per camera sample in sampled mode")

	if err := flagSet.Parse(args); err != nil {
		h.err = err
		return h
	}

	if len(pixelWindowFlags) > 0 {
		windows, err := parsePixelWindowFlags(pixelWindowFlags)
		if err != nil {
			h.err = err
			return h
		}
		context.PixelWindows = windows
	}

	scriptPaths = append(scriptPaths, flagSet.Args()...)
	if len(scriptPaths) == 0 {
		scriptPaths = append(scriptPaths, defaultScriptPath)
	}
	if len(scriptPaths) != 1 {
		h.err = fmt.Errorf("engine accepts exactly one --script; use studio to merge multiple scripts")
		return h
	}
	context.ScriptPath = scriptPaths[0]
	if context.Dimension < 0 || context.Dimension == 1 {
		h.err = fmt.Errorf("dimension must be 0 or >= 2")
		return h
	}
	if context.ThreadNum < 0 {
		h.err = fmt.Errorf("threads must be >= 0")
		return h
	}
	if context.Samples < 0 {
		h.err = fmt.Errorf("samples must be >= 0")
		return h
	}
	if context.Integrator != "" && !isSupportedIntegrator(context.Integrator) {
		h.err = fmt.Errorf("unsupported integrator %q", context.Integrator)
		return h
	}
	if context.SpectrumMode != "" && !isSupportedSpectrumMode(context.SpectrumMode) {
		h.err = fmt.Errorf("unsupported spectrum-mode %q", context.SpectrumMode)
		return h
	}
	if context.WavelengthSamples < 0 {
		h.err = fmt.Errorf("wavelength-samples must be >= 0")
		return h
	}

	h.Context = context
	return h
}

func isSupportedIntegrator(value string) bool {
	_, err := ray_tracing.ParseIntegratorKind(value)
	return err == nil
}

func ResolveRenderContext(script *parser.Script, requested RenderContext) RenderContext {
	context := RenderContext{
		ScriptPath:        requested.ScriptPath,
		Integrator:        "path",
		Dimension:         3,
		ThreadNum:         runtime.NumCPU(),
		Samples:           defaultSamples,
		SpectrumMode:      "hero_wavelength",
		WavelengthSamples: 1,
	}

	if script != nil {
		if script.Render.Integrator != "" {
			context.Integrator = script.Render.Integrator
		}
		if script.Render.Dimension > 0 {
			context.Dimension = script.Render.Dimension
		}
		if script.Render.ThreadNum > 0 {
			context.ThreadNum = script.Render.ThreadNum
		}
		if script.Render.Samples > 0 {
			context.Samples = script.Render.Samples
		}
		context.CameraID = script.Render.CameraID
		if script.Render.SpectrumMode != "" {
			context.SpectrumMode = script.Render.SpectrumMode
		}
		if script.Render.WavelengthSamples > 0 {
			context.WavelengthSamples = script.Render.WavelengthSamples
		}
	}
	if requested.CameraID != "" {
		context.CameraID = requested.CameraID
	}
	context = applyRequestedContext(context, requested)
	if context.SpectrumMode == "sampled" && context.WavelengthSamples <= 1 {
		context.WavelengthSamples = 4
	}

	return context
}

func ResolveRenderContexts(script *parser.Script, requested RenderContext) []RenderContext {
	base := ResolveRenderContext(script, RenderContext{ScriptPath: requested.ScriptPath})

	jobs := []RenderContext{base}
	if script != nil && len(script.Renders) > 0 {
		jobs = make([]RenderContext, 0, len(script.Renders))
		for _, render := range script.Renders {
			context := applyRenderScriptToContext(base, render)
			if requested.CameraID != "" {
				context.CameraID = requested.CameraID
			}
			jobs = append(jobs, applyRequestedContext(context, requested))
		}
		return jobs
	}

	jobs[0] = applyRequestedContext(jobs[0], requested)
	return jobs
}

func applyRenderScriptToContext(context RenderContext, render parser.RenderScript) RenderContext {
	if render.Integrator != "" {
		context.Integrator = render.Integrator
	}
	if render.Dimension > 0 {
		context.Dimension = render.Dimension
	}
	if render.CameraID != "" {
		context.CameraID = render.CameraID
	}
	if render.ThreadNum > 0 {
		context.ThreadNum = render.ThreadNum
	}
	if render.Samples > 0 {
		context.Samples = render.Samples
	}
	if render.SpectrumMode != "" {
		context.SpectrumMode = render.SpectrumMode
	}
	if render.WavelengthSamples > 0 {
		context.WavelengthSamples = render.WavelengthSamples
	}
	if context.SpectrumMode == "sampled" && context.WavelengthSamples <= 1 {
		context.WavelengthSamples = 4
	}
	return context
}

func applyRequestedContext(context RenderContext, requested RenderContext) RenderContext {
	if requested.Integrator != "" {
		context.Integrator = requested.Integrator
	}
	if requested.CameraID != "" {
		context.CameraID = requested.CameraID
	}
	if requested.Dimension > 0 {
		context.Dimension = requested.Dimension
	}
	if requested.ThreadNum > 0 {
		context.ThreadNum = requested.ThreadNum
	}
	if len(requested.FilmShapeOverride) > 0 {
		context.FilmShapeOverride = append([]int(nil), requested.FilmShapeOverride...)
	}
	if requested.Samples > 0 {
		context.Samples = requested.Samples
	}
	if requested.OutputFilm != "" {
		context.OutputFilm = requested.OutputFilm
	}
	if requested.SpectrumMode != "" {
		context.SpectrumMode = requested.SpectrumMode
	}
	if requested.WavelengthSamples > 0 {
		context.WavelengthSamples = requested.WavelengthSamples
	}
	if len(requested.PixelWindows) > 0 {
		context.PixelWindows = clonePixelWindows(requested.PixelWindows)
	}
	if context.SpectrumMode == "sampled" && context.WavelengthSamples <= 1 {
		context.WavelengthSamples = 4
	}
	return context
}

func parsePixelWindowFlags(values []string) ([]camera.PixelWindow, error) {
	windows := make([]camera.PixelWindow, 0, len(values))
	for _, value := range values {
		window, err := parsePixelWindowFlag(value)
		if err != nil {
			return nil, err
		}
		windows = append(windows, window)
	}
	return windows, nil
}

func parseWidths(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	width := make([]int, len(parts))
	for i, part := range parts {
		parsed, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || parsed <= 0 {
			return nil, fmt.Errorf("widths[%d] must be a positive integer", i)
		}
		width[i] = parsed
	}
	return width, nil
}

func parsePixelWindowFlag(value string) (camera.PixelWindow, error) {
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return camera.PixelWindow{}, fmt.Errorf("pixel-window is empty")
	}

	min := make([]int, len(parts))
	max := make([]int, len(parts))
	for dim, part := range parts {
		lo, hi, err := parsePixelWindowAxis(part)
		if err != nil {
			return camera.PixelWindow{}, fmt.Errorf("pixel-window dimension %d: %w", dim, err)
		}
		min[dim] = lo
		max[dim] = hi
	}
	return camera.PixelWindow{Min: min, Max: max}, nil
}

func parsePixelWindowAxis(value string) (int, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, 0, fmt.Errorf("range is empty")
	}

	separator := ":"
	if strings.Contains(value, ":") {
		separator = ":"
	} else if strings.Contains(value, "-") {
		separator = "-"
	} else {
		return 0, 0, fmt.Errorf("range must use ':' or '-'")
	}

	parts := strings.Split(value, separator)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("range must contain exactly one separator")
	}

	lo, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid min %q", strings.TrimSpace(parts[0]))
	}
	hi, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid max %q", strings.TrimSpace(parts[1]))
	}
	if lo < 0 || hi < 0 {
		return 0, 0, fmt.Errorf("bounds must be non-negative")
	}
	if lo >= hi {
		return 0, 0, fmt.Errorf("min must be less than max")
	}
	return lo, hi, nil
}

func clonePixelWindows(windows []camera.PixelWindow) []camera.PixelWindow {
	if len(windows) == 0 {
		return nil
	}
	cloned := make([]camera.PixelWindow, len(windows))
	for i, window := range windows {
		cloned[i] = camera.PixelWindow{
			Min: append([]int(nil), window.Min...),
			Max: append([]int(nil), window.Max...),
		}
	}
	return cloned
}

type stringListFlag []string

func (s *stringListFlag) String() string {
	return fmt.Sprint([]string(*s))
}

func (s *stringListFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func isSupportedSpectrumMode(value string) bool {
	switch value {
	case "hero_wavelength", "sampled":
		return true
	default:
		return false
	}
}
