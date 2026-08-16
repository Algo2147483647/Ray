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
		if err := runEndless(outputPath, script, config); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
		return 0
	}
	resumeFilm := resolveResumeFilm(script, config)
	if resumeFilm == "" {
		outputs := resolveRenderOutputs(script, config, "")
		outputIndex := 0
		code := controller.RunWithRenderSink(config.engineArgs(outputPath, "", 0), func(result controller.RenderResult) error {
			if outputIndex >= len(outputs) {
				return fmt.Errorf("engine produced more render jobs than Studio configured")
			}
			output := outputs[outputIndex]
			outputIndex++
			return writeStudioOutput(result.Film, output)
		})
		if code != 0 {
			return code
		}
		if outputIndex != len(outputs) {
			fmt.Printf("Error: engine produced %d render jobs; Studio configured %d\n", outputIndex, len(outputs))
			return 1
		}
		return 0
	}

	var renderedFilm *modelcamera.Film
	code := controller.RunWithRenderSink(config.engineArgs(outputPath, "", 0), func(result controller.RenderResult) error {
		renderedFilm = result.Film
		return nil
	})
	if code != 0 {
		return code
	}
	if renderedFilm == nil {
		fmt.Println("Error: engine produced no Film")
		return 1
	}

	outputFilm := resolveOutputFilm(script, config)
	fmt.Printf("Studio merging in-memory Film with %s -> %s\n", resumeFilm, outputFilm)
	mergedFilm, err := studiofilm.LoadFilm(resumeFilm)
	if err != nil {
		fmt.Printf("Error: load resume film %q: %v\n", resumeFilm, err)
		return 1
	}
	if err := studiofilm.MergeFilmsWithPixelWindows(mergedFilm, renderedFilm, resolvePixelWindows(script, config)); err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	if err := studiofilm.SaveFilm(mergedFilm, outputFilm); err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	if err := writeStudioImagesFromFilm(mergedFilm, resolveRenderOutputs(script, config, outputFilm)); err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	return 0
}

func runEndless(scriptPath string, script *schema.StudioScript, config studioConfig) error {
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

		fmt.Printf("Studio endless checkpoint %d: rendering %d samples\n", nextIteration, config.checkpointInterval)
		var renderedFilm *modelcamera.Film
		code := controller.RunWithRenderSink(config.engineArgs(scriptPath, "", config.checkpointInterval), func(result controller.RenderResult) error {
			renderedFilm = result.Film
			return nil
		})
		if code != 0 {
			return fmt.Errorf("engine render failed with exit code %d", code)
		}
		if renderedFilm == nil {
			return fmt.Errorf("engine produced no Film")
		}

		accumulatedFilm := renderedFilm
		if currentFilm != "" {
			baseFilm, err := studiofilm.LoadFilm(currentFilm)
			if err != nil {
				return err
			}
			if err := studiofilm.MergeFilmsWithPixelWindows(baseFilm, renderedFilm, resolvePixelWindows(script, config)); err != nil {
				return err
			}
			accumulatedFilm = baseFilm
		}
		if err := studiofilm.SaveFilm(accumulatedFilm, checkpointFilm); err != nil {
			return err
		}

		output := studioRenderOutputFromScript(baseStudioRender(script), config, checkpointFilm)
		output.ImagePath = checkpointImage
		if err := writeStudioImagesFromFilm(accumulatedFilm, []studioRenderOutput{output}); err != nil {
			return err
		}
		fmt.Printf("Studio saved checkpoint: %s and %s\n", checkpointFilm, checkpointImage)

		currentIteration = nextIteration
		currentFilm = checkpointFilm
	}
}

func checkpointPaths(dir string, iteration int64) (string, string) {
	stem := fmt.Sprintf("iteration-%012d", iteration)
	return filepath.Join(dir, stem+".bin"), filepath.Join(dir, stem+".png")
}

type studioRenderOutput struct {
	FilmPath  string
	ImagePath string
	Options   modelcamera.ImageOptions
}

