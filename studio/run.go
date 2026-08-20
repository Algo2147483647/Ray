package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Algo2147483647/ray/engine/controller"
	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/studio/adapt"
	studiofilm "github.com/Algo2147483647/ray/studio/film"
	"github.com/Algo2147483647/ray/studio/schema"
	"github.com/Algo2147483647/ray/studio/storage"
)

func run(args []string) int {
	config, err := parseStudioConfig(args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	if config.inputFilm != "" {
		if err := runFilmConversion(config); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
		return 0
	}

	script, err := storage.ReadStudioScriptFiles(config.scriptPaths)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}

	dimension := resolveDimension(script, config)
	adapted, err := adapt.AdaptScript(script, config.scriptPaths, dimension)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	config.applyEngineOverrides(adapted, "", 0)

	outputPath, err := storage.WriteIntermediateScript(adapted, config.scriptPaths)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	fmt.Printf("Studio wrote intermediate script: %s\n", outputPath)

	root, err := storage.RepoRoot()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	if err := os.Chdir(filepath.Join(root, "engine")); err != nil {
		fmt.Printf("Error: enter engine directory: %v\n", err)
		return 1
	}
	if config.endless {
		if err := runEndless(adapted, script, config); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
		return 0
	}
	resumeFilm := resolveResumeFilm(script, config)
	if resumeFilm == "" {
		code := controller.Run(config.engineArgs(outputPath))
		if code != 0 {
			return code
		}
		if err := writeStudioImages(resolveRenderOutputs(script, config, "")); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
		return 0
	}

	tempFilmPath, err := createTempFilmPath()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	defer os.Remove(tempFilmPath)

	config.applyEngineOverrides(adapted, tempFilmPath, 0)
	outputPath, err = storage.WriteIntermediateScript(adapted, config.scriptPaths)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	code := controller.Run(config.engineArgs(outputPath))
	if code != 0 {
		return code
	}

	outputFilm := resolveOutputFilm(script, config)
	fmt.Printf("Studio merging film: %s + %s -> %s\n", resumeFilm, tempFilmPath, outputFilm)
	if err := studiofilm.MergeFilmFilesWithPixelWindows(resumeFilm, tempFilmPath, outputFilm, resolvePixelWindows(script, config)); err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	if err := writeStudioImages(resolveRenderOutputs(script, config, outputFilm)); err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	return 0
}

func runFilmConversion(config studioConfig) error {
	output := studioRenderOutputFromFilm(schema.StudioFilmScript{}, config, config.inputFilm)
	output.ImagePath = config.filmConversionImagePath()
	if err := writeStudioImages([]studioRenderOutput{output}); err != nil {
		return fmt.Errorf("convert Film %q to PNG %q: %w", config.inputFilm, output.ImagePath, err)
	}
	fmt.Printf("Studio converted Film to PNG: %s -> %s\n", config.inputFilm, output.ImagePath)
	return nil
}

func runEndless(adapted *schema.IntermediateScript, script *schema.StudioScript, config studioConfig) error {
	if script != nil && len(script.Renders) > 0 {
		return fmt.Errorf("endless mode supports a single render; remove renders or run them separately")
	}
	if err := os.MkdirAll(config.checkpointDir, 0o755); err != nil {
		return fmt.Errorf("create checkpoint directory %q: %w", config.checkpointDir, err)
	}

	currentIteration := config.startIteration
	currentFilm := resolveResumeFilm(script, config)
	fmt.Printf("Studio endless mode: +%d samples per checkpoint -> %s\n", config.checkpointInterval, config.checkpointDir)
	if currentFilm != "" {
		fmt.Printf("Studio resuming endless mode from iteration %d: %s\n", currentIteration, currentFilm)
	}

	for {
		nextIteration := currentIteration + config.checkpointInterval
		checkpointFilm, checkpointImage := checkpointPaths(config.checkpointDir, nextIteration)
		tempFilmPath, err := createTempFilmPath()
		if err != nil {
			return err
		}

		fmt.Printf("Studio endless checkpoint %d: rendering %d samples\n", nextIteration, config.checkpointInterval)
		config.applyEngineOverrides(adapted, tempFilmPath, config.checkpointInterval)
		scriptPath, err := storage.WriteIntermediateScript(adapted, config.scriptPaths)
		if err != nil {
			os.Remove(tempFilmPath)
			return err
		}
		code := controller.Run(config.engineArgs(scriptPath))
		if code != 0 {
			os.Remove(tempFilmPath)
			return fmt.Errorf("engine render failed with exit code %d", code)
		}

		if currentFilm == "" {
			err = studiofilm.CopyFilmFile(tempFilmPath, checkpointFilm)
		} else {
			err = studiofilm.MergeFilmFilesWithPixelWindows(currentFilm, tempFilmPath, checkpointFilm, resolvePixelWindows(script, config))
		}
		os.Remove(tempFilmPath)
		if err != nil {
			return err
		}

		output := studioRenderOutputFromFilm(resolveFilm(script, script.Render), config, checkpointFilm)
		output.ImagePath = checkpointImage
		if err := writeStudioImages([]studioRenderOutput{output}); err != nil {
			return err
		}
		fmt.Printf("Studio saved checkpoint: %s and %s\n", checkpointFilm, checkpointImage)

		currentIteration = nextIteration
		currentFilm = checkpointFilm
	}
}

