package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadScriptFilePreservesGeometry(t *testing.T) {
	dir := t.TempDir()
	writeTestScript(t, filepath.Join(dir, "main.json"), `{
		"dimension": 3,
		"geometry": {"type": "klein", "max_arc": 12.5},
		"renders": [{}]
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
	if script.Dimension != 3 {
		t.Fatalf("unexpected scene dimension: %d", script.Dimension)
	}
}

func TestReadScriptFileAcceptsRenderTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.json")
	writeTestScript(t, path, `{
		"cameras":[{"id":"main"}],
		"renders":[{"camera_id":"main","film":{"shape":[800,600]},"output":"main.bin"}]
	}`)
	script, err := ReadScriptFile(path)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	if len(script.Renders) != 1 || script.Renders[0].CameraID != "main" || len(script.Cameras) != 1 || script.Renders[0].Film.Shape[0] != 800 || script.Renders[0].Output != "main.bin" {
		t.Fatalf("unexpected render target: %+v", script)
	}
}

func TestReadScriptFileRejectsCameraOwnedFilm(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.json")
	writeTestScript(t, path, `{"cameras":[{"id":"main","film":{"shape":[1,1]}}]}`)
	if _, err := ReadScriptFile(path); err == nil {
		t.Fatal("expected removed camera film field to be rejected")
	}
}

func TestReadScriptFileRejectsOutputInsideFilm(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.json")
	writeTestScript(t, path, `{"renders":[{"camera_id":"main","film":{"shape":[1,1],"output_film":"old.bin"},"output":"main.bin"}]}`)
	if _, err := ReadScriptFile(path); err == nil {
		t.Fatal("expected removed film output_film field to be rejected")
	}
}

func TestReadScriptFileRejectsBDPTFallbackPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.json")
	writeTestScript(t, path, `{"renders":[{"integrator":"bdpt","bdpt_fallback_policy":"path"}]}`)
	if _, err := ReadScriptFile(path); err == nil {
		t.Fatal("expected removed BDPT fallback policy to be rejected")
	}
}

func TestReadScriptFileRejectsRenderDimension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.json")
	writeTestScript(t, path, `{"dimension":3,"renders":[{"dimension":3}]}`)
	if _, err := ReadScriptFile(path); err == nil {
		t.Fatal("expected render-local dimension to be rejected")
	}
}

func TestReadScriptFileRejectsRemovedSpectrumMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.json")
	writeTestScript(t, path, `{"renders":[{"spectrum_mode":"hero_wavelength"}]}`)
	if _, err := ReadScriptFile(path); err == nil {
		t.Fatal("expected removed spectrum_mode to be rejected")
	}
}

func TestReadScriptFileRejectsNonPositiveExplicitRenderCounts(t *testing.T) {
	tests := []string{
		`{"renders":[{"samples":0}]}`,
		`{"renders":[{"samples":-1}]}`,
		`{"renders":[{"thread_num":0}]}`,
		`{"renders":[{"thread_num":-1}]}`,
		`{"renders":[{"wavelength_samples":0}]}`,
		`{"renders":[{"wavelength_samples":-1}]}`,
	}
	for _, script := range tests {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.json")
		writeTestScript(t, path, script)
		if _, err := ReadScriptFile(path); err == nil {
			t.Fatalf("expected explicit non-positive render count to be rejected: %s", script)
		}
	}
}

func TestReadScriptFileRejectsUnknownFieldsAtEverySceneBoundary(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		message string
	}{
		{
			name:    "script",
			script:  `{"dimension":3,"mystery":true}`,
			message: `unsupported script field "mystery"`,
		},
		{
			name:    "geometry",
			script:  `{"geometry":{"type":"euclidean","warp":1}}`,
			message: `unsupported geometry field "warp"`,
		},
		{
			name:    "medium",
			script:  `{"media":{"water":{"type":"homogeneous","density":1}}}`,
			message: `unsupported medium field "density"`,
		},
		{
			name:    "medium discriminator",
			script:  `{"media":{"fog":{"type":"heterogeneous"}}}`,
			message: `unsupported medium type "heterogeneous"`,
		},
		{
			name:    "ior variant",
			script:  `{"media":{"glass":{"ior":{"type":"constant","eta":1.5,"a":1}}}}`,
			message: `unsupported ior field "a"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "main.json")
			writeTestScript(t, path, test.script)
			_, err := ReadScriptFile(path)
			if err == nil {
				t.Fatalf("expected %s field to be rejected", test.name)
			}
			if !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected error containing %q, got %v", test.message, err)
			}
		})
	}
}

func writeTestScript(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
