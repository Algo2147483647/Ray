package main

import (
	"fmt"
	"os"
)

func main() {
	scriptPath := "C:/Algo/Projects/Ray/src-golang/test.json"

	if len(os.Args) > 1 {
		scriptPath = os.Args[1]
	} else {
		fmt.Printf("Using default script: %s\n", scriptPath)
	}

	h := NewHandler().
		LoadScript(scriptPath).
		BuildCamera(100, 100, 100).
		BuildFilm(100, 100, 100).
		Render(20).
		//MergeFilm("img.bin").
		SaveFilm("img.bin").
		SaveImg("output.png").
		SaveDebugInfo("debug_traces.json")

	if h.err != nil {
		fmt.Printf("Error: %v\n", h.err)
		os.Exit(1)
	}

	fmt.Println("Ray tracing completed successfully")
}
