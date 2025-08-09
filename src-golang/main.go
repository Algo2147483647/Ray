package main

import (
	"fmt"
	"os"
)

func main() {
	defaultFile := "C:/Algo/Projects/Ray/src-golang/test.json"
	scriptPath := defaultFile

	if len(os.Args) > 1 {
		scriptPath = os.Args[1]
	} else {
		fmt.Printf("Using default script: %s\n", defaultFile)
	}

	h := NewHandler(scriptPath).
		PreCheck().
		LoadScript().
		BuildCamera().
		Render().
		BuildResult().
		SaveResult()

	if h.err != nil {
		fmt.Printf("Error: %v\n", h.err)
		os.Exit(1)
	}

	fmt.Println("Ray tracing completed successfully")
}

func min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
