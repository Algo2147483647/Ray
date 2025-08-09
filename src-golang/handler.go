package main

import (
	"fmt"
	"gonum.org/v1/gonum/spatial/r3"
	"image"
	"image/color"
	"os"
	"src-golang/model"
	"src-golang/model/object"
	"time"
)

type Handler struct {
	err        error
	ScriptPath string
	objTree    object.ObjectTree
	camera     model.Camera
	img        [3][800][800]float32
	imgout     *image.RGBA
}

func NewHandler() *Handler {
	return &Handler{
		imgout: image.NewRGBA(image.Rect(0, 0, 800, 800)),
	}
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

	// 模拟场景加载
	fmt.Printf("Loading scene from: %s\n", h.ScriptPath)
	return h
}

func (h *Handler) BuildCamera() *Handler {
	if h.err != nil {
		return h
	}

	h.camera = model.Camera{
		Position:  r3.Vec{600.0, 1100.0, 600.0},
		Direction: r3.Vec{400.0, -100.0, -100.0},
	}
	return h
}

func (h *Handler) Render() *Handler {
	if h.err != nil {
		return h
	}

	fmt.Println("Starting rendering...")
	start := time.Now()

	// 简化的渲染逻辑 - 实际应实现光线追踪算法
	for i := 0; i < 800; i++ {
		for j := 0; j < 800; j++ {
			// 示例渲染模式
			h.img[0][i][j] = float32(i) / 800
			h.img[1][i][j] = float32(j) / 800
			h.img[2][i][j] = 0.5
		}
	}

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
			r := uint8(min(h.img[0][i][j]*255, 255))
			g := uint8(min(h.img[1][i][j]*255, 255))
			b := uint8(min(h.img[2][i][j]*255, 255))
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
