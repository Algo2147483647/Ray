package main

import (
	"fmt"
	"os"
)

func main() {
	overrides, err := ParseRenderOverrides(os.Args[1:])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if overrides.ScriptPath == defaultScriptPath {
		fmt.Printf("Using default script: %s\n", overrides.ScriptPath)
	}

	h := NewHandler().
		LoadScript(overrides.ScriptPath).
		ConfigureRender(overrides).
		Render().
		SaveOutputs()

	if h.err != nil {
		fmt.Printf("Error: %v\n", h.err)
		os.Exit(1)
	}

	fmt.Println("Ray tracing completed successfully")
}
