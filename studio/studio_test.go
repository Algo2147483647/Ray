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
	modelshape "github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
)

func TestFlattenNestedGroupAndInheritFields(t *testing.T) {
	script := &studioScript{
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
	script := &studioScript{
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

func TestGroupDoesNotRequireMaterialID(t *testing.T) {
	script := &studioScript{
		Objects: []map[string]interface{}{
			{
				"id":     "g",
				"shape":  "group",
				"center": []interface{}{1, 2, 3},
				"objects": []interface{}{
					map[string]interface{}{
						"id":          "child",
						"shape":       "sphere",
						"center":      []interface{}{0, 0, 0},
						"r":           1,
						"material_id": "child-material",
					},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt group without material_id: %v", err)
	}
	if len(adapted.Objects) != 1 {
		t.Fatalf("expected one flattened object, got %d", len(adapted.Objects))
	}
	object := adapted.Objects[0]
	if object["id"] != "g/child" {
		t.Fatalf("unexpected child id: %v", object["id"])
	}
	if object["material_id"] != "child-material" {
		t.Fatalf("expected child material_id to be preserved, got %v", object["material_id"])
	}
}

func TestStudioAdaptsArrayCells(t *testing.T) {
	script := &studioScript{
		Objects: []map[string]interface{}{
			{
				"id":          "grid",
				"shape":       "array",
				"origin":      []interface{}{10, 0, 0},
				"delta":       []interface{}{[]interface{}{1, 0, 0}, []interface{}{0, 2, 0}},
				"counts":      []interface{}{2, 2},
				"material_id": "array-material",
				"objects": map[string]interface{}{
					"1,1": []interface{}{
						map[string]interface{}{
							"id":     "a",
							"shape":  "sphere",
							"center": []interface{}{0.5, 0, 0},
							"r":      0.25,
						},
					},
					"2,2": []interface{}{
						map[string]interface{}{
							"id":     "b",
							"shape":  "sphere",
							"center": []interface{}{0, 0.5, 0},
							"r":      0.25,
						},
					},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt array: %v", err)
	}
	if len(adapted.Objects) != 2 {
		t.Fatalf("expected two array objects, got %d", len(adapted.Objects))
	}

	first := adapted.Objects[0]
	if first["id"] != "grid/i1-j1/a" {
		t.Fatalf("unexpected first id: %v", first["id"])
	}
	assertFloatSlice(t, first["center"], []float64{10.5, 0, 0})
	if first["material_id"] != "array-material" {
		t.Fatalf("expected inherited material, got %v", first["material_id"])
	}

	second := adapted.Objects[1]
	if second["id"] != "grid/i2-j2/b" {
		t.Fatalf("unexpected second id: %v", second["id"])
	}
	assertFloatSlice(t, second["center"], []float64{11, 2.5, 0})
}

func TestStudioMergesArrayObjectsAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "a.json")
	secondPath := filepath.Join(dir, "b.json")
	if err := os.WriteFile(firstPath, []byte(`{
	  "objects": [
	    {
	      "id": "grid",
	      "shape": "array",
	      "origin": [0, 0, 0],
	      "delta": [[1, 0, 0]],
	      "counts": [2],
	      "material_id": "shared",
	      "objects": {
	        "1": [
	          { "id": "left", "shape": "sphere", "center": [0, 0, 0], "r": 0.25 }
	        ]
	      }
	    }
	  ]
	}`), 0o644); err != nil {
		t.Fatalf("write first script: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte(`{
	  "objects": [
	    {
	      "id": "grid",
	      "shape": "array",
	      "objects": {
	        "2": [
	          { "id": "right", "shape": "sphere", "center": [0, 0, 0], "r": 0.25 }
	        ]
	      }
	    }
	  ]
	}`), 0o644); err != nil {
		t.Fatalf("write second script: %v", err)
	}

	script, err := readStudioScriptFiles([]string{firstPath, secondPath})
	if err != nil {
		t.Fatalf("read merged studio scripts: %v", err)
	}
	if len(script.Objects) != 1 {
		t.Fatalf("expected one merged array, got %d", len(script.Objects))
	}

	adapted, err := adaptScript(script, []string{firstPath, secondPath}, 3)
	if err != nil {
		t.Fatalf("adapt merged array: %v", err)
	}
	if len(adapted.Objects) != 2 {
		t.Fatalf("expected two merged array children, got %d", len(adapted.Objects))
	}
	if adapted.Objects[0]["id"] != "grid/i1/left" || adapted.Objects[1]["id"] != "grid/i2/right" {
		t.Fatalf("unexpected merged ids: %v, %v", adapted.Objects[0]["id"], adapted.Objects[1]["id"])
	}
	assertFloatSlice(t, adapted.Objects[0]["center"], []float64{0, 0, 0})
	assertFloatSlice(t, adapted.Objects[1]["center"], []float64{1, 0, 0})
}

func TestStudioMergesGroupObjectsAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "group-a.json")
	secondPath := filepath.Join(dir, "group-b.json")
	if err := os.WriteFile(firstPath, []byte(`{
	  "objects": [
	    {
	      "id": "cluster",
	      "shape": "group",
	      "center": [2, 0, 0],
	      "objects": [
	        { "id": "left", "shape": "sphere", "center": [0, 0, 0], "r": 0.25, "material_id": "mat" }
	      ]
	    }
	  ]
	}`), 0o644); err != nil {
		t.Fatalf("write first group script: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte(`{
	  "objects": [
	    {
	      "id": "cluster",
	      "shape": "group",
	      "objects": [
	        { "id": "right", "shape": "sphere", "center": [1, 0, 0], "r": 0.25, "material_id": "mat" }
	      ]
	    }
	  ]
	}`), 0o644); err != nil {
		t.Fatalf("write second group script: %v", err)
	}

	script, err := readStudioScriptFiles([]string{firstPath, secondPath})
	if err != nil {
		t.Fatalf("read merged group scripts: %v", err)
	}
	adapted, err := adaptScript(script, []string{firstPath, secondPath}, 3)
	if err != nil {
		t.Fatalf("adapt merged group: %v", err)
	}
	if len(adapted.Objects) != 2 {
		t.Fatalf("expected two merged group children, got %d", len(adapted.Objects))
	}
	if adapted.Objects[0]["id"] != "cluster/left" || adapted.Objects[1]["id"] != "cluster/right" {
		t.Fatalf("unexpected merged group ids: %v, %v", adapted.Objects[0]["id"], adapted.Objects[1]["id"])
	}
	assertFloatSlice(t, adapted.Objects[0]["center"], []float64{2, 0, 0})
	assertFloatSlice(t, adapted.Objects[1]["center"], []float64{3, 0, 0})
}

func TestStudioAdaptsTriangleCenterAndGroupPlacement(t *testing.T) {
	script := &studioScript{
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

func TestStudioAdaptsBasicShapesWithGroupPlacement(t *testing.T) {
	script := &studioScript{
		Objects: []map[string]interface{}{
			{
				"id":     "g",
				"shape":  "group",
				"center": []interface{}{10, 0, 1},
				"scale":  2,
				"objects": []interface{}{
					map[string]interface{}{
						"id":     "ball",
						"shape":  "sphere",
						"center": []interface{}{1, 2, 3},
						"r":      0.5,
					},
					map[string]interface{}{
						"id":     "disk",
						"shape":  "circle",
						"center": []interface{}{0, 1, 0},
						"normal": []interface{}{0, 0, 1},
						"r":      0.25,
					},
					map[string]interface{}{
						"id":     "tube",
						"shape":  "cylinder",
						"center": []interface{}{0, 0, 1},
						"axis":   []interface{}{0, 0, 1},
						"r":      0.1,
						"height": 0.75,
					},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt basic shapes: %v", err)
	}
	if len(adapted.Objects) != 3 {
		t.Fatalf("expected three flattened objects, got %d", len(adapted.Objects))
	}

	sphere := adapted.Objects[0]
	assertFloatSlice(t, sphere["center"], []float64{12, 4, 7})
	assertFloatValue(t, sphere["r"], 1)

	circle := adapted.Objects[1]
	assertFloatSlice(t, circle["center"], []float64{10, 2, 1})
	assertFloatValue(t, circle["r"], 0.5)

	cylinder := adapted.Objects[2]
	assertFloatSlice(t, cylinder["center"], []float64{10, 0, 3})
	assertFloatValue(t, cylinder["r"], 0.2)
	assertFloatValue(t, cylinder["height"], 1.5)
}

func TestStudioAdaptsCuboidPositionSizeToMinMax(t *testing.T) {
	script := &studioScript{
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
	script := &studioScript{
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

func TestStudioAdaptsBoundsCenterSizeToMinMax(t *testing.T) {
	script := &studioScript{
		Objects: []map[string]interface{}{
			{
				"id":    "expr",
				"shape": "implicit equation",
				"field": map[string]interface{}{
					"type": "expr",
					"expr": "x*x + y*y + z*z - 1",
				},
				"bounds": map[string]interface{}{
					"center": []interface{}{1, 2, 3},
					"size":   []interface{}{2, 4, 6},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt bounds: %v", err)
	}

	bounds, ok := adapted.Objects[0]["bounds"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected bounds object, got %T", adapted.Objects[0]["bounds"])
	}
	assertFloatSlice(t, bounds["pmin"], []float64{0, 0, 0})
	assertFloatSlice(t, bounds["pmax"], []float64{2, 4, 6})
	if _, ok := bounds["center"]; ok {
		t.Fatal("intermediate bounds should not keep center")
	}
	if _, ok := bounds["size"]; ok {
		t.Fatal("intermediate bounds should not keep size")
	}
}

func TestStudioAdaptsImplicitEquationCenterScaleBasisToTransform(t *testing.T) {
	script := &studioScript{
		Objects: []map[string]interface{}{
			{
				"id":    "expr",
				"shape": "implicit equation",
				"field": map[string]interface{}{
					"type": "expr",
					"expr": "x",
				},
				"center": []interface{}{2, 0, 0},
				"scale":  []interface{}{3, 1, 1},
				"basis": []interface{}{
					[]interface{}{0, 0, 1},
					[]interface{}{0, 1, 0},
					[]interface{}{-1, 0, 0},
				},
				"bounds": map[string]interface{}{
					"pmin": []interface{}{-3, -3, -3},
					"pmax": []interface{}{3, 3, 3},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt implicit equation: %v", err)
	}
	object := adapted.Objects[0]
	if _, ok := object["center"]; ok {
		t.Fatal("implicit equation intermediate object should not keep center")
	}
	if _, ok := object["scale"]; ok {
		t.Fatal("implicit equation intermediate object should not keep scale")
	}
	if _, ok := object["basis"]; ok {
		t.Fatal("implicit equation intermediate object should not keep basis")
	}

	transform, ok := object["transform"].([][]float64)
	if !ok {
		t.Fatalf("expected transform matrix, got %T", object["transform"])
	}
	assertFloatSlice(t, transform[1], []float64{0, 0, 0, 1.0 / 3.0})
	assertFloatSlice(t, transform[2], []float64{0, 0, 1, 0})
	assertFloatSlice(t, transform[3], []float64{2, -1, 0, 0})
}

func TestStudioAdaptsCameraLookAtFromRawFields(t *testing.T) {
	script := &studioScript{}
	cameras := []studioCameraScript{
		{
			Type:        "3d",
			Position:    []float64{-4, 0, 1},
			LookAt:      []float64{0, 0, 0},
			Up:          []float64{0, 0, 1},
			FieldOfView: 60,
		},
	}

	script.Cameras = cameras
	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt camera: %v", err)
	}
	camera := adapted.Cameras[0]
	assertDirectFloatSlice(t, camera.Direction, []float64{4, 0, -1})
}

func TestStudioDoesNotEmitResumeFilmToIntermediateScript(t *testing.T) {
	script := &studioScript{
		Render: studioRenderScript{
			OutputFilm:  "final.bin",
			OutputImage: "final.png",
			ResumeFilm:  "existing.bin",
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if _, ok := adapted.Render["resume_film"]; ok {
		t.Fatal("resume_film must stay in studio and not be emitted to engine intermediate scripts")
	}
	if _, ok := adapted.Render["output_image"]; ok {
		t.Fatal("output_image must stay in studio and not be emitted to engine intermediate scripts")
	}
	if adapted.Render["output_film"] != "final.bin" {
		t.Fatalf("expected output_film to remain in intermediate render config, got %v", adapted.Render["output_film"])
	}
}

func TestStudioEngineArgsDoNotForwardResumeFilm(t *testing.T) {
	config := studioConfig{
		provided: map[string]bool{
			"resume-film": true,
			"output-film": true,
		},
		resumeFilm: "existing.bin",
		outputFilm: "final.bin",
	}

	args := config.engineArgs("intermediate.json", "rendered.bin", 0)
	if containsString(args, "--resume-film") || containsString(args, "existing.bin") {
		t.Fatalf("engine args must not contain resume-film: %v", args)
	}
	if !containsString(args, "--output-film") || !containsString(args, "rendered.bin") {
		t.Fatalf("expected output-film override to point at rendered temp film: %v", args)
	}
	if containsString(args, "final.bin") {
		t.Fatalf("final output film should be written by studio, not engine: %v", args)
	}
}

func TestParseStudioConfigRequiresEndlessCheckpointSettings(t *testing.T) {
	_, err := parseStudioConfig([]string{"--endless", "--checkpoint-dir", "checkpoints"})
	if err == nil {
		t.Fatal("expected endless mode without checkpoint interval to fail")
	}

	_, err = parseStudioConfig([]string{"--endless", "--checkpoint-interval", "100"})
	if err == nil {
		t.Fatal("expected endless mode without checkpoint dir to fail")
	}
}

func TestParseStudioConfigSupportsEndlessResumeCheckpoint(t *testing.T) {
	config, err := parseStudioConfig([]string{
		"--endless",
		"--checkpoint-interval", "100",
		"--checkpoint-dir", "checkpoints",
		"--start-iteration", "300",
		"--resume-film", "checkpoints/iteration-000000300.bin",
	})
	if err != nil {
		t.Fatalf("parse endless config: %v", err)
	}
	if !config.endless || config.checkpointInterval != 100 || config.startIteration != 300 {
		t.Fatalf("unexpected endless config: %+v", config)
	}
}

func TestParseStudioConfigSupportsEngineBin(t *testing.T) {
	config, err := parseStudioConfig([]string{"--engine-bin", "bin/ray"})
	if err != nil {
		t.Fatalf("parse engine bin: %v", err)
	}
	if config.engineBin != "bin/ray" {
		t.Fatalf("unexpected engine bin: %q", config.engineBin)
	}
}

func TestEngineCommandDefaultsToGoRun(t *testing.T) {
	name, args, err := engineCommand(studioConfig{}, "repo")
	if err != nil {
		t.Fatalf("engine command: %v", err)
	}
	if name != "go" {
		t.Fatalf("expected go command, got %q", name)
	}
	expected := []string{"-C", filepath.Join("repo", "engine"), "run", "."}
	if strings.Join(args, "\x00") != strings.Join(expected, "\x00") {
		t.Fatalf("unexpected engine args: %v", args)
	}
}

func TestStudioEngineArgsUsesEndlessSampleOverride(t *testing.T) {
	config := studioConfig{
		provided: map[string]bool{"samples": true},
		samples:  10,
	}

	args := config.engineArgs("intermediate.json", "checkpoint.bin", 100)
	if !containsString(args, "--samples") || !containsString(args, "100") {
		t.Fatalf("expected endless sample override in engine args: %v", args)
	}
	if containsString(args, "10") {
		t.Fatalf("configured samples should not override endless interval: %v", args)
	}
}

func TestCheckpointPathsUseIterationNames(t *testing.T) {
	filmPath, imagePath := checkpointPaths("checkpoints", 100)
	if filepath.Base(filmPath) != "iteration-000000000100.bin" {
		t.Fatalf("unexpected checkpoint film path: %s", filmPath)
	}
	if filepath.Base(imagePath) != "iteration-000000000100.png" {
		t.Fatalf("unexpected checkpoint image path: %s", imagePath)
	}
}

func TestStudioRejectsUnequalHypercubeExtents(t *testing.T) {
	script := &studioScript{
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
	script := &studioScript{
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

func TestStudioAdaptsFourOrderCenterScaleBasisToWorldCoefficients(t *testing.T) {
	script := &studioScript{
		Objects: []map[string]interface{}{
			{
				"id":    "quartic",
				"shape": "four-order equation",
				"a": fourOrderCoefficients(map[[4]int]float64{
					[4]int{1, 1, 1, 1}: 1,
					[4]int{0, 0, 0, 0}: -1,
				}),
				"center": []interface{}{2, 0, 0},
				"scale":  []interface{}{3, 1, 1},
				"basis": []interface{}{
					[]interface{}{0, 0, 1},
					[]interface{}{0, 1, 0},
					[]interface{}{-1, 0, 0},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt four-order equation: %v", err)
	}
	object := adapted.Objects[0]
	if _, ok := object["center"]; ok {
		t.Fatal("four-order intermediate object should not keep center")
	}
	if _, ok := object["scale"]; ok {
		t.Fatal("four-order intermediate object should not keep scale")
	}
	if _, ok := object["basis"]; ok {
		t.Fatal("four-order intermediate object should not keep basis")
	}

	quartic := modelshape.NewFourOrderEquation(mustFloatSlice(t, object["a"]))
	interaction, ok := quartic.IntersectRange(
		mat.NewVecDense(3, []float64{2, 0, -6}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		0,
		math.MaxFloat64,
	)
	if !ok {
		t.Fatal("expected baked four-order equation to hit")
	}
	if math.Abs(interaction.Distance-3) > 1e-8 {
		t.Fatalf("expected hit at distance 3, got %f", interaction.Distance)
	}
	if math.Abs(interaction.GeometricNormal.AtVec(2)+1) > 1e-8 {
		t.Fatalf("expected baked normal to face negative z, got %v", interaction.GeometricNormal.RawVector().Data)
	}
}

func TestStudioAdaptsPolynomialSurfaceCenterScaleBasisToTransform(t *testing.T) {
	script := &studioScript{
		Objects: []map[string]interface{}{
			{
				"id":        "surface",
				"shape":     "polynomial surface",
				"input_dim": 3,
				"center":    []interface{}{2, 0, 0},
				"scale":     []interface{}{3, 1, 1},
				"basis": []interface{}{
					[]interface{}{0, 0, 1},
					[]interface{}{0, 1, 0},
					[]interface{}{-1, 0, 0},
				},
				"coefficients": map[string]interface{}{
					"format": "coo",
					"terms": []interface{}{
						map[string]interface{}{"index": []interface{}{0, 0, 1}, "value": 1},
					},
				},
			},
		},
	}

	adapted, err := adaptScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt polynomial surface: %v", err)
	}
	object := adapted.Objects[0]
	if _, ok := object["center"]; ok {
		t.Fatal("polynomial surface intermediate object should not keep center")
	}
	if _, ok := object["scale"]; ok {
		t.Fatal("polynomial surface intermediate object should not keep scale")
	}
	if _, ok := object["basis"]; ok {
		t.Fatal("polynomial surface intermediate object should not keep basis")
	}

	transform, ok := object["transform"].([][]float64)
	if !ok {
		t.Fatalf("expected transform matrix, got %T", object["transform"])
	}
	assertFloatSlice(t, transform[1], []float64{0, 0, 0, 1.0 / 3.0})
	assertFloatSlice(t, transform[2], []float64{0, 0, 1, 0})
	assertFloatSlice(t, transform[3], []float64{2, -1, 0, 0})
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
	script, err := readStudioScriptFiles(scriptPaths)
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
	if len(adapted.Objects) != 20 {
		t.Fatalf("expected room objects without example geometry, got %d", len(adapted.Objects))
	}
	for _, object := range adapted.Objects {
		if shape, _ := stringField(object, "shape"); strings.EqualFold(shape, "group") {
			t.Fatalf("studio intermediate output must not contain group object: %#v", object)
		}
		if shape, _ := stringField(object, "shape"); strings.EqualFold(shape, "array") {
			t.Fatalf("studio intermediate output must not contain array object: %#v", object)
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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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

func fourOrderCoefficients(values map[[4]int]float64) []interface{} {
	coefficients := make([]interface{}, 256)
	for i := range coefficients {
		coefficients[i] = 0.0
	}
	for index, value := range values {
		coefficients[((index[0]*4+index[1])*4+index[2])*4+index[3]] = value
	}
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

func assertDirectFloatSlice(t *testing.T, values, expected []float64) {
	t.Helper()
	if len(values) != len(expected) {
		t.Fatalf("expected %d values, got %d: %v", len(expected), len(values), values)
	}
	for i := range values {
		if math.Abs(values[i]-expected[i]) > 1e-10 {
			t.Fatalf("index %d: expected %v, got %v", i, expected, values)
		}
	}
}

func assertFloatValue(t *testing.T, raw interface{}, expected float64) {
	t.Helper()
	value, ok := raw.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", raw)
	}
	if math.Abs(value-expected) > 1e-10 {
		t.Fatalf("expected %v, got %v", expected, value)
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
