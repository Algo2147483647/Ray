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

	width := 100

	h := NewHandler().
		LoadScript(scriptPath).
		BuildCamera(width, width, width).
		BuildFilm(width, width, width).
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
