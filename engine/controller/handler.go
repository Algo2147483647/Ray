package controller

import (
	"fmt"
	"time"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/camera"
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
		Renders()
	if h.err != nil {
		fmt.Printf("Error: %v\n", h.err)
		return 1
	}

	fmt.Println("Ray tracing completed successfully")
	return 0
}

func (h *Handler) Renders() *Handler {
	if h.err != nil {
		return h
	}

	if h.Script == nil || len(h.Script.Renders) == 0 {
		h.err = fmt.Errorf("no renders")
		return h
	}

	for idx, render := range h.Script.Renders {
		fmt.Printf("Starting render job %d/%d\n", idx+1, len(h.Script.Renders))

		context := mergeRenderContext(defaultRenderContext(), renderScriptContext(render))
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

	fmt.Printf("Starting rendering (integrator: %s)...\n", h.Context.Integrator)
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
	renderHandler.BDPTFallbackPolicy = ray_tracing.BDPTFallbackPolicy(h.Context.BDPTFallbackPolicy)
	renderHandler.SceneGeometry = h.Scene.Geometry
	renderHandler.MaxArc = h.Scene.MaxArc
	if err := renderHandler.TraceScene(
		h.Camera,
		h.Scene.ObjectTree,
		h.Context.Samples,
	); err != nil {
		h.err = err
		return h
	}

	fmt.Printf("Rendering completed in %v\n", time.Since(start))
	return h
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
