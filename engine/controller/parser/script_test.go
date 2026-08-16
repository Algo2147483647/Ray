package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadScriptFilePreservesGeometry(t *testing.T) {
	dir := t.TempDir()
	writeTestScript(t, filepath.Join(dir, "main.json"), `{
		"geometry": {"type": "klein", "max_arc": 12.5},
		"renders": [{"dimension": 3}]
	}`)

	script, err := ReadScriptFile(filepath.Join(dir, "main.json"))
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	if script.Geometry == nil {
		t.Fatal("expected geometry to survive script read")
	}
	if script.Geometry.Type != "klein" || script.Geometry.MaxArc != 12.5 {
		t.Fatalf("unexpected geometry: %#v", script.Geometry)
	}
}

func TestReadScriptFileAcceptsCameraOwnedFilm(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.json")
	writeTestScript(t, path, `{
		"cameras":[{"id":"main","film":{"shape":[800,600],"output_film":"main.bin"}}],
		"renders":[{"camera_id":"main"}]
	}`)
	script, err := ReadScriptFile(path)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	if len(script.Renders) != 1 || script.Renders[0].CameraID != "main" || len(script.Cameras) != 1 || script.Cameras[0].Film.Shape[0] != 800 {
		t.Fatalf("unexpected camera-owned film: %+v", script)
	}
}

func writeTestScript(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
