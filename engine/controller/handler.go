package controller

import (
	"fmt"
	"time"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/ray_tracing"
)

type Handler struct {
	err        error
	Scene      *model.Scene
	Script     *parser.Script
	ScriptPath string
	Job        ray_tracing.RenderJob
}

func NewHandler() *Handler {
	return &Handler{
		Scene: model.NewScene(geometry.DefaultSceneSpace()),
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

		job, err := ResolveRenderJob(render, h.Scene)
		if err != nil {
			h.err = fmt.Errorf("render[%d]: %w", idx, err)
			return h
		}
		h.Job = job
		h.Render().SaveFilm()
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

	if h.Job.Camera() == nil {
		h.err = fmt.Errorf("render camera is not configured")
		return h
	}

	renderFilm := h.Job.Film()
	if renderFilm == nil {
		h.err = fmt.Errorf("film is not initialized")
		return h
	}
	renderFilm.Reset()

	fmt.Printf("Starting rendering (integrator: %s)...\n", h.Job.Integrator())
	start := time.Now()

	renderHandler := ray_tracing.NewHandler(h.Scene.Space)
	renderHandler.MaxArc = h.Scene.MaxArc
	if err := renderHandler.TraceScene(h.Job); err != nil {
		h.err = err
		return h
	}

	fmt.Printf("Rendering completed in %v\n", time.Since(start))
	return h
}

func (h *Handler) SaveFilm() *Handler {
	if h.err != nil {
		return h
	}
	if h.Job.Film() == nil || h.Job.Output() == "" {
		h.err = fmt.Errorf("render job is not configured")
		return h
	}

	err := h.Job.Film().SaveToFile(h.Job.Output())
	if err != nil {
		h.err = err
		return h
	}

	return h
}
