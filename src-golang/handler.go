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
	"src-golang/model/object/optics"
	"src-golang/ray_tracing"
	"time"
)

type Handler struct {
	err        error
	ScriptPath string
	Scene      *model.Scene
	img        [3]*mat.Dense
	imgout     *image.RGBA
	Width      int
	Height     int
}

func NewHandler() *Handler {
	Width := 800
	Height := 800

	h := &Handler{
		Scene:  model.NewScene(),
		Width:  Width,
		Height: Height,
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

	fmt.Printf("Loading scene from: %s\n", h.ScriptPath)
	err := controller.LoadSceneFromScript(controller.ReadScriptFile(h.ScriptPath), h.Scene)
	if err != nil {
		h.err = err
		return h
	}

	return h
}

func (h *Handler) BuildCamera() *Handler {
	if h.err != nil {
		return h
	}

	camera := &optics.Camera{
		Position:    mat.NewVecDense(3, []float64{200, 0, 0}),
		Up:          mat.NewVecDense(3, []float64{0, 0, 1}),
		Width:       h.Width,
		Height:      h.Height,
		AspectRatio: 1,
		FieldOfView: 120,
	}
	camera.SetLookAt(mat.NewVecDense(3, []float64{500, 0, 0}))
	h.Scene.Cameras = append(h.Scene.Cameras, camera)

	return h
}

func (h *Handler) Render() *Handler {
	if h.err != nil {
		return h
	}

	fmt.Println("Starting rendering...")
	start := time.Now()

	ray_tracing.TraceScene(h.Scene, h.img, 50)

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

	const outputPath = "output.png" // 修改文件扩展名
	fmt.Printf("Saving result to: %s\n", outputPath)

	file, err := os.Create(outputPath)
	if err != nil {
		h.err = err
		return h
	}
	defer file.Close()

	// 使用 PNG 编码器直接写入整个图像
	err = png.Encode(file, h.imgout)
	if err != nil {
		h.err = err
	}

	// Debug Record
	{
		rows, cols := h.img[0].Dims()
		result := make([][][3]float64, rows)
		for i := 0; i < rows; i++ {
			result[i] = make([][3]float64, cols)

			for j := 0; j < cols; j++ {
				result[i][j] = [3]float64{
					h.img[0].At(i, j),
					h.img[1].At(i, j),
					h.img[2].At(i, j),
				}
			}
		}

		// 创建并写入 JSON 文件
		file, err = os.Create("result.json")
		if err != nil {
			h.err = err
			return h
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(result)
		if err != nil {
			h.err = err
			return h
		}
	}

	{
		file, _ = os.Create("debug_traces.json")
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
