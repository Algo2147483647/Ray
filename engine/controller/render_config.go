package controller

import (
	"flag"
	"fmt"
	"github.com/Algo2147483647/ray/engine/controller/parser"
	"io"
	"runtime"

	"github.com/Algo2147483647/ray/engine/model/camera"
)

const (
	defaultScriptPath   = "../../examples/scenes/default.json"
	defaultRenderWidth  = 400
	defaultRenderHeight = 400
	defaultSamples      = int64(20)
	defaultOutputImage  = "../../outputs/output.png"
	defaultOutputFilm   = "../../outputs/img.bin"
)

type RenderOverrides struct {
	ScriptPath        string
	ScriptPaths       []string
	Dimension         int
	CameraIndex       int
	ThreadNum         int
	Width             int
	Height            int
	Samples           int64
	OutputImage       string
	OutputFilm        string
	ResumeFilm        string
	Exposure          float64
	ToneMapping       string
	Gamma             float64
	SpectrumMode      string
	WavelengthSamples int
	ColorSpace        string
}

type RenderConfig struct {
	ScriptPath        string
	Dimension         int
	CameraIndex       int
	ThreadNum         int
	Width             int
	Height            int
	Samples           int64
	OutputImage       string
	OutputFilm        string
	ResumeFilm        string
	Exposure          float64
	ToneMapping       string
	Gamma             float64
	SpectrumMode      string
	WavelengthSamples int
	ColorSpace        string
}

