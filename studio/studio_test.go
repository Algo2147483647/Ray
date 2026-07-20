package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/factory"
	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model"
)

func TestFlattenNestedGroupAndInheritFields(t *testing.T) {
	script := &parser.Script{
		Objects: []map[string]interface{}{
			{
				"id":          "outer",
				"shape":       "group",
				"center":      []interface{}{2, 0, 0},
				"scale":       3,
				"material_id": "glass",
				"objects": []interface{}{
					map[string]interface{}{
						"id":     "inner",
						"shape":  "group",
						"center": []interface{}{1, 0, 0},
						"scale":  []interface{}{1, 2, 1},
						"objects": []interface{}{
							map[string]interface{}{
								"id":     "surface",
								"shape":  "cubic equation",
								"a":      unitCubicCoefficients(),
								"center": []interface{}{0, 0, 0},
								"scale":  1,
							},
							map[string]interface{}{
								"id":    "marker",
								"shape": "sphere",
								"r":     1,
							},
						},
					},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if len(adapted.Objects) != 2 {
		t.Fatalf("expected two flattened objects, got %d", len(adapted.Objects))
	}

	cubic := adapted.Objects[0]
	if cubic["id"] != "outer/inner/surface" {
		t.Fatalf("unexpected cubic id: %v", cubic["id"])
	}
	if cubic["material_id"] != "glass" {
		t.Fatalf("expected inherited material_id, got %v", cubic["material_id"])
	}
	if _, ok := cubic["center"]; ok {
		t.Fatal("cubic intermediate object should not keep center")
	}
	if _, ok := cubic["scale"]; ok {
		t.Fatal("cubic intermediate object should not keep scale")
	}
	coefficients, ok := cubic["a"].([]float64)
	if !ok {
		t.Fatalf("expected baked coefficients, got %T", cubic["a"])
	}
	if math.Abs(coefficients[0]+152.0/27.0) > 1e-10 {
		t.Fatalf("expected baked constant -152/27, got %f", coefficients[0])
	}

	sphere := adapted.Objects[1]
	if sphere["id"] != "outer/inner/marker" {
		t.Fatalf("unexpected sphere id: %v", sphere["id"])
	}
	if sphere["shape"] == "group" {
		t.Fatal("intermediate output must not contain group objects")
	}
	if sphere["material_id"] != "glass" {
		t.Fatalf("expected inherited material_id, got %v", sphere["material_id"])
	}
}

func TestChildFieldOverridesGroupInheritance(t *testing.T) {
	script := &parser.Script{
		Objects: []map[string]interface{}{
			{
				"id":          "g",
				"shape":       "group",
				"material_id": "outer",
				"objects": []interface{}{
					map[string]interface{}{
						"id":          "child",
						"shape":       "sphere",
						"material_id": "inner",
						"r":           1,
					},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if adapted.Objects[0]["material_id"] != "inner" {
		t.Fatalf("expected child material override, got %v", adapted.Objects[0]["material_id"])
	}
}

func TestStudioAdaptsCopiedGeometryBenchmarkMatrixExample(t *testing.T) {
	sourceDir := filepath.Join("..", "examples", "scenes", "geometry-benchmark-matrix")
	sceneDir := filepath.Join(t.TempDir(), "geometry-benchmark-matrix")
	if err := copyDirectory(sourceDir, sceneDir); err != nil {
		t.Fatalf("copy geometry benchmark matrix scene: %v", err)
	}

	scriptPaths := []string{
		filepath.Join(sceneDir, "room.json"),
		filepath.Join(sceneDir, "main.json"),
		filepath.Join(sceneDir, "materials.json"),
		filepath.Join(sceneDir, "geo_example.json"),
	}
	script, err := parser.ReadScriptFiles(scriptPaths)
	if err != nil {
		t.Fatalf("read copied geometry benchmark scripts: %v", err)
	}

	adapted, err := adaptScript(script, scriptPaths, 3)
	if err != nil {
		t.Fatalf("adapt copied geometry benchmark scene through studio: %v", err)
	}
	if adapted.Studio.Version == "" {
		t.Fatal("expected studio metadata on intermediate script")
	}
	if len(adapted.Objects) != 21 {
		t.Fatalf("expected room plus example geometry objects, got %d", len(adapted.Objects))
	}
	for _, object := range adapted.Objects {
		if shape, _ := stringField(object, "shape"); strings.EqualFold(shape, "group") {
			t.Fatalf("studio intermediate output must not contain group object: %#v", object)
		}
	}

	data, err := json.Marshal(adapted)
	if err != nil {
		t.Fatalf("marshal studio intermediate script: %v", err)
	}
	var engineScript parser.Script
	if err := json.Unmarshal(data, &engineScript); err != nil {
		t.Fatalf("unmarshal studio intermediate as engine script: %v", err)
	}
	if err := factory.LoadSceneFromScript(&engineScript, model.NewScene()); err != nil {
		t.Fatalf("engine failed to load studio intermediate geometry benchmark scene: %v", err)
	}
}

func unitCubicCoefficients() []interface{} {
	coefficients := make([]interface{}, 64)
	for i := range coefficients {
		coefficients[i] = 0
	}
	coefficients[(1*4+1)*4+1] = 1
	coefficients[0] = -1
	return coefficients
}

func copyDirectory(source, destination string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(source, entry.Name())
		destinationPath := filepath.Join(destination, entry.Name())
		if entry.IsDir() {
			if err := copyDirectory(sourcePath, destinationPath); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(destinationPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
