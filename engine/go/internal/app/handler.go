package app

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/Algo2147483647/ray/engine/go/internal/controller"
	"github.com/Algo2147483647/ray/engine/go/internal/model"
	"github.com/Algo2147483647/ray/engine/go/internal/model/camera"
	"github.com/Algo2147483647/ray/engine/go/internal/model/optics"
	"github.com/Algo2147483647/ray/engine/go/internal/ray_tracing"
	"github.com/Algo2147483647/ray/engine/go/internal/utils"
)

type Handler struct {
	err          error
	Scene        *model.Scene
	Script       *controller.Script
	Film         *camera.Film
	ActiveCamera camera.Camera
	Config       RenderConfig
}

func NewHandler() *Handler {
	return &Handler{
		Scene: model.NewScene(),
	}
}

func Run(args []string) int {
	overrides, err := ParseRenderOverrides(args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}

	if overrides.ScriptPath == defaultScriptPath {
		fmt.Printf("Using default script: %s\n", overrides.ScriptPath)
	}

	h := NewHandler().
		LoadScript(overrides.ScriptPath).
		ConfigureRender(overrides).
		Render().
		SaveOutputs()

	if h.err != nil {
		fmt.Printf("Error: %v\n", h.err)
		return 1
	}

	fmt.Println("Ray tracing completed successfully")
	return 0
}

func (h *Handler) LoadScript(scriptPath string) *Handler {
	if h.err != nil {
		return h
	}

	fmt.Printf("Loading scene from: %s\n", scriptPath)

	script, err := controller.ReadScriptFile(scriptPath)
	if err != nil {
		h.err = err
		return h
	}

	h.Script = script
	if err := controller.LoadSceneFromScript(script, h.Scene); err != nil {
		h.err = err
		return h
	}

	return h
}

func (h *Handler) ConfigureRender(overrides RenderOverrides) *Handler {
	if h.err != nil {
		return h
	}

	config := ResolveRenderConfig(h.Script, overrides)
	renderCamera, width, height, err := h.selectRenderCamera(config.CameraIndex, config.Width, config.Height)
	if err != nil {
		h.err = err
		return h
	}

	config.Width = width
	config.Height = height
	h.Config = config
	h.ActiveCamera = renderCamera
	h.Film = camera.NewFilm(width, height)
	return h
}

func (h *Handler) selectRenderCamera(cameraIndex, width, height int) (camera.Camera, int, int, error) {
	if len(h.Scene.Cameras) == 0 {
		defaultCamera, err := controller.BuildCamera3DFromScript(controller.DefaultCameraScript())
		if err != nil {
			return nil, 0, 0, err
		}
		h.Scene.Cameras = append(h.Scene.Cameras, defaultCamera)
	}

	if cameraIndex < 0 || cameraIndex >= len(h.Scene.Cameras) {
		return nil, 0, 0, fmt.Errorf("camera index %d out of range (available: %d)", cameraIndex, len(h.Scene.Cameras))
	}

	selectedCamera := h.Scene.Cameras[cameraIndex]
	switch c := selectedCamera.(type) {
	case *camera.Camera3D:
		resolvedWidth := firstPositiveInt(width, c.Width, defaultRenderWidth)
		resolvedHeight := firstPositiveInt(height, c.Height, defaultRenderHeight)
		c.Width = resolvedWidth
		c.Height = resolvedHeight
		c.AspectRatio = float64(resolvedWidth) / float64(resolvedHeight)
		return c, resolvedWidth, resolvedHeight, nil
	default:
		return selectedCamera, firstPositiveInt(width, defaultRenderWidth), firstPositiveInt(height, defaultRenderHeight), nil
	}
}

func (h *Handler) Render() *Handler {
	if h.err != nil {
		return h
	}

	if h.ActiveCamera == nil {
		h.err = fmt.Errorf("render camera is not configured")
		return h
	}
	if h.Film == nil {
		h.err = fmt.Errorf("film is not initialized")
		return h
	}

	fmt.Println("Starting rendering...")
	start := time.Now()

	renderHandler := ray_tracing.NewHandler()
	renderHandler.ThreadNum = h.Config.ThreadNum
	renderHandler.TraceScene(h.ActiveCamera, h.Scene.ObjectTree, h.Film, h.Config.Samples)

	elapsed := time.Since(start)
	fmt.Printf("Rendering completed in %v\n", elapsed)
	return h
}

func (h *Handler) SaveOutputs() *Handler {
	if h.err != nil {
		return h
	}

	if h.Config.OutputFilm != "" {
		h.SaveFilm(h.Config.OutputFilm)
	}
	if h.err != nil {
		return h
	}

	if h.Config.OutputImage != "" {
		h.SaveImg(h.Config.OutputImage)
	}
	if h.err != nil {
		return h
	}

	if h.Config.DebugOutput != "" {
		h.SaveDebugInfo(h.Config.DebugOutput)
	}

	return h
}

func (h *Handler) SaveDebugInfo(filename string) *Handler {
	if h.err != nil {
		return h
	}

	if utils.IsDebug {
		if err := ensureParentDir(filename); err != nil {
			h.err = err
			return h
		}

		file, err := os.Create(filename)
		if err != nil {
			h.err = err
			return h
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(optics.DebugRayTraces); err != nil {
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

	if err := ensureParentDir(filename); err != nil {
		h.err = err
		return h
	}

	file, err := os.Create(filename)
	if err != nil {
		h.err = err
		return h
	}
	defer file.Close()

	if err := png.Encode(file, h.Film.ToImage()); err != nil {
		h.err = err
	}

	return h
}

func (h *Handler) SaveFilm(filename string) *Handler {
	if h.err != nil {
		return h
	}

	if err := ensureParentDir(filename); err != nil {
		h.err = err
		return h
	}

	if err := h.Film.SaveToFile(filename); err != nil {
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
	if err := t.LoadFromFile(filename); err != nil {
		h.err = err
		return h
	}

	h.Film.Merge(t)
	return h
}

func ensureParentDir(filename string) error {
	dir := filepath.Dir(filename)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