func writeStudioOutput(film *modelcamera.Film, output studioRenderOutput) error {
	if err := studiofilm.SaveFilm(film, output.FilmPath); err != nil {
		return err
	}
	if output.ImagePath == "" {
		return nil
	}
	return studiofilm.SaveFilmImageFromFilm(film, output.ImagePath, output.Options)
}

func writeStudioImagesFromFilm(film *modelcamera.Film, outputs []studioRenderOutput) error {
	for _, output := range outputs {
		if output.ImagePath == "" {
			continue
		}
		if err := studiofilm.SaveFilmImageFromFilm(film, output.ImagePath, output.Options); err != nil {
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
		return script.Render.ResumeFilm
	}
	return ""
}

func resolveOutputFilm(script *schema.StudioScript, config studioConfig) string {
	outputs := resolveRenderOutputs(script, config, "")
	if len(outputs) > 0 {
		return outputs[0].FilmPath
	}
	return defaultEngineOutputFilm
}

func resolvePixelWindows(script *schema.StudioScript, config studioConfig) []modelcamera.PixelWindow {
	if config.provided["pixel-window"] {
		return studioPixelWindowsToEngine(config.pixelWindows)
	}
	render := baseStudioRender(script)
	return studioPixelWindowsToEngine(render.PixelWindows)
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
	base := baseStudioRender(script)

	if script != nil && len(script.Renders) > 0 {
		outputs := make([]studioRenderOutput, 0, len(script.Renders))
		for _, render := range script.Renders {
			outputs = append(outputs, studioRenderOutputFromScript(applyStudioRenderOverride(base, render), config, outputFilmOverride))
		}
		return outputs
	}
	return []studioRenderOutput{studioRenderOutputFromScript(base, config, outputFilmOverride)}
}

func baseStudioRender(script *schema.StudioScript) schema.StudioRenderScript {
	if script != nil {
		return script.Render
	}
	return schema.StudioRenderScript{}
}

func applyStudioRenderOverride(base, override schema.StudioRenderScript) schema.StudioRenderScript {
	result := base
	if override.OutputFilm != "" {
		result.OutputFilm = override.OutputFilm
	}
	if override.OutputImage != "" {
		result.OutputImage = override.OutputImage
	}
	if override.Exposure > 0 {
		result.Exposure = override.Exposure
	}
	if override.ToneMapping != "" {
		result.ToneMapping = override.ToneMapping
	}
	if override.Gamma > 0 {
		result.Gamma = override.Gamma
	}
	return result
}

func studioRenderOutputFromScript(render schema.StudioRenderScript, config studioConfig, outputFilmOverride string) studioRenderOutput {
	filmPath := render.OutputFilm
	if filmPath == "" {
		filmPath = defaultEngineOutputFilm
	}
	if config.provided["output-film"] {
		filmPath = config.outputFilm
	}
	if outputFilmOverride != "" {
		filmPath = outputFilmOverride
	}

	imagePath := render.OutputImage
	if imagePath == "" {
		imagePath = defaultEngineOutputImage
	}
	if config.provided["output-image"] {
		imagePath = config.outputImage
	}

	options := modelcamera.ImageOptions{
		Exposure:    1,
		ToneMapping: modelcamera.ToneMappingLinear,
		Gamma:       1,
	}
	if render.Exposure > 0 {
		options.Exposure = render.Exposure
	}
	if render.ToneMapping != "" {
		options.ToneMapping = modelcamera.ToneMapping(render.ToneMapping)
	}
	if render.Gamma > 0 {
		options.Gamma = render.Gamma
	}
	if config.provided["exposure"] {
		options.Exposure = config.exposure
	}
	if config.provided["tone-mapping"] {
		options.ToneMapping = modelcamera.ToneMapping(config.toneMapping)
	}
	if config.provided["gamma"] {
		options.Gamma = config.gamma
	}

	return studioRenderOutput{
		FilmPath:  filmPath,
		ImagePath: imagePath,
		Options:   options,
	}
}
