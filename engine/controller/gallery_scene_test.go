package controller

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/factory"
	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model"
)

func TestNonEuclideanGallerySceneLoadsAndRendersSmokeFrame(t *testing.T) {
	scenePath := filepath.Join("..", "..", "examples", "scenes", "non-euclidean", "hyperbolic.json")
	script, err := parser.ReadScriptFile(scenePath)
	if err != nil {
		t.Fatalf("read gallery scene: %v", err)
	}

	if len(script.Objects) < 40 {
		t.Fatalf("expected a furnished gallery scene, got %d objects", len(script.Objects))
	}
	if script.Geometry == nil || script.Geometry.Type != "klein" {
		t.Fatalf("expected Klein scene geometry, got %#v", script.Geometry)
	}
	if got := script.Materials[4]["id"]; got != "wall_back" {
		t.Fatalf("expected hyperbolic gallery back wall material, got %v", got)
	}

	scene := model.NewScene()
	if err := factory.LoadSceneFromScript(script, scene); err != nil {
		t.Fatalf("load gallery scene: %v", err)
	}
	if len(scene.Cameras) != 1 {
		t.Fatalf("expected one camera, got %d", len(scene.Cameras))
	}

	outputImage := filepath.Join(t.TempDir(), "gallery-smoke.png")
	handler := NewHandler().
		LoadScript(scenePath).
		ConfigureRender(RenderOverrides{
			CameraIndex: 0,
			ThreadNum:   1,
			Width:       12,
			Height:      12,
			Samples:     1,
			OutputImage: outputImage,
			OutputFilm:  "",
		}).
		Render().
		SaveOutputs()
	if handler.err != nil {
		t.Fatalf("smoke render failed: %v", handler.err)
	}

	file, err := os.Open(outputImage)
	if err != nil {
		t.Fatalf("open smoke render: %v", err)
	}
	defer file.Close()

	config, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatalf("decode smoke render: %v", err)
	}
	if config.Width != 12 || config.Height != 12 {
		t.Fatalf("expected 12x12 smoke render, got %dx%d", config.Width, config.Height)
	}
}