func ParseRenderOverrides(args []string) (RenderOverrides, error) {
	overrides := RenderOverrides{
		CameraIndex: -1,
	}
	scriptPaths := stringListFlag{}

	flagSet := flag.NewFlagSet("ray", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.Var(&scriptPaths, "script", "path to a scene script; repeat to merge multiple scripts")
	flagSet.IntVar(&overrides.Dimension, "dimension", 0, "scene dimension")
	flagSet.IntVar(&overrides.CameraIndex, "camera-index", -1, "camera index to render")
	flagSet.IntVar(&overrides.ThreadNum, "threads", 0, "worker thread count")
	flagSet.IntVar(&overrides.Width, "width", 0, "output width")
	flagSet.IntVar(&overrides.Height, "height", 0, "output height")
	flagSet.Int64Var(&overrides.Samples, "samples", 0, "samples per pixel")
	flagSet.StringVar(&overrides.OutputImage, "output-image", "", "output image path")
	flagSet.StringVar(&overrides.OutputFilm, "output-film", "", "output film path")
	flagSet.StringVar(&overrides.ResumeFilm, "resume-film", "", "existing film path to merge before saving outputs")
	flagSet.Float64Var(&overrides.Exposure, "exposure", 0, "output exposure multiplier")
	flagSet.StringVar(&overrides.ToneMapping, "tone-mapping", "", "output tone mapping: linear, reinhard, aces")
	flagSet.Float64Var(&overrides.Gamma, "gamma", 0, "output gamma, for example 2.2")
	flagSet.StringVar(&overrides.SpectrumMode, "spectrum-mode", "", "spectrum mode: rgb, hero_wavelength, sampled")
	flagSet.IntVar(&overrides.WavelengthSamples, "wavelength-samples", 0, "wavelength samples per camera sample in sampled mode")
	flagSet.StringVar(&overrides.ColorSpace, "working-space", "", "film working space: linear_srgb, acescg, xyz")

	if err := flagSet.Parse(args); err != nil {
		return RenderOverrides{}, err
	}

	overrides.ScriptPaths = append(overrides.ScriptPaths, scriptPaths...)
	if len(overrides.ScriptPaths) == 0 && len(flagSet.Args()) > 0 {
		overrides.ScriptPaths = append(overrides.ScriptPaths, flagSet.Args()...)
	}
	if len(overrides.ScriptPaths) == 0 {
		overrides.ScriptPaths = []string{defaultScriptPath}
	}
	overrides.ScriptPath = overrides.ScriptPaths[0]
	if len(overrides.ScriptPaths) == 1 {
		overrides.ScriptPath = overrides.ScriptPaths[0]
	}
	if overrides.CameraIndex < -1 {
		return RenderOverrides{}, fmt.Errorf("camera-index must be >= -1")
	}
	if overrides.Dimension < 0 || overrides.Dimension == 1 {
		return RenderOverrides{}, fmt.Errorf("dimension must be 0 or >= 2")
	}
	if overrides.ThreadNum < 0 {
		return RenderOverrides{}, fmt.Errorf("threads must be >= 0")
	}
	if overrides.Width < 0 || overrides.Height < 0 {
		return RenderOverrides{}, fmt.Errorf("width and height must be >= 0")
	}
	if overrides.Samples < 0 {
		return RenderOverrides{}, fmt.Errorf("samples must be >= 0")
	}
	if overrides.Exposure < 0 {
		return RenderOverrides{}, fmt.Errorf("exposure must be >= 0")
	}
	if overrides.Gamma < 0 {
		return RenderOverrides{}, fmt.Errorf("gamma must be >= 0")
	}
	if overrides.ToneMapping != "" && !isSupportedToneMapping(overrides.ToneMapping) {
		return RenderOverrides{}, fmt.Errorf("unsupported tone-mapping %q", overrides.ToneMapping)
	}
	if overrides.SpectrumMode != "" && !isSupportedSpectrumMode(overrides.SpectrumMode) {
		return RenderOverrides{}, fmt.Errorf("unsupported spectrum-mode %q", overrides.SpectrumMode)
	}
	if overrides.WavelengthSamples < 0 {
		return RenderOverrides{}, fmt.Errorf("wavelength-samples must be >= 0")
	}
	if overrides.ColorSpace != "" && !isSupportedColorSpace(overrides.ColorSpace) {
		return RenderOverrides{}, fmt.Errorf("unsupported working-space %q", overrides.ColorSpace)
	}

	return overrides, nil
}

func ResolveRenderConfig(script *parser.Script, overrides RenderOverrides) RenderConfig {
	config := RenderConfig{
		ScriptPath:        overrides.ScriptPath,
		Dimension:         3,
		CameraIndex:       0,
		ThreadNum:         runtime.NumCPU(),
		Samples:           defaultSamples,
		OutputImage:       defaultOutputImage,
		OutputFilm:        defaultOutputFilm,
		Exposure:          1,
		ToneMapping:       string(camera.ToneMappingLinear),
		Gamma:             1,
		SpectrumMode:      "hero_wavelength",
		WavelengthSamples: 1,
		ColorSpace:        "linear_srgb",
	}

	if script != nil {
		if script.Render.Dimension > 0 {
			config.Dimension = script.Render.Dimension
		}
		if script.Render.CameraIndexSet {
			config.CameraIndex = script.Render.CameraIndex
		}
		if script.Render.ThreadNum > 0 {
			config.ThreadNum = script.Render.ThreadNum
		}
		if script.Render.Width > 0 {
			config.Width = script.Render.Width
		}
		if script.Render.Height > 0 {
			config.Height = script.Render.Height
		}
		if script.Render.Samples > 0 {
			config.Samples = script.Render.Samples
		}
		if script.Render.OutputImage != "" {
			config.OutputImage = script.Render.OutputImage
		}
		if script.Render.OutputFilm != "" {
			config.OutputFilm = script.Render.OutputFilm
		}
		if script.Render.ResumeFilm != "" {
			config.ResumeFilm = script.Render.ResumeFilm
		}
		if script.Render.Exposure > 0 {
			config.Exposure = script.Render.Exposure
		}
		if script.Render.ToneMapping != "" {
			config.ToneMapping = script.Render.ToneMapping
		}
		if script.Render.Gamma > 0 {
			config.Gamma = script.Render.Gamma
		}
		if script.Render.SpectrumMode != "" {
			config.SpectrumMode = script.Render.SpectrumMode
		}
		if script.Render.WavelengthSamples > 0 {
			config.WavelengthSamples = script.Render.WavelengthSamples
		}
		if script.Render.ColorSpace != "" {
			config.ColorSpace = script.Render.ColorSpace
		} else if script.Render.FilmColorSpace != "" {
			config.ColorSpace = script.Render.FilmColorSpace
		}
	}

	if overrides.CameraIndex >= 0 {
		config.CameraIndex = overrides.CameraIndex
	}
	if overrides.Dimension > 0 {
		config.Dimension = overrides.Dimension
	}
	if overrides.ThreadNum > 0 {
		config.ThreadNum = overrides.ThreadNum
	}
	if overrides.Width > 0 {
		config.Width = overrides.Width
	}
	if overrides.Height > 0 {
		config.Height = overrides.Height
	}
	if overrides.Samples > 0 {
		config.Samples = overrides.Samples
	}
	if overrides.OutputImage != "" {
		config.OutputImage = overrides.OutputImage
	}
	if overrides.OutputFilm != "" {
		config.OutputFilm = overrides.OutputFilm
	}
	if overrides.ResumeFilm != "" {
		config.ResumeFilm = overrides.ResumeFilm
	}
	if overrides.Exposure > 0 {
		config.Exposure = overrides.Exposure
	}
	if overrides.ToneMapping != "" {
		config.ToneMapping = overrides.ToneMapping
	}
	if overrides.Gamma > 0 {
		config.Gamma = overrides.Gamma
	}
	if overrides.SpectrumMode != "" {
		config.SpectrumMode = overrides.SpectrumMode
	}
	if overrides.WavelengthSamples > 0 {
		config.WavelengthSamples = overrides.WavelengthSamples
	}
	if overrides.ColorSpace != "" {
		config.ColorSpace = overrides.ColorSpace
	}
	if config.SpectrumMode == "sampled" && config.WavelengthSamples <= 1 {
		config.WavelengthSamples = 4
	}

	return config
}

func ResolveRenderConfigs(script *parser.Script, overrides RenderOverrides) []RenderConfig {
	base := ResolveRenderConfig(script, RenderOverrides{
		ScriptPath:  overrides.ScriptPath,
		ScriptPaths: append([]string(nil), overrides.ScriptPaths...),
		CameraIndex: -1,
	})

	jobs := []RenderConfig{base}
	if script != nil && len(script.Renders) > 0 {
		jobs = make([]RenderConfig, 0, len(script.Renders))
		for _, render := range script.Renders {
			config := applyRenderScriptToConfig(base, render)
			jobs = append(jobs, applyRenderOverridesToConfig(config, overrides))
		}
		return jobs
	}

	jobs[0] = applyRenderOverridesToConfig(jobs[0], overrides)
	return jobs
}

func applyRenderScriptToConfig(config RenderConfig, render parser.RenderScript) RenderConfig {
	if render.Dimension > 0 {
		config.Dimension = render.Dimension
	}
	if render.CameraIndexSet {
		config.CameraIndex = render.CameraIndex
	}
	if render.ThreadNum > 0 {
		config.ThreadNum = render.ThreadNum
	}
	if render.Width > 0 {
		config.Width = render.Width
	}
	if render.Height > 0 {
		config.Height = render.Height
	}
	if render.Samples > 0 {
		config.Samples = render.Samples
	}
	if render.OutputImage != "" {
		config.OutputImage = render.OutputImage
	}
	if render.OutputFilm != "" {
		config.OutputFilm = render.OutputFilm
	}
	if render.ResumeFilm != "" {
		config.ResumeFilm = render.ResumeFilm
	}
	if render.Exposure > 0 {
		config.Exposure = render.Exposure
	}
	if render.ToneMapping != "" {
		config.ToneMapping = render.ToneMapping
	}
	if render.Gamma > 0 {
		config.Gamma = render.Gamma
	}
	if render.SpectrumMode != "" {
		config.SpectrumMode = render.SpectrumMode
	}
	if render.WavelengthSamples > 0 {
		config.WavelengthSamples = render.WavelengthSamples
	}
	if render.ColorSpace != "" {
		config.ColorSpace = render.ColorSpace
	} else if render.FilmColorSpace != "" {
		config.ColorSpace = render.FilmColorSpace
	}
	if config.SpectrumMode == "sampled" && config.WavelengthSamples <= 1 {
		config.WavelengthSamples = 4
	}
	return config
}

func applyRenderOverridesToConfig(config RenderConfig, overrides RenderOverrides) RenderConfig {
	if overrides.CameraIndex >= 0 {
		config.CameraIndex = overrides.CameraIndex
	}
	if overrides.Dimension > 0 {
		config.Dimension = overrides.Dimension
	}
	if overrides.ThreadNum > 0 {
		config.ThreadNum = overrides.ThreadNum
	}
	if overrides.Width > 0 {
		config.Width = overrides.Width
	}
	if overrides.Height > 0 {
		config.Height = overrides.Height
	}
	if overrides.Samples > 0 {
		config.Samples = overrides.Samples
	}
	if overrides.OutputImage != "" {
		config.OutputImage = overrides.OutputImage
	}
	if overrides.OutputFilm != "" {
		config.OutputFilm = overrides.OutputFilm
	}
	if overrides.ResumeFilm != "" {
		config.ResumeFilm = overrides.ResumeFilm
	}
	if overrides.Exposure > 0 {
		config.Exposure = overrides.Exposure
	}
	if overrides.ToneMapping != "" {
		config.ToneMapping = overrides.ToneMapping
	}
	if overrides.Gamma > 0 {
		config.Gamma = overrides.Gamma
	}
	if overrides.SpectrumMode != "" {
		config.SpectrumMode = overrides.SpectrumMode
	}
	if overrides.WavelengthSamples > 0 {
		config.WavelengthSamples = overrides.WavelengthSamples
	}
	if overrides.ColorSpace != "" {
		config.ColorSpace = overrides.ColorSpace
	}
	if config.SpectrumMode == "sampled" && config.WavelengthSamples <= 1 {
		config.WavelengthSamples = 4
	}
	return config
}

type stringListFlag []string

func (s *stringListFlag) String() string {
	return fmt.Sprint([]string(*s))
}

func (s *stringListFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func isSupportedSpectrumMode(value string) bool {
	switch value {
	case "rgb", "hero_wavelength", "sampled":
		return true
	default:
		return false
	}
}

func isSupportedColorSpace(value string) bool {
	switch camera.FilmColorSpace(value) {
	case camera.FilmColorSpaceLinearSRGB, camera.FilmColorSpaceACEScg, camera.FilmColorSpaceXYZ:
		return true
	default:
		return false
	}
}

func isSupportedToneMapping(value string) bool {
	switch camera.ToneMapping(value) {
	case camera.ToneMappingLinear, camera.ToneMappingReinhard, camera.ToneMappingACES:
		return true
	default:
		return false
	}
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
