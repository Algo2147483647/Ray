package controller

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
)

func TestNonEuclideanCanonicalSceneLoadsAndRendersSmokeFrame(t *testing.T) {
	dir := t.TempDir()
	scenePath := filepath.Join(dir, "hyperbolic-canonical.json")
	writeControllerTestScript(t, scenePath, `{
		"render": {
			"dimension": 3,
			"samples": 1,
			"camera_index": 0,
			"spectrum_mode": "rgb"
		},
		"geometry": {
			"type": "klein"
		},
		"materials": [
			{
				"id": "matte",
				"surface": {
					"type": "lambert",
					"albedo": [0.8, 0.6, 0.4]
				}
			}
		],
		"objects": [
			{
				"id": "ball",
				"shape": "sphere",
				"center": [0.2, 0, 0],
				"r": 0.2,
				"material_id": "matte"
			}
		],
		"cameras": [
			{
				"id": "main",
				"type": "hyperbolic",
				"position": [-0.4, 0, 0],
				"direction": [1, 0, 0],
				"up": [0, 0, 1],
				"field_of_view": 70,
				"aspect_ratio": 1
			}
		]
	}`)

	script, err := parser.ReadScriptFile(scenePath)
	if err != nil {
		t.Fatalf("read scene: %v", err)
	}

	if len(script.Objects) != 1 {
		t.Fatalf("expected one scene object, got %d", len(script.Objects))
	}
	if script.Geometry == nil || script.Geometry.Type != "klein" {
		t.Fatalf("expected Klein scene geometry, got %#v", script.Geometry)
	}
	if !hasMaterialID(script.Materials, "matte") {
		t.Fatal("expected matte material")
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

func hasMaterialID(materials []map[string]interface{}, id string) bool {
	for _, material := range materials {
		if material["id"] == id {
			return true
		}
	}
	return false
}

func writeControllerTestScript(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
