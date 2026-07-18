package controller

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/controller/factory"
	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/ray_tracing"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

type Handler struct {
	err          error
	Scene        *model.Scene
	Script       *parser.Script
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

	if len(overrides.ScriptPaths) == 1 && overrides.ScriptPath == defaultScriptPath {
		fmt.Printf("Using default script: %s\n", overrides.ScriptPath)
	}

	h := NewHandler().
		LoadScripts(overrides.ScriptPaths).
		RenderJobs(overrides)

	if h.err != nil {
		fmt.Printf("Error: %v\n", h.err)
		return 1
	}

	fmt.Println("Ray tracing completed successfully")
	return 0
}

func (h *Handler) LoadScript(scriptPath string) *Handler {
	return h.LoadScripts([]string{scriptPath})
}

func (h *Handler) LoadScripts(scriptPaths []string) *Handler {
	if h.err != nil {
		return h
	}

	if len(scriptPaths) == 1 {
		fmt.Printf("Loading scene from: %s\n", scriptPaths[0])
	} else {
		fmt.Printf("Loading scene from %d scripts:\n", len(scriptPaths))
		for _, scriptPath := range scriptPaths {
			fmt.Printf("  %s\n", scriptPath)
		}
	}

	script, err := parser.ReadScriptFiles(scriptPaths)
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

func (h *Handler) ConfigureRender(overrides RenderOverrides) *Handler {
	config := ResolveRenderConfig(h.Script, overrides)
	return h.ConfigureRenderConfig(config)
}

func (h *Handler) ConfigureRenderConfig(config RenderConfig) *Handler {
	if h.err != nil {
		return h
	}

	renderCamera, filmShape, err := h.selectRenderCamera(config.CameraIndex, config.Width, config.Height)
	if err != nil {
		h.err = err
		return h
	}

	if len(filmShape) > 0 {
		config.Width = filmShape[0]
	}
	if len(filmShape) > 1 {
		config.Height = filmShape[1]
	}
	h.Config = config
	h.ActiveCamera = renderCamera
	h.Film = camera.NewFilm(filmShape...)
	h.Film.ColorSpace = renderColorSpace(config.ColorSpace)
	return h
}

func (h *Handler) RenderJobs(overrides RenderOverrides) *Handler {
	if h.err != nil {
		return h
	}

	jobs := ResolveRenderConfigs(h.Script, overrides)
	for idx, config := range jobs {
		if len(jobs) > 1 {
			fmt.Printf("Starting render job %d/%d\n", idx+1, len(jobs))
		}
		h.ConfigureRenderConfig(config).
			Render().
			ResumeFilm().
			SaveOutputs()
		if h.err != nil {
			return h
		}
	}
	return h
}

func (h *Handler) selectRenderCamera(cameraIndex, width, height int) (camera.Camera, []int, error) {
	if len(h.Scene.Cameras) == 0 {
		defaultCamera, err := factory.BuildCamera3DFromScript(factory.DefaultCameraScript())
		if err != nil {
			return nil, nil, err
		}
		h.Scene.Cameras = append(h.Scene.Cameras, defaultCamera)
	}

	if cameraIndex < 0 || cameraIndex >= len(h.Scene.Cameras) {
		return nil, nil, fmt.Errorf("camera index %d out of range (available: %d)", cameraIndex, len(h.Scene.Cameras))
	}

	selectedCamera := h.Scene.Cameras[cameraIndex]
	switch c := selectedCamera.(type) {
	case *camera.Camera3D:
		resolvedWidth := firstPositiveInt(width, defaultRenderWidth)
		resolvedHeight := firstPositiveInt(height, defaultRenderHeight)
		c.Width = resolvedWidth
		c.Height = resolvedHeight
		if err := c.Prepare(); err != nil {
			return nil, nil, err
		}
		return c, []int{resolvedWidth, resolvedHeight}, nil
	case *camera.HyperbolicCamera:
		resolvedWidth := firstPositiveInt(width, defaultRenderWidth)
		resolvedHeight := firstPositiveInt(height, defaultRenderHeight)
		c.Width = resolvedWidth
		c.Height = resolvedHeight
		if err := c.Prepare(); err != nil {
			return nil, nil, err
		}
		return c, []int{resolvedWidth, resolvedHeight}, nil
	case *camera.SphericalCamera:
		resolvedWidth := firstPositiveInt(width, defaultRenderWidth)
		resolvedHeight := firstPositiveInt(height, defaultRenderHeight)
		c.Width = resolvedWidth
		c.Height = resolvedHeight
		if err := c.Prepare(); err != nil {
			return nil, nil, err
		}
		return c, []int{resolvedWidth, resolvedHeight}, nil
	case *camera.CameraNDim:
		if len(c.Width) == 0 {
			return nil, nil, fmt.Errorf("n_dim camera has no film widths")
		}
		filmShape := append([]int(nil), c.Width...)
		if width > 0 {
			filmShape[0] = width
		}
		if height > 0 && len(filmShape) > 1 {
			filmShape[1] = height
		}
		c.Width = append([]int(nil), filmShape...)
		if err := c.Prepare(); err != nil {
			return nil, nil, err
		}
		return c, filmShape, nil
	default:
		return selectedCamera, []int{firstPositiveInt(width, defaultRenderWidth), firstPositiveInt(height, defaultRenderHeight)}, nil
	}
}

func (h *Handler) Render() *Handler {
	if h.err != nil {
		return h
	}

	if h.ActiveCamera == nil {
		h.err = fmt.Errorf("render camera is not configured")
		return h
	} else if h.Film == nil {
		h.err = fmt.Errorf("film is not initialized")
		return h
	}

	fmt.Println("Starting rendering...")
	start := time.Now()

	renderHandler := ray_tracing.NewHandler()
	renderHandler.ThreadNum = h.Config.ThreadNum
	renderHandler.SpectrumMode = renderSpectrumMode(h.Config.SpectrumMode)
	renderHandler.WavelengthSamples = h.Config.WavelengthSamples
	renderHandler.FilmColorSpace = h.Film.ColorSpace
	renderHandler.SceneGeometry = h.Scene.Geometry
	renderHandler.MaxArc = h.Scene.MaxArc
	renderHandler.TraceScene(h.ActiveCamera, h.Scene.ObjectTree, h.Film, h.Config.Samples)

	fmt.Printf("Rendering completed in %v\n", time.Since(start))
	return h
}

func renderSpectrumMode(value string) optics.SpectrumMode {
	switch value {
	case "rgb":
		return optics.SpectrumModeRGB
	case "sampled":
		return optics.SpectrumModeSampledWavelengths
	default:
		return optics.SpectrumModeHeroWavelength
	}
}

func renderColorSpace(value string) camera.FilmColorSpace {
	switch value {
	case string(camera.FilmColorSpaceXYZ):
		return camera.FilmColorSpaceXYZ
	case string(camera.FilmColorSpaceACEScg):
		return camera.FilmColorSpaceACEScg
	default:
		return camera.FilmColorSpaceLinearSRGB
	}
}

func (h *Handler) SaveOutputs() *Handler {
	if h.err != nil {
		return h
	}

	if h.Config.OutputFilm != "" {
		h.SaveFilm(h.Config.OutputFilm)
	}

	if h.Config.OutputImage != "" {
		h.SaveImg(h.Config.OutputImage)
	}

	return h
}

func (h *Handler) ResumeFilm() *Handler {
	if h.err != nil {
		return h
	}
	if h.Config.ResumeFilm == "" {
		return h
	}

	fmt.Printf("Merging existing film: %s\n", h.Config.ResumeFilm)
	return h.MergeFilm(h.Config.ResumeFilm)
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

	img := h.Film.ToImageWithOptions(camera.ImageOptions{
		Exposure:    h.Config.Exposure,
		ToneMapping: camera.ToneMapping(h.Config.ToneMapping),
		Gamma:       h.Config.Gamma,
	})
	if err := png.Encode(file, img); err != nil {
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
