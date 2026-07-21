package controller

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model/camera"
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
				"field_of_views": [70, 70]
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

	outputFilm := filepath.Join(t.TempDir(), "gallery-smoke.bin")
	handler := NewHandler().
		LoadScript(scenePath).
		ConfigureRender(RenderOverrides{
			CameraIndex: 0,
			ThreadNum:   1,
			Width:       12,
			Height:      12,
			Samples:     1,
			OutputFilm:  outputFilm,
		}).
		Render().
		SaveOutputs()
	if handler.err != nil {
		t.Fatalf("smoke render failed: %v", handler.err)
	}

	film := camera.NewFilm()
	if err := film.LoadFromFile(outputFilm); err != nil {
		t.Fatalf("load smoke render film: %v", err)
	}
	if len(film.Data[0].Shape) != 2 || film.Data[0].Shape[0] != 12 || film.Data[0].Shape[1] != 12 {
		t.Fatalf("expected 12x12 smoke render film, got shape %v", film.Data[0].Shape)
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
