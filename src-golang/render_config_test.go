package main

import (
	"runtime"
	"src-golang/controller"
	"testing"
)

func TestResolveRenderConfigMergesScriptAndCLI(t *testing.T) {
	script := &controller.Script{
		Render: controller.RenderScript{
			Samples:     12,
			ThreadNum:   3,
			CameraIndex: 1,
			Width:       800,
			Height:      600,
			OutputImage: "scene.png",
			OutputFilm:  "scene.bin",
			DebugOutput: "scene-debug.json",
		},
	}

	config := ResolveRenderConfig(script, RenderOverrides{
		ScriptPath:  "custom.json",
		CameraIndex: 2,
		ThreadNum:   6,
		Width:       1024,
		Samples:     32,
		OutputImage: "override.png",
	})

	if config.ScriptPath != "custom.json" {
		t.Fatalf("unexpected script path: %s", config.ScriptPath)
	}
	if config.CameraIndex != 2 {
		t.Fatalf("expected CLI camera index override, got %d", config.CameraIndex)
	}
	if config.ThreadNum != 6 {
		t.Fatalf("expected CLI thread override, got %d", config.ThreadNum)
	}
	if config.Width != 1024 || config.Height != 600 {
		t.Fatalf("unexpected dimensions: %dx%d", config.Width, config.Height)
	}
	if config.Samples != 32 {
		t.Fatalf("expected CLI samples override, got %d", config.Samples)
	}
	if config.OutputImage != "override.png" {
		t.Fatalf("expected CLI output image override, got %s", config.OutputImage)
	}
	if config.OutputFilm != "scene.bin" || config.DebugOutput != "scene-debug.json" {
		t.Fatalf("unexpected output fallback: film=%s debug=%s", config.OutputFilm, config.DebugOutput)
	}
}

func TestResolveRenderConfigDefaultsThreadNumToNumCPU(t *testing.T) {
	config := ResolveRenderConfig(nil, RenderOverrides{ScriptPath: "test.json"})
	if config.ThreadNum != runtime.NumCPU() {
		t.Fatalf("expected default thread count %d, got %d", runtime.NumCPU(), config.ThreadNum)
	}
}
