package main

import (
	"encoding/json"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"image/png"
	"os"
	"src-golang/controller"
	"src-golang/model"
	"src-golang/model/camera"
	"src-golang/model/optics"
	"src-golang/ray_tracing"
	"src-golang/utils"
	"time"
)

type Handler struct {
	err   error
	Scene *model.Scene
	Film  *camera.Film
}

func NewHandler() *Handler {
	h := &Handler{
		Scene: model.NewScene(),
	}

	return h
}

func (h *Handler) LoadScript(ScriptPath string) *Handler {
	if h.err != nil {
		return h
	}

	fmt.Printf("Loading scene from: %s\n", ScriptPath)
	err := controller.LoadSceneFromScript(controller.ReadScriptFile(ScriptPath), h.Scene)
	if err != nil {
		h.err = err
		return h
	}

	return h
}

func (h *Handler) BuildCamera(Width ...int) *Handler {
	if h.err != nil {
		return h
	}

	c := &camera.CameraNDim{
		Position: mat.NewVecDense(utils.Dimension, []float64{0, 0, 0, 0}),
		Coordinates: []*mat.VecDense{
			mat.NewVecDense(utils.Dimension, []float64{1, 0, 0, 0}),
			mat.NewVecDense(utils.Dimension, []float64{0, 1, 0, 0}),
			mat.NewVecDense(utils.Dimension, []float64{0, 0, 1, 0}),
			mat.NewVecDense(utils.Dimension, []float64{0, 0, 0, 1}),
		},
		Width:       Width,
		FieldOfView: []float64{100, 100, 100},
	}
	h.Scene.Cameras = append(h.Scene.Cameras, c)
	return h
}

func (h *Handler) BuildFilm(Width ...int) *Handler {
	if h.err != nil {
		return h
	}

	h.Film = camera.NewFilm(Width...)

	return h
}

func (h *Handler) Render(samples int64) *Handler {
	if h.err != nil {
		return h
	}

	fmt.Println("Starting rendering...")
	start := time.Now()

	renderHandler := ray_tracing.NewHandler()
	renderHandler.TraceScene(h.Scene, h.Film, samples)

	elapsed := time.Since(start)
	fmt.Printf("Rendering completed in %v\n", elapsed)
	return h
}

func (h *Handler) SaveDebugInfo(filename string) *Handler {
	if h.err != nil {
		return h
	}

	if utils.IsDebug {
		file, err := os.Create(filename)
		if err != nil {
			h.err = err
			return h
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(optics.DebugRayTraces)
		if err != nil {
			h.err = err
			return h
		}
	}

	return h
}

func (h *Handler) SaveImg(filename string) *Handler {
	if h.err != nil {
		return h
	}

	fmt.Printf("Saving result to: %s\n", filename)

	file, err := os.Create(filename)
	if err != nil {
		h.err = err
		return h
	}
	defer file.Close()

	err = png.Encode(file, h.Film.ToImage()) // 使用 PNG 编码器直接写入整个图像
	if err != nil {
		h.err = err
	}

	return h
}

func (h *Handler) SaveFilm(filename string) *Handler {
	if h.err != nil {
		return h
	}

	err := h.Film.SaveToFile(filename)
	if err != nil {
		h.err = err
		return h
	}

	return h
}

func (h *Handler) MergeFilm(filename string) *Handler {
	if h.err != nil {
		return h
	}

	t := camera.NewFilm(h.Film.Data[0].Shape...)
	err := t.LoadFromFile(filename)
	if err != nil {
		h.err = err
		return h
	}

	h.Film.Merge(t)
	return h
}
