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
		code := controller.Run(config.engineArgs(outputPath, "", 0))
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

	code := controller.Run(config.engineArgs(outputPath, tempFilmPath, 0))
	if code != 0 {
		return code
	}

	outputFilm := resolveOutputFilm(script, config)
	fmt.Printf("Studio merging film: %s + %s -> %s\n", resumeFilm, tempFilmPath, outputFilm)
	if err := studiofilm.MergeFilmFiles(resumeFilm, tempFilmPath, outputFilm); err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}

	if err := writeStudioImages(resolveRenderOutputs(script, config, outputFilm)); err != nil {
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
		tempFilmPath, err := createTempFilmPath()
		if err != nil {
			return err
		}

		fmt.Printf("Studio endless checkpoint %d: rendering %d samples\n", nextIteration, config.checkpointInterval)
		code := controller.Run(config.engineArgs(scriptPath, tempFilmPath, config.checkpointInterval))
		if code != 0 {
			os.Remove(tempFilmPath)
			return fmt.Errorf("engine render failed with exit code %d", code)
		}

		if currentFilm == "" {
			err = studiofilm.CopyFilmFile(tempFilmPath, checkpointFilm)
		} else {
			err = studiofilm.MergeFilmFiles(currentFilm, tempFilmPath, checkpointFilm)
		}
		os.Remove(tempFilmPath)
		if err != nil {
			return err
		}

		output := studioRenderOutputFromScript(baseStudioRender(script), config, checkpointFilm)
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
	Options   modelcamera.ImageOptions
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
