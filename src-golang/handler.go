package main

import (
	"encoding/json"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"image"
	"image/color"
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

func (h *Handler) BuildCamera(Width, Height int) *Handler {
	if h.err != nil {
		return h
	}

	camera := camera.NewCamera3D()
	camera.Position = mat.NewVecDense(utils.Dimension, []float64{-1.7, 0.1, 0.5})
	camera.Up = mat.NewVecDense(utils.Dimension, []float64{0, 0, 1})
	camera.Width = Width
	camera.Height = Height
	camera.AspectRatio = 1
	camera.FieldOfView = 100
	camera.SetLookAt(mat.NewVecDense(utils.Dimension, []float64{2, 0, 0}))
	c := camera.Camera(camera)
	h.Scene.Cameras = append(h.Scene.Cameras, c)

	return h
}

func (h *Handler) Render(samples, samplesSt int64) *Handler {
	if h.err != nil {
		return h
	}

	c := h.Scene.Cameras[0].(*camera.Camera3D)
	var img [3]*mat.Dense
	for i, _ := range img {
		img[i] = mat.NewDense(c.Width, c.Height, nil)
	}

	fmt.Println("Starting rendering...")
	start := time.Now()

	renderHandler := ray_tracing.NewHandler()
	renderHandler.TraceScene(h.Scene, img, samples)

	elapsed := time.Since(start)
	fmt.Printf("Rendering completed in %v\n", elapsed)

	c.MergeImage(img, samples, samplesSt)
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
	c := h.Scene.Cameras[0].(*camera.Camera3D)
	imgout := image.NewRGBA(image.Rect(0, 0, c.Width, c.Height))
	for i := 0; i < c.Width; i++ {
		for j := 0; j < c.Height; j++ {
			r := uint8(min(c.Image[0].At(i, j)*255, 255))
			g := uint8(min(c.Image[1].At(i, j)*255, 255))
			b := uint8(min(c.Image[2].At(i, j)*255, 255))
			imgout.Set(i, j, color.RGBA{r, g, b, 255})
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		h.err = err
		return h
	}
	defer file.Close()

	err = png.Encode(file, imgout) // 使用 PNG 编码器直接写入整个图像
	if err != nil {
		h.err = err
	}

	return h
}

func (h *Handler) SaveResult(filename string) *Handler {
	if h.err != nil {
		return h
	}

	c := h.Scene.Cameras[0].(*camera.Camera3D)
	c.SaveImage(filename)

	return h
}

func (h *Handler) LoadResult(filename string) *Handler {
	if h.err != nil {
		return h
	}
	c := h.Scene.Cameras[0].(*camera.Camera3D)
	c.LoadImage(filename)

	return h
}
