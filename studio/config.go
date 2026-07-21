package main

import (
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/Algo2147483647/ray/studio/schema"
)

const defaultScriptPath = "../examples/scenes/default.json"

type studioConfig struct {
	scriptPaths       []string
	provided          map[string]bool
	dimension         int
	cameraIndex       int
	threadNum         int
	width             int
	height            int
	samples           int64
	outputImage       string
	outputFilm        string
	resumeFilm        string
	exposure          float64
	toneMapping       string
	gamma             float64
	spectrumMode      string
	wavelengthSamples int
	colorSpace        string
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
		cameraIndex: -1,
		provided:    map[string]bool{},
	}
	scriptPaths := stringListFlag{}

	flagSet := flag.NewFlagSet("ray", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.Var(&scriptPaths, "script", "path to a scene script; repeat to merge multiple scripts")
	flagSet.IntVar(&config.dimension, "dimension", 0, "scene dimension")
	flagSet.IntVar(&config.cameraIndex, "camera-index", -1, "camera index to render")
	flagSet.IntVar(&config.threadNum, "threads", 0, "worker thread count")
	flagSet.IntVar(&config.width, "width", 0, "output width")
	flagSet.IntVar(&config.height, "height", 0, "output height")
	flagSet.Int64Var(&config.samples, "samples", 0, "samples per pixel")
	flagSet.StringVar(&config.outputImage, "output-image", "", "output image path")
	flagSet.StringVar(&config.outputFilm, "output-film", "", "output film path")
	flagSet.StringVar(&config.resumeFilm, "resume-film", "", "existing film path to merge before saving outputs")
	flagSet.Float64Var(&config.exposure, "exposure", 0, "output exposure multiplier")
	flagSet.StringVar(&config.toneMapping, "tone-mapping", "", "output tone mapping: linear, reinhard, aces")
	flagSet.Float64Var(&config.gamma, "gamma", 0, "output gamma, for example 2.2")
	flagSet.StringVar(&config.spectrumMode, "spectrum-mode", "", "spectrum mode: rgb, hero_wavelength, sampled")
	flagSet.IntVar(&config.wavelengthSamples, "wavelength-samples", 0, "wavelength samples per camera sample in sampled mode")
	flagSet.StringVar(&config.colorSpace, "working-space", "", "film working space: linear_srgb, acescg, xyz")

	if err := flagSet.Parse(args); err != nil {
		return studioConfig{}, err
	}
	flagSet.Visit(func(f *flag.Flag) {
		config.provided[f.Name] = true
	})

	config.scriptPaths = append(config.scriptPaths, scriptPaths...)
	if len(config.scriptPaths) == 0 && len(flagSet.Args()) > 0 {
		config.scriptPaths = append(config.scriptPaths, flagSet.Args()...)
	}
	if len(config.scriptPaths) == 0 {
		config.scriptPaths = []string{defaultScriptPath}
	}

	if config.cameraIndex < -1 {
		return studioConfig{}, fmt.Errorf("camera-index must be >= -1")
	}
	if config.dimension < 0 || config.dimension == 1 {
		return studioConfig{}, fmt.Errorf("dimension must be 0 or >= 2")
	}
	if config.threadNum < 0 {
		return studioConfig{}, fmt.Errorf("threads must be >= 0")
	}
	if config.width < 0 || config.height < 0 {
		return studioConfig{}, fmt.Errorf("width and height must be >= 0")
	}
	if config.samples < 0 {
		return studioConfig{}, fmt.Errorf("samples must be >= 0")
	}
	if config.exposure < 0 {
		return studioConfig{}, fmt.Errorf("exposure must be >= 0")
	}
	if config.gamma < 0 {
		return studioConfig{}, fmt.Errorf("gamma must be >= 0")
	}
	if config.wavelengthSamples < 0 {
		return studioConfig{}, fmt.Errorf("wavelength-samples must be >= 0")
	}
	return config, nil
}

func (c studioConfig) engineArgs(scriptPath string) []string {
	args := []string{"--script", scriptPath}
	if c.provided["dimension"] {
		args = append(args, "--dimension", strconv.Itoa(c.dimension))
	}
	if c.provided["camera-index"] {
		args = append(args, "--camera-index", strconv.Itoa(c.cameraIndex))
	}
	if c.provided["threads"] {
		args = append(args, "--threads", strconv.Itoa(c.threadNum))
	}
	if c.provided["width"] {
		args = append(args, "--width", strconv.Itoa(c.width))
	}
	if c.provided["height"] {
		args = append(args, "--height", strconv.Itoa(c.height))
	}
	if c.provided["samples"] {
		args = append(args, "--samples", strconv.FormatInt(c.samples, 10))
	}
	if c.provided["output-image"] {
		args = append(args, "--output-image", c.outputImage)
	}
	if c.provided["output-film"] {
		args = append(args, "--output-film", c.outputFilm)
	}
	if c.provided["resume-film"] {
		args = append(args, "--resume-film", c.resumeFilm)
	}
	if c.provided["exposure"] {
		args = append(args, "--exposure", strconv.FormatFloat(c.exposure, 'g', -1, 64))
	}
	if c.provided["tone-mapping"] {
		args = append(args, "--tone-mapping", c.toneMapping)
	}
	if c.provided["gamma"] {
		args = append(args, "--gamma", strconv.FormatFloat(c.gamma, 'g', -1, 64))
	}
	if c.provided["spectrum-mode"] {
		args = append(args, "--spectrum-mode", c.spectrumMode)
	}
	if c.provided["wavelength-samples"] {
		args = append(args, "--wavelength-samples", strconv.Itoa(c.wavelengthSamples))
	}
	if c.provided["working-space"] {
		args = append(args, "--working-space", c.colorSpace)
	}
	return args
}

func resolveDimension(script *schema.StudioScript, config studioConfig) int {
	if config.dimension > 0 {
		return config.dimension
	}
	if script != nil && script.Render.Dimension > 0 {
		return script.Render.Dimension
	}
	return 3
}
