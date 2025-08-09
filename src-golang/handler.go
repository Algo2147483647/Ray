package main

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
	"image"
	"image/color"
	"os"
	"src-golang/model"
	"src-golang/model/object"
	"src-golang/ray_tracing"
	"time"
)

type Handler struct {
	err        error
	ScriptPath string
	objTree    object.ObjectTree
	Camera     *model.Camera
	img        [3]*mat.Dense
	imgout     *image.RGBA
	Width      int
	Height     int
}

func NewHandler() *Handler {
	Width := 800
	Height := 800

	h := &Handler{
		Width:  Width,
		Height: Height,
		Camera: model.NewCamera(),
		imgout: image.NewRGBA(image.Rect(0, 0, Width, Height)),
	}
	for i, _ := range h.img {
		h.img[i] = mat.NewDense(h.Width, h.Height, nil)
	}

	return h
}

func (h *Handler) SetScriptPath(scriptPath string) *Handler {
	if h.err != nil {
		return h
	}

	h.ScriptPath = scriptPath
	return h
}

func (h *Handler) PreCheck() *Handler {
	if h.err != nil {
		return h
	}

	if _, err := os.Stat(h.ScriptPath); os.IsNotExist(err) {
		h.err = fmt.Errorf("script file not found: %s", h.ScriptPath)
	}
	return h
}

func (h *Handler) LoadScript() *Handler {
	if h.err != nil {
		return h
	}

	ray_tracing.LoadSceneFromScript(h.ScriptPath, &h.objTree)
	fmt.Printf("Loading scene from: %s\n", h.ScriptPath)
	return h
}

func (h *Handler) BuildCamera() *Handler {
	if h.err != nil {
		return h
	}

	h.Camera.Position = mat.NewVecDense(3, []float64{600.0, 1100.0, 600.0})
	h.Camera.Direction = mat.NewVecDense(3, []float64{400.0, -100.0, -100.0})

	return h
}

func (h *Handler) Render() *Handler {
	if h.err != nil {
		return h
	}

	fmt.Println("Starting rendering...")
	start := time.Now()

	ray_tracing.TraceScene(h.Camera, &h.objTree, h.img, 100)

	elapsed := time.Since(start)
	fmt.Printf("Rendering completed in %v\n", elapsed)
	return h
}

func (h *Handler) BuildResult() *Handler {
	if h.err != nil {
		return h
	}

	fmt.Println("Processing image...")
	for i := 0; i < 800; i++ {
		for j := 0; j < 800; j++ {
			r := uint8(min(h.img[0].At(i, j)*255, 255))
			g := uint8(min(h.img[1].At(i, j)*255, 255))
			b := uint8(min(h.img[2].At(i, j)*255, 255))
			h.imgout.Set(i, j, color.RGBA{r, g, b, 255})
		}
	}
	return h
}

func (h *Handler) SaveResult() *Handler {
	if h.err != nil {
		return h
	}

	const outputPath = "output.ppm"
	fmt.Printf("Saving result to: %s\n", outputPath)

	file, err := os.Create(outputPath)
	if err != nil {
		h.err = err
		return h
	}
	defer file.Close()

	// 写入PPM文件头
	fmt.Fprintf(file, "P6\n800 800\n255\n")

	// 写入像素数据
	for y := 0; y < 800; y++ {
		for x := 0; x < 800; x++ {
			r, g, b, _ := h.imgout.At(x, y).RGBA()
			file.Write([]byte{uint8(r), uint8(g), uint8(b)})
		}
	}
	return h
}
