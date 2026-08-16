package controller

import (
	"fmt"
	"time"

	"github.com/Algo2147483647/ray/engine/controller/factory"
	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/ray_tracing"
)

type Handler struct {
	err        error
	Scene      *model.Scene
	Script     *parser.Script
	ScriptPath string
	Camera     camera.RayCamera
	Context    RenderContext
}

func NewHandler() *Handler {
	return &Handler{
		Scene: model.NewScene(),
	}
}

func Run(args []string) int {
	h := NewHandler().
		ParseArgs(args).
		LoadScript().
		RenderJobs()
	if h.err != nil {
		fmt.Printf("Error: %v\n", h.err)
		return 1
	}

	fmt.Println("Ray tracing completed successfully")
	return 0
}

func (h *Handler) LoadScript() *Handler {
	if h.err != nil {
		return h
	}

	fmt.Printf("Loading scene from: %s\n", h.ScriptPath)

	script, err := parser.ReadScriptFile(h.ScriptPath)
	if err != nil {
		h.err = err
		return h
	}

	h.Script = script
	if err := factory.LoadSceneFromScript(script, h.Scene); err != nil {
		h.err = err
		return h
	}

	return h
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
	normalizedWindows, err := camera.NormalizePixelWindows(film.PixelWindows, film.Shape)
	if err != nil {
		h.err = err
		return h
	}

	context.PixelWindows = normalizedWindows
	context.OutputFilm = film.OutputFilm
	if context.OutputFilm == "" {
		context.OutputFilm = defaultOutputFilm
	}
	h.Context = context
	return h
}

func (h *Handler) RenderJobs() *Handler {
	if h.err != nil {
		return h
	}

	jobs := make([]RenderContext, 0, len(h.Script.Renders))
	if h.Script == nil || len(h.Script.Renders) == 0 {
		jobs = []RenderContext{defaultRenderContext()}
	}

	for _, render := range h.Script.Renders {
		job := mergeRenderContext(defaultRenderContext(), renderScriptContext(render))
		jobs = append(jobs, job)
	}

	for idx, context := range jobs {
		if len(jobs) > 1 {
			fmt.Printf("Starting render job %d/%d\n", idx+1, len(jobs))
		}

		h.ConfigureRenderContext(context).
			Render().
			SaveFilm(h.Context.OutputFilm)
		if h.err != nil {
			return h
		}
	}
	return h
}

func (h *Handler) Render() *Handler {
	if h.err != nil {
		return h
	}

	if h.Camera == nil {
		h.err = fmt.Errorf("render camera is not configured")
		return h
	}

	film := h.Camera.GetFilm()
	if film == nil {
		h.err = fmt.Errorf("film is not initialized")
		return h
	}
	film.Reset()

	fmt.Println("Starting rendering...")
	start := time.Now()

	var err error

	renderHandler := ray_tracing.NewHandler()
	renderHandler.IntegratorKind, err = ray_tracing.ParseIntegratorKind(h.Context.Integrator)
	if err != nil {
		h.err = err
		return h
	}
	renderHandler.ThreadNum = h.Context.ThreadNum
	renderHandler.SpectrumMode = renderSpectrumMode(h.Context.SpectrumMode)
	renderHandler.WavelengthSamples = h.Context.WavelengthSamples
	renderHandler.SceneGeometry = h.Scene.Geometry
	renderHandler.MaxArc = h.Scene.MaxArc
	if err := renderHandler.TraceScene(
		h.Camera,
		h.Scene.ObjectTree,
		h.Context.Samples,
		h.Context.PixelWindows,
	); err != nil {
		h.err = err
		return h
	}

	fmt.Printf("Rendering completed in %v\n", time.Since(start))
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

func (h *Handler) SaveFilm(filename string) *Handler {
	if h.err != nil {
		return h
	}

	err := h.Camera.GetFilm().SaveToFile(filename)
	if err != nil {
		h.err = err
		return h
	}

	return h
}
