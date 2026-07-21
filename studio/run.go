package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Algo2147483647/ray/engine/controller"
	"github.com/Algo2147483647/ray/studio/adapt"
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
	return controller.Run(config.engineArgs(outputPath))
}
