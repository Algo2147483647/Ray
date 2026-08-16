package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Algo2147483647/ray/engine/controller/factory"
	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/ray_tracing"
)

type Handler struct {
	err     error
	Scene   *model.Scene
	Script  *parser.Script
	Film    *camera.Film
	Camera  camera.RayCamera
	Context RenderContext
}

func NewHandler() *Handler {
	return &Handler{
		Scene: model.NewScene(),
	}
}

func Run(args []string) int {
	h := NewHandler().
		ParseRenderArgs(args).
		LoadScript().
		RenderJobs()
	if h.err != nil {
		fmt.Printf("Error: %v\n", h.err)
		return 1
	}

	fmt.Println("Ray tracing completed successfully")
	return 0
}

func (h *Handler) LoadScript() *Handler {
	if h.err != nil {
		return h
	}

	fmt.Printf("Loading scene from: %s\n", h.Context.ScriptPath)

	script, err := parser.ReadScriptFile(h.Context.ScriptPath)
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

func (h *Handler) ConfigureRenderContext(context RenderContext) *Handler {
	if h.err != nil {
		return h
	}

	renderCamera, err := h.selectRenderCamera(context.CameraIndex)
	if err != nil {
		h.err = err
		return h
	}

	filmShape := append([]int(nil), context.Width...)
	if len(filmShape) == 0 {
		h.err = fmt.Errorf("render film widths are not configured")
		return h
	}

	normalizedWindows, err := camera.NormalizePixelWindows(context.PixelWindows, filmShape)
	if err != nil {
		h.err = err
		return h
	}
	context.PixelWindows = normalizedWindows
	h.Context = context
	h.Film = camera.NewFilm(filmShape...)
	renderCamera.SetFilm(h.Film)
	h.Camera = renderCamera
	return h
}

func (h *Handler) RenderJobs() *Handler {
	if h.err != nil {
		return h
	}

	jobs := ResolveRenderContexts(h.Script, h.Context)
	for idx, context := range jobs {
		if len(jobs) > 1 {
			fmt.Printf("Starting render job %d/%d\n", idx+1, len(jobs))
		}
		h.ConfigureRenderContext(context).
			Render().
			SaveFilm(h.Context.OutputFilm)
		if h.err != nil {
			return h
		}
	}
	return h
}

func (h *Handler) selectRenderCamera(cameraIndex int) (camera.RayCamera, error) {
	if len(h.Scene.Cameras) == 0 {
		return nil, fmt.Errorf("scene has no cameras; use studio to generate a default camera")
	}
	if cameraIndex < 0 || cameraIndex >= len(h.Scene.Cameras) {
		return nil, fmt.Errorf("camera index %d out of range (available: %d)", cameraIndex, len(h.Scene.Cameras))
	}

	selectedCamera := h.Scene.Cameras[cameraIndex]
	preparedCamera, ok := selectedCamera.(interface {
		camera.RayCamera
		Prepare() error
	})
	if !ok {
		return nil, fmt.Errorf("camera does not support preparation")
	}
	if err := preparedCamera.Prepare(); err != nil {
		return nil, err
	}
	return selectedCamera, nil
}

func (h *Handler) Render() *Handler {
	if h.err != nil {
		return h
	}

	if h.Camera == nil {
		h.err = fmt.Errorf("render camera is not configured")
		return h
	} else if h.Film == nil {
		h.err = fmt.Errorf("film is not initialized")
		return h
	}

	fmt.Println("Starting rendering...")
	start := time.Now()

	renderHandler := ray_tracing.NewHandler()
	integratorKind, err := ray_tracing.ParseIntegratorKind(h.Context.Integrator)
	if err != nil {
		h.err = err
		return h
	}
	renderHandler.IntegratorKind = integratorKind
	renderHandler.ThreadNum = h.Context.ThreadNum
	renderHandler.SpectrumMode = renderSpectrumMode(h.Context.SpectrumMode)
	renderHandler.WavelengthSamples = h.Context.WavelengthSamples
	renderHandler.SceneGeometry = h.Scene.Geometry
	renderHandler.MaxArc = h.Scene.MaxArc
	if err := renderHandler.TraceScene(
		h.Camera,
		h.Scene.ObjectTree,
		h.Film,
		h.Context.Samples,
		h.Context.PixelWindows,
	); err != nil {
		h.err = err
		return h
	}

	fmt.Printf("Rendering completed in %v\n", time.Since(start))
	return h
}

func renderSpectrumMode(value string) optics.SpectrumMode {
	switch value {
	case "sampled":
		return optics.SpectrumModeSampledWavelengths
	default:
		return optics.SpectrumModeHeroWavelength
	}
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

func ensureParentDir(filename string) error {
	dir := filepath.Dir(filename)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
