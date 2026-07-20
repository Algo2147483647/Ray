package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Algo2147483647/ray/engine/controller"
)

func run(args []string) int {
	config, err := parseStudioConfig(args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}

	script, err := readStudioScriptFiles(config.scriptPaths)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}

	dimension := resolveDimension(script, config)
	adapted, err := adaptScript(script, config.scriptPaths, dimension)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}

	outputPath, err := writeIntermediateScript(adapted, config.scriptPaths)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	fmt.Printf("Studio wrote intermediate script: %s\n", outputPath)

	root, err := repoRoot()
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
