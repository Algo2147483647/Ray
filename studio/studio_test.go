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

func TestStudioAdaptsTriangleCenterAndGroupPlacement(t *testing.T) {
	script := &parser.Script{
		Objects: []map[string]interface{}{
			{
				"id":     "g",
				"shape":  "group",
				"center": []interface{}{10, 0, 0},
				"scale":  2,
				"objects": []interface{}{
					map[string]interface{}{
						"id":     "tri",
						"shape":  "triangle",
						"center": []interface{}{1, 1, 1},
						"p1":     []interface{}{0, 0, 0},
						"p2":     []interface{}{1, 0, 0},
						"p3":     []interface{}{0, 1, 0},
					},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt triangle: %v", err)
	}
	triangle := adapted.Objects[0]
	assertFloatSlice(t, triangle["p1"], []float64{12, 2, 2})
	assertFloatSlice(t, triangle["p2"], []float64{14, 2, 2})
	assertFloatSlice(t, triangle["p3"], []float64{12, 4, 2})
	if _, ok := triangle["center"]; ok {
		t.Fatal("triangle intermediate object should not keep center")
	}
}

func TestStudioAdaptsCuboidPositionSizeToMinMax(t *testing.T) {
	script := &parser.Script{
		Objects: []map[string]interface{}{
			{
				"id":     "g",
				"shape":  "group",
				"center": []interface{}{10, 0, 0},
				"scale":  2,
				"objects": []interface{}{
					map[string]interface{}{
						"id":       "box",
						"shape":    "cuboid",
						"position": []interface{}{1, 1, 1},
						"size":     []interface{}{2, 4, 6},
					},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt cuboid: %v", err)
	}
	cuboid := adapted.Objects[0]
	assertFloatSlice(t, cuboid["pmin"], []float64{10, -2, -4})
	assertFloatSlice(t, cuboid["pmax"], []float64{14, 6, 8})
	if _, ok := cuboid["position"]; ok {
		t.Fatal("cuboid intermediate object should not keep position")
	}
	if _, ok := cuboid["size"]; ok {
		t.Fatal("cuboid intermediate object should not keep size")
	}
}

func TestStudioAdaptsHypercubeToCuboid(t *testing.T) {
	script := &parser.Script{
		Objects: []map[string]interface{}{
			{
				"id":     "cube",
				"shape":  "hypercube",
				"center": []interface{}{1, 2, 3, 4},
				"size":   []interface{}{2, 2, 2, 2},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 4)
	if err != nil {
		t.Fatalf("adapt hypercube: %v", err)
	}
	cuboid := adapted.Objects[0]
	if cuboid["shape"] != "cuboid" {
		t.Fatalf("expected hypercube to become engine cuboid, got %v", cuboid["shape"])
	}
	assertFloatSlice(t, cuboid["pmin"], []float64{0, 1, 2, 3})
	assertFloatSlice(t, cuboid["pmax"], []float64{2, 3, 4, 5})
}

func TestStudioRejectsUnequalHypercubeExtents(t *testing.T) {
	script := &parser.Script{
		Objects: []map[string]interface{}{
			{
				"id":     "bad-cube",
				"shape":  "hypercube",
				"center": []interface{}{0, 0, 0},
				"size":   []interface{}{2, 3, 2},
			},
		},
	}

	if _, err := adaptScript(script, []string{"scene.json"}, 3); err == nil {
		t.Fatal("expected unequal hypercube extents to fail")
	}
}

func TestStudioAdaptsQuadraticCenterScaleToWorldCoefficients(t *testing.T) {
	script := &parser.Script{
		Objects: []map[string]interface{}{
			{
				"id":     "quad",
				"shape":  "quadratic equation",
				"a":      []interface{}{1, 0, 0, 0, 0, 0, 0, 0, 0},
				"b":      []interface{}{0, 0, 0},
				"c":      -1,
				"center": []interface{}{2, 0, 0},
				"scale":  3,
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt quadratic: %v", err)
	}
	quadratic := adapted.Objects[0]
	a := mustFloatSlice(t, quadratic["a"])
	b := mustFloatSlice(t, quadratic["b"])
	c, ok := quadratic["c"].(float64)
	if !ok {
		t.Fatalf("expected quadratic c float64, got %T", quadratic["c"])
	}
	if math.Abs(a[0]-1.0/9.0) > 1e-10 || math.Abs(b[0]+4.0/9.0) > 1e-10 || math.Abs(c+5.0/9.0) > 1e-10 {
		t.Fatalf("unexpected baked quadratic coefficients: a=%v b=%v c=%v", a, b, c)
	}
	if _, ok := quadratic["center"]; ok {
		t.Fatal("quadratic intermediate object should not keep center")
	}
	if _, ok := quadratic["scale"]; ok {
		t.Fatal("quadratic intermediate object should not keep scale")
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

func assertFloatSlice(t *testing.T, raw interface{}, expected []float64) {
	t.Helper()
	values := mustFloatSlice(t, raw)
	if len(values) != len(expected) {
		t.Fatalf("expected %d values, got %d: %v", len(expected), len(values), values)
	}
	for i := range values {
		if math.Abs(values[i]-expected[i]) > 1e-10 {
			t.Fatalf("index %d: expected %v, got %v", i, expected, values)
		}
	}
}

func mustFloatSlice(t *testing.T, raw interface{}) []float64 {
	t.Helper()
	values, ok := raw.([]float64)
	if !ok {
		t.Fatalf("expected []float64, got %T", raw)
	}
	return values
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
