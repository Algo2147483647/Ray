package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadScriptFileMergesIncludesRelativeToParent(t *testing.T) {
	dir := t.TempDir()
	writeTestScript(t, filepath.Join(dir, "studio.json"), `{
		"materials": [
			{"id": "floor", "surface": {"type": "lambert", "albedo": [0.5, 0.5, 0.5]}}
		],
		"objects": [
			{"id": "floor-a", "shape": "sphere", "position": [0, 0, 0], "r": 1, "material_id": "floor"}
		],
		"cameras": [
			{"id": "cam-a", "type": "3d", "position": [0, -3, 1], "look_at": [0, 0, 0], "up": [0, 0, 1]}
		],
		"render": {
			"samples": 8,
			"output_image": "studio.png"
		}
	}`)
	writeTestScript(t, filepath.Join(dir, "main.json"), `{
		"includes": ["studio.json"],
		"materials": [
			{"id": "heart-red", "surface": {"type": "lambert", "albedo": [0.85, 0.05, 0.035]}}
		],
		"objects": [
			{"id": "heart", "shape": "sphere", "position": [0, 0, 1], "r": 1, "material_id": "heart-red"}
		],
		"render": {
			"samples": 16
		},
		"renders": [
			{"output_image": "front.png"},
			{"samples": 32, "output_image": "detail.png"}
		]
	}`)

	script, err := ReadScriptFile(filepath.Join(dir, "main.json"))
	if err != nil {
		t.Fatalf("read merged script: %v", err)
	}

	if got := len(script.Materials); got != 2 {
		t.Fatalf("expected two merged materials, got %d", got)
	}
	if got := len(script.Objects); got != 2 {
		t.Fatalf("expected two merged objects, got %d", got)
	}
	if got := len(script.Cameras); got != 1 {
		t.Fatalf("expected one merged camera, got %d", got)
	}
	if script.Render.Samples != 16 {
		t.Fatalf("expected main render to override included samples, got %d", script.Render.Samples)
	}
	if script.Render.OutputImage != "studio.png" {
		t.Fatalf("expected included output image to remain when main omits it, got %q", script.Render.OutputImage)
	}
	if got := len(script.Renders); got != 2 {
		t.Fatalf("expected two render jobs, got %d", got)
	}
}

func TestReadScriptFileRejectsDuplicateIncludedIDs(t *testing.T) {
	dir := t.TempDir()
	writeTestScript(t, filepath.Join(dir, "a.json"), `{
		"materials": [
			{"id": "mat", "surface": {"type": "lambert", "albedo": [0.5, 0.5, 0.5]}}
		]
	}`)
	writeTestScript(t, filepath.Join(dir, "main.json"), `{
		"includes": ["a.json"],
		"materials": [
			{"id": "mat", "surface": {"type": "lambert", "albedo": [0.8, 0.2, 0.2]}}
		]
	}`)

	_, err := ReadScriptFile(filepath.Join(dir, "main.json"))
	if err == nil || !strings.Contains(err.Error(), `duplicate material id "mat"`) {
		t.Fatalf("expected duplicate material id error, got %v", err)
	}
}

func TestReadScriptFileRejectsIncludeCycles(t *testing.T) {
	dir := t.TempDir()
	writeTestScript(t, filepath.Join(dir, "a.json"), `{"includes": ["b.json"]}`)
	writeTestScript(t, filepath.Join(dir, "b.json"), `{"includes": ["a.json"]}`)

	_, err := ReadScriptFile(filepath.Join(dir, "a.json"))
	if err == nil || !strings.Contains(err.Error(), "include cycle detected") {
		t.Fatalf("expected include cycle error, got %v", err)
	}
}

func writeTestScript(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
