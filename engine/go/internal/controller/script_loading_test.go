package controller

import (
	"github.com/Algo2147483647/ray/engine/go/internal/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadScriptFileReturnsErrorForInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(path, []byte(`{"materials": [`), 0o600); err != nil {
		t.Fatalf("write temp script: %v", err)
	}

	script, err := ReadScriptFile(path)
	if err == nil {
		t.Fatal("expected invalid JSON to return an error")
	}
	if script != nil {
		t.Fatal("expected nil script on invalid JSON")
	}
}

func TestLoadSceneFromScriptReportsUndefinedMaterial(t *testing.T) {
	script := &Script{
		Objects: []map[string]interface{}{
			{
				"id":          "orphan-sphere",
				"shape":       "sphere",
				"position":    []interface{}{0.0, 0.0, 0.0},
				"r":           1.0,
				"material_id": "missing",
			},
		},
	}

	err := LoadSceneFromScript(script, model.NewScene())
	if err == nil {
		t.Fatal("expected undefined material error")
	}
	if !strings.Contains(err.Error(), `undefined material "missing"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), `object[0] id="orphan-sphere"`) {
		t.Fatalf("missing object context in error: %v", err)
	}
}

func TestLoadSceneFromScriptReportsUnsupportedShape(t *testing.T) {
	script := &Script{
		Materials: []map[string]interface{}{
			{
				"id": "mat",
				"surface": map[string]interface{}{
					"type":   "lambert",
					"albedo": []interface{}{1.0, 1.0, 1.0},
				},
			},
		},
		Objects: []map[string]interface{}{
			{
				"id":          "mystery",
				"shape":       "banana",
				"material_id": "mat",
			},
		},
	}

	err := LoadSceneFromScript(script, model.NewScene())
	if err == nil {
		t.Fatal("expected unsupported shape error")
	}
	if !strings.Contains(err.Error(), `unsupported shape "banana"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMaterialsRejectsDuplicateID(t *testing.T) {
	script := &Script{
		Materials: []map[string]interface{}{
			{
				"id": "dup",
				"surface": map[string]interface{}{
					"type":   "lambert",
					"albedo": []interface{}{1.0, 1.0, 1.0},
				},
			},
			{
				"id": "dup",
				"surface": map[string]interface{}{
					"type":   "lambert",
					"albedo": []interface{}{0.0, 0.0, 0.0},
				},
			},
		},
	}

	_, err := ParseMaterials(script)
	if err == nil {
		t.Fatal("expected duplicate material id error")
	}
	if !strings.Contains(err.Error(), `duplicate material id`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMaterialsSupportsRoughConductor(t *testing.T) {
	script := &Script{
		Materials: []map[string]interface{}{
			{
				"id": "metal",
				"surface": map[string]interface{}{
					"type":      "rough_conductor",
					"eta":       []interface{}{0.2, 0.9, 1.5},
					"k":         []interface{}{3.9, 2.5, 1.9},
					"roughness": 0.25,
				},
			},
		},
	}

	materials, err := ParseMaterials(script)
	if err != nil {
		t.Fatalf("parse rough conductor: %v", err)
	}
	if materials["metal"] == nil || materials["metal"].Surface == nil {
		t.Fatal("expected parsed rough conductor surface")
	}
}

func TestParseMaterialsSupportsCauchyIOR(t *testing.T) {
	script := &Script{
		Materials: []map[string]interface{}{
			{
				"id": "dispersion-glass",
				"surface": map[string]interface{}{
					"type":          "specular_dielectric",
					"reflectance":   []interface{}{1.0, 1.0, 1.0},
					"transmittance": []interface{}{1.0, 1.0, 1.0},
					"eta_outside":   1.0,
					"ior": map[string]interface{}{
						"type": "cauchy",
						"a":    1.5046,
						"b":    0.0042,
						"c":    0.0,
					},
				},
			},
		},
	}

	materials, err := ParseMaterials(script)
	if err != nil {
		t.Fatalf("parse cauchy ior: %v", err)
	}
	if materials["dispersion-glass"] == nil || materials["dispersion-glass"].Surface == nil {
		t.Fatal("expected parsed dielectric surface")
	}
}
