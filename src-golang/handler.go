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
	"src-golang/utils"
	"time"
)

type Handler struct {
	err    error
	Scene  *model.Scene
	img    [3]*mat.Dense
	Width  int
	Height int
}

func NewHandler(Width, Height int) *Handler {
	h := &Handler{
		Scene:  model.NewScene(),
		Width:  Width,
		Height: Height,
	}
	for i, _ := range h.img {
		h.img[i] = mat.NewDense(h.Width, h.Height, nil)
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

func (h *Handler) BuildCamera() *Handler {
	if h.err != nil {
		return h
	}

	camera := &optics.Camera{
		Position:    mat.NewVecDense(3, []float64{-1.7, 0.1, 0.5}),
		Up:          mat.NewVecDense(3, []float64{0, 0, 1}),
		Width:       h.Width,
		Height:      h.Height,
		AspectRatio: 1,
		FieldOfView: 100,
	}
	camera.SetLookAt(mat.NewVecDense(3, []float64{2, 0, 0}))
	h.Scene.Cameras = append(h.Scene.Cameras, camera)

	return h
}

func (h *Handler) Render(samples, samplesSt int) *Handler {
	if h.err != nil {
		return h
	}

	var img [3]*mat.Dense
	for i, _ := range h.img {
		img[i] = mat.NewDense(h.Width, h.Height, nil)
	}

	fmt.Println("Starting rendering...")
	start := time.Now()

	ray_tracing.TraceScene(h.Scene, img, samples)

	elapsed := time.Since(start)
	fmt.Printf("Rendering completed in %v\n", elapsed)

	totalSamples := samples + samplesSt
	if samplesSt > 0 {
		for i := range h.img { // 使用加权平均合并采样结果
			for x := 0; x < h.Width; x++ {
				for y := 0; y < h.Height; y++ {
					mergedValue := (h.img[i].At(x, y)*float64(samplesSt) + img[i].At(x, y)*float64(samples)) / float64(totalSamples) // 加权平均: (oldValue * samplesSt + newValue * samples) / totalSamples
					h.img[i].Set(x, y, mergedValue)
				}
			}
		}
	} else {
		for i := range h.img {
			h.img[i].Copy(img[i])
		}
	}
	return h
}

func (h *Handler) SaveDebugInfo(filename string) *Handler {
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
	return h
}

func (h *Handler) SaveImg(filename string) *Handler {
	if h.err != nil {
		return h
	}

	fmt.Printf("Saving result to: %s\n", filename)

	imgout := image.NewRGBA(image.Rect(0, 0, h.Width, h.Height))
	for i := 0; i < h.Width; i++ {
		for j := 0; j < h.Height; j++ {
			r := uint8(min(h.img[0].At(i, j)*255, 255))
			g := uint8(min(h.img[1].At(i, j)*255, 255))
			b := uint8(min(h.img[2].At(i, j)*255, 255))
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

	if err := utils.SaveMatrices(filename, h.img); err != nil {
		panic(err)
	}

	return h
}

func (h *Handler) LoadResult(filename string) *Handler {
	if h.err != nil {
		return h
	}

	h.img, h.err = utils.LoadMatrices(filename)
	if h.err != nil {
		panic(h.err)
	}

	return h
}
