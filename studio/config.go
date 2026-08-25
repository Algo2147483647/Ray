package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Algo2147483647/ray/engine/ray_tracing"
	"github.com/Algo2147483647/ray/studio/schema"
)

const (
	defaultScriptPath  = "../examples/scenes/default.json"
	defaultOutputImage = "../../outputs/output.png"
	defaultOutputFilm  = "../../outputs/img.bin"
	defaultFilmWidth   = 400
	defaultFilmHeight  = 400
)

type studioConfig struct {
	scriptPaths        []string
	inputFilm          string
	provided           map[string]bool
	integrator         string
	cameraID           string
	dimension          int
	threadNum          int
	width              int
	height             int
	widths             []int
	samples            int64
	outputImage        string
	outputFilm         string
	resumeFilm         string
	endless            bool
	checkpointInterval int64
	checkpointDir      string
	startIteration     int64
	exposure           float64
	toneMapping        string
	tanhOmega          float64
	gamma              float64
	spectrumMode       string
	wavelengthSamples  int
	colorSpace         string
	pixelWindows       []schema.PixelWindowScript
}

type stringListFlag []string

func (s *stringListFlag) String() string {
	return fmt.Sprint([]string(*s))
}

func (s *stringListFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func parseStudioConfig(args []string) (studioConfig, error) {
	config := studioConfig{
		provided: map[string]bool{},
	}
	scriptPaths := stringListFlag{}
	pixelWindowFlags := stringListFlag{}

	flagSet := flag.NewFlagSet("ray", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.Var(&scriptPaths, "script", "path to a scene script; repeat to merge multiple scripts")
	flagSet.StringVar(&config.inputFilm, "input-film", "", "existing binary Film to convert to PNG without rendering")
	flagSet.Var(&pixelWindowFlags, "pixel-window", "pixel render window, for example 100:150,600:650; repeat for multiple windows")
	flagSet.StringVar(&config.integrator, "integrator", "", "light transport integrator: path, bdpt, light_tracing")
	flagSet.StringVar(&config.cameraID, "camera-id", "", "canonical Engine camera ID override")
	flagSet.IntVar(&config.dimension, "dimension", 0, "scene dimension")
	flagSet.IntVar(&config.threadNum, "threads", 0, "worker thread count")
	flagSet.IntVar(&config.width, "width", 0, "output width")
	flagSet.IntVar(&config.height, "height", 0, "output height")
	flagSet.Func("widths", "film dimensions, for example 1920,1080", func(value string) error {
		widths, err := parseStudioWidths(value)
		if err == nil {
			config.widths = widths
		}
		return err
	})
	flagSet.Int64Var(&config.samples, "samples", 0, "samples per pixel")
	flagSet.StringVar(&config.outputImage, "output-image", "", "output image path")
	flagSet.StringVar(&config.outputFilm, "output-film", "", "output film path")
	flagSet.StringVar(&config.resumeFilm, "resume-film", "", "existing film path to merge before saving outputs")
	flagSet.BoolVar(&config.endless, "endless", false, "render forever and save periodic film and image checkpoints")
	flagSet.Int64Var(&config.checkpointInterval, "checkpoint-interval", 0, "samples to render between endless checkpoints")
	flagSet.StringVar(&config.checkpointDir, "checkpoint-dir", "", "directory for endless film and image checkpoints")
	flagSet.Int64Var(&config.startIteration, "start-iteration", 0, "sample iteration count represented by resume-film")
	flagSet.Float64Var(&config.exposure, "exposure", 0, "output exposure multiplier")
	flagSet.StringVar(&config.toneMapping, "tone-mapping", "", "output tone mapping: linear, reinhard, aces, spectral_tanh")
	flagSet.Float64Var(&config.tanhOmega, "tanh-omega", 0, "spectral_tanh slope coefficient (default 1)")
	flagSet.Float64Var(&config.gamma, "gamma", 0, "output gamma, for example 2.2")
	flagSet.StringVar(&config.spectrumMode, "spectrum-mode", "", "spectrum mode: hero_wavelength, sampled")
	flagSet.IntVar(&config.wavelengthSamples, "wavelength-samples", 0, "wavelength samples per camera sample in sampled mode")
	flagSet.StringVar(&config.colorSpace, "color-space", "", "Studio output color space: linear_srgb, acescg, xyz")

	if err := flagSet.Parse(args); err != nil {
		return studioConfig{}, err
	}
	flagSet.Visit(func(f *flag.Flag) {
		config.provided[f.Name] = true
	})
	if len(pixelWindowFlags) > 0 {
		windows, err := parseStudioPixelWindowFlags(pixelWindowFlags)
		if err != nil {
			return studioConfig{}, err
		}
		config.pixelWindows = windows
	}

	config.scriptPaths = append(config.scriptPaths, scriptPaths...)
	config.scriptPaths = append(config.scriptPaths, flagSet.Args()...)

	if config.dimension < 0 || config.dimension == 1 {
		return studioConfig{}, fmt.Errorf("dimension must be 0 or >= 2")
	}
	if config.threadNum < 0 {
		return studioConfig{}, fmt.Errorf("threads must be >= 0")
	}
	if config.width < 0 || config.height < 0 {
		return studioConfig{}, fmt.Errorf("width and height must be >= 0")
	}
	if len(config.widths) > 0 && (config.provided["width"] || config.provided["height"]) {
		return studioConfig{}, fmt.Errorf("widths cannot be combined with width or height")
	}
	if config.integrator != "" {
		if _, err := ray_tracing.ParseIntegratorKind(config.integrator); err != nil {
			return studioConfig{}, err
		}
	}
	if config.samples < 0 {
		return studioConfig{}, fmt.Errorf("samples must be >= 0")
	}
	if config.checkpointInterval < 0 {
		return studioConfig{}, fmt.Errorf("checkpoint-interval must be >= 0")
	}
	if config.startIteration < 0 {
		return studioConfig{}, fmt.Errorf("start-iteration must be >= 0")
	}
	if config.endless {
		if config.checkpointInterval <= 0 {
			return studioConfig{}, fmt.Errorf("checkpoint-interval must be > 0 when endless mode is enabled")
		}
		if config.checkpointDir == "" {
			return studioConfig{}, fmt.Errorf("checkpoint-dir is required when endless mode is enabled")
		}
		if config.startIteration > 0 && config.resumeFilm == "" {
			return studioConfig{}, fmt.Errorf("resume-film is required when start-iteration is > 0")
		}
	}
	if config.exposure < 0 {
		return studioConfig{}, fmt.Errorf("exposure must be >= 0")
	}
	if config.gamma < 0 {
		return studioConfig{}, fmt.Errorf("gamma must be >= 0")
	}
	if config.provided["tanh-omega"] && (config.tanhOmega <= 0 || math.IsNaN(config.tanhOmega) || math.IsInf(config.tanhOmega, 0)) {
		return studioConfig{}, fmt.Errorf("tanh-omega must be finite and > 0")
	}
	if config.wavelengthSamples < 0 {
		return studioConfig{}, fmt.Errorf("wavelength-samples must be >= 0")
	}
	if config.spectrumMode != "" && config.spectrumMode != "hero_wavelength" && config.spectrumMode != "sampled" {
		return studioConfig{}, fmt.Errorf("spectrum-mode must be hero_wavelength or sampled")
	}
	if config.toneMapping != "" && config.toneMapping != "linear" && config.toneMapping != "reinhard" && config.toneMapping != "aces" && config.toneMapping != "spectral_tanh" {
		return studioConfig{}, fmt.Errorf("tone-mapping must be linear, reinhard, aces, or spectral_tanh")
	}
	if config.colorSpace != "" && config.colorSpace != "linear_srgb" && config.colorSpace != "acescg" && config.colorSpace != "xyz" {
		return studioConfig{}, fmt.Errorf("color-space must be linear_srgb, acescg, or xyz")
	}
	if config.provided["input-film"] {
		if config.inputFilm == "" {
			return studioConfig{}, fmt.Errorf("input-film cannot be empty")
		}
		if err := validateFilmConversionConfig(config); err != nil {
			return studioConfig{}, err
		}
		return config, nil
	}
	if len(config.scriptPaths) == 0 {
		config.scriptPaths = []string{defaultScriptPath}
	}
	return config, nil
}

func validateFilmConversionConfig(config studioConfig) error {
	if len(config.scriptPaths) > 0 {
		return fmt.Errorf("input-film cannot be combined with scene scripts")
	}
	if config.provided["output-image"] && config.outputImage == "" {
		return fmt.Errorf("output-image cannot be empty when input-film is used")
	}
	for _, name := range []string{
		"integrator", "camera-id", "dimension", "threads", "width", "height", "widths",
		"samples", "output-film", "resume-film", "endless", "checkpoint-interval",
		"checkpoint-dir", "start-iteration", "spectrum-mode", "wavelength-samples", "pixel-window",
	} {
		if config.provided[name] {
			return fmt.Errorf("input-film cannot be combined with --%s", name)
		}
	}
	return nil
}

func (c studioConfig) filmConversionImagePath() string {
	if c.provided["output-image"] {
		return c.outputImage
	}
	ext := filepath.Ext(c.inputFilm)
	if strings.EqualFold(ext, ".bin") {
		return strings.TrimSuffix(c.inputFilm, ext) + ".png"
	}
	return c.inputFilm + ".png"
}

func (c studioConfig) engineArgs(scriptPath string) []string {
	return []string{"--script", scriptPath}
}

func (c studioConfig) applyEngineOverrides(script *schema.IntermediateScript, outputFilmOverride string, samplesOverride int64) {
	if script == nil {
		return
	}
	for _, render := range script.Renders {
		if c.provided["integrator"] {
			render["integrator"] = c.integrator
		}
		if c.provided["camera-id"] {
			render["camera_id"] = c.cameraID
		}
		if c.provided["threads"] {
			render["thread_num"] = c.threadNum
		}
		if samplesOverride > 0 {
			render["samples"] = samplesOverride
		} else if c.provided["samples"] {
			render["samples"] = c.samples
		}
		if c.provided["spectrum-mode"] {
			render["spectrum_mode"] = c.spectrumMode
		}
		if c.provided["wavelength-samples"] {
			render["wavelength_samples"] = c.wavelengthSamples
		}
		normalizeIntermediateRender(render)
	}

	widths := c.filmShapeOverride()
	for i := range script.Cameras {
		film := &script.Cameras[i].Film
		if len(widths) > 0 {
			film.Shape = append([]int(nil), widths...)
		}
		if outputFilmOverride != "" {
			film.OutputFilm = outputFilmOverride
		} else if c.provided["output-film"] {
			film.OutputFilm = c.outputFilm
		}
		if c.provided["pixel-window"] {
			film.PixelWindows = cloneStudioPixelWindows(c.pixelWindows)
		}
	}
}

func normalizeIntermediateRender(render map[string]interface{}) {
	if render["spectrum_mode"] != "sampled" {
		return
	}
}

func (c studioConfig) filmShapeOverride() []int {
	if len(c.widths) > 0 {
		return append([]int(nil), c.widths...)
	}
	if !c.provided["width"] && !c.provided["height"] {
		return nil
	}
	width := c.width
	if width <= 0 {
		width = defaultFilmWidth
	}
	height := c.height
	if height <= 0 {
		height = defaultFilmHeight
	}
	return []int{width, height}
}

func parseStudioWidths(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	widths := make([]int, len(parts))
	for i, part := range parts {
		width, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || width <= 0 {
			return nil, fmt.Errorf("widths[%d] must be a positive integer", i)
		}
		widths[i] = width
	}
	return widths, nil
}

func cloneStudioPixelWindows(windows []schema.PixelWindowScript) []schema.PixelWindowScript {
	cloned := make([]schema.PixelWindowScript, len(windows))
	for i, window := range windows {
		cloned[i] = schema.PixelWindowScript{
			Min: append([]int(nil), window.Min...),
			Max: append([]int(nil), window.Max...),
		}
	}
	return cloned
}

func parseStudioPixelWindowFlags(values []string) ([]schema.PixelWindowScript, error) {
	windows := make([]schema.PixelWindowScript, 0, len(values))
	for _, value := range values {
		window, err := parseStudioPixelWindowFlag(value)
		if err != nil {
			return nil, err
		}
		windows = append(windows, window)
	}
	return windows, nil
}

func parseStudioPixelWindowFlag(value string) (schema.PixelWindowScript, error) {
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return schema.PixelWindowScript{}, fmt.Errorf("pixel-window is empty")
	}

	min := make([]int, len(parts))
	max := make([]int, len(parts))
	for dim, part := range parts {
		lo, hi, err := parseStudioPixelWindowAxis(part)
		if err != nil {
			return schema.PixelWindowScript{}, fmt.Errorf("pixel-window dimension %d: %w", dim, err)
		}
		min[dim] = lo
		max[dim] = hi
	}
	return schema.PixelWindowScript{Min: min, Max: max}, nil
}

func parseStudioPixelWindowAxis(value string) (int, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, 0, fmt.Errorf("range is empty")
	}

	separator := ":"
	if strings.Contains(value, ":") {
		separator = ":"
	} else if strings.Contains(value, "-") {
		separator = "-"
	} else {
		return 0, 0, fmt.Errorf("range must use ':' or '-'")
	}

	parts := strings.Split(value, separator)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("range must contain exactly one separator")
	}

	lo, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid min %q", strings.TrimSpace(parts[0]))
	}
	hi, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid max %q", strings.TrimSpace(parts[1]))
	}
	if lo < 0 || hi < 0 {
		return 0, 0, fmt.Errorf("bounds must be non-negative")
	}
	if lo >= hi {
		return 0, 0, fmt.Errorf("min must be less than max")
	}
	return lo, hi, nil
}

func resolveDimension(script *schema.StudioScript, config studioConfig) (int, error) {
	if config.dimension > 0 {
		return config.dimension, nil
	}
	if script == nil {
		return 3, nil
	}
	dimension := script.Dimension
	if dimension <= 0 {
		dimension = 3
	}
	return dimension, nil
}