func createTempFilmPath() (string, error) {
	tempFilm, err := os.CreateTemp("", "ray-studio-render-*.bin")
	if err != nil {
		return "", fmt.Errorf("create temporary film: %w", err)
	}
	tempFilmPath := tempFilm.Name()
	if err := tempFilm.Close(); err != nil {
		return "", fmt.Errorf("close temporary film: %w", err)
	}
	return tempFilmPath, nil
}

func checkpointPaths(dir string, iteration int64) (string, string) {
	stem := fmt.Sprintf("iteration-%012d", iteration)
	return filepath.Join(dir, stem+".bin"), filepath.Join(dir, stem+".png")
}

type studioRenderOutput struct {
	FilmPath  string
	ImagePath string
	Options   studiofilm.ImageOptions
}

func writeStudioImages(outputs []studioRenderOutput) error {
	for _, output := range outputs {
		if output.ImagePath == "" {
			continue
		}
		if err := studiofilm.SaveFilmImage(output.FilmPath, output.ImagePath, output.Options); err != nil {
			return err
		}
	}
	return nil
}

func resolveResumeFilm(script *schema.StudioScript, config studioConfig) string {
	if config.provided["resume-film"] {
		return config.resumeFilm
	}
	if script != nil {
		return resolveFilm(script, script.Render).ResumeFilm
	}
	return ""
}

func resolveOutputFilm(script *schema.StudioScript, config studioConfig) string {
	outputs := resolveRenderOutputs(script, config, "")
	if len(outputs) > 0 {
		return outputs[0].FilmPath
	}
	return defaultOutputFilm
}

func resolvePixelWindows(script *schema.StudioScript, config studioConfig) []modelcamera.PixelWindow {
	if config.provided["pixel-window"] {
		return studioPixelWindowsToEngine(config.pixelWindows)
	}
	return studioPixelWindowsToEngine(resolveFilm(script, script.Render).PixelWindows)
}

func studioPixelWindowsToEngine(windows []schema.PixelWindowScript) []modelcamera.PixelWindow {
	if len(windows) == 0 {
		return nil
	}
	result := make([]modelcamera.PixelWindow, len(windows))
	for i, window := range windows {
		result[i] = modelcamera.PixelWindow{
			Min: append([]int(nil), window.Min...),
			Max: append([]int(nil), window.Max...),
		}
	}
	return result
}

func resolveRenderOutputs(script *schema.StudioScript, config studioConfig, outputFilmOverride string) []studioRenderOutput {
	if script != nil && len(script.Renders) > 0 {
		outputs := make([]studioRenderOutput, 0, len(script.Renders))
		for _, render := range script.Renders {
			if render.FilmID == "" {
				render.FilmID = script.Render.FilmID
			}
			outputs = append(outputs, studioRenderOutputFromFilm(resolveFilm(script, render), config, outputFilmOverride))
		}
		return outputs
	}
	if script == nil {
		return nil
	}
	return []studioRenderOutput{studioRenderOutputFromFilm(resolveFilm(script, script.Render), config, outputFilmOverride)}
}

func resolveFilm(script *schema.StudioScript, render schema.StudioRenderScript) schema.StudioFilmScript {
	if script == nil {
		return schema.StudioFilmScript{}
	}
	for _, film := range script.Films {
		if film.ID == render.FilmID {
			return film
		}
	}
	return schema.StudioFilmScript{}
}

func studioRenderOutputFromFilm(film schema.StudioFilmScript, config studioConfig, outputFilmOverride string) studioRenderOutput {
	filmPath := film.OutputFilm
	if filmPath == "" {
		filmPath = defaultOutputFilm
	}
	if config.provided["output-film"] {
		filmPath = config.outputFilm
	}
	if outputFilmOverride != "" {
		filmPath = outputFilmOverride
	}

	imagePath := film.OutputImage
	if imagePath == "" {
		imagePath = defaultOutputImage
	}
	if config.provided["output-image"] {
		imagePath = config.outputImage
	}

	options := studiofilm.ImageOptions{
		Exposure:    1,
		ToneMapping: studiofilm.ToneMappingLinear,
		Gamma:       1,
		ColorSpace:  studiofilm.ColorSpaceLinearSRGB,
	}
	if film.Exposure > 0 {
		options.Exposure = film.Exposure
	}
	if film.ToneMapping != "" {
		options.ToneMapping = studiofilm.ToneMapping(film.ToneMapping)
	}
	if film.Gamma > 0 {
		options.Gamma = film.Gamma
	}
	if film.ColorSpace != "" {
		options.ColorSpace = studiofilm.ColorSpace(film.ColorSpace)
	}
	if config.provided["exposure"] {
		options.Exposure = config.exposure
	}
	if config.provided["tone-mapping"] {
		options.ToneMapping = studiofilm.ToneMapping(config.toneMapping)
	}
	if config.provided["gamma"] {
		options.Gamma = config.gamma
	}
	if config.provided["color-space"] {
		options.ColorSpace = studiofilm.ColorSpace(config.colorSpace)
	}

	return studioRenderOutput{
		FilmPath:  filmPath,
		ImagePath: imagePath,
		Options:   options,
	}
}
