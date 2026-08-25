package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Algo2147483647/ray/studio/adapt"
	studiofilm "github.com/Algo2147483647/ray/studio/film"
	"github.com/Algo2147483647/ray/studio/schema"
	"github.com/Algo2147483647/ray/studio/storage"
)

func TestStudioSchemaRejectsRemovedFields(t *testing.T) {
	for name, source := range map[string]string{
		"render width":  `{"render":{"film_id":"main","width":800}}`,
		"render output": `{"render":{"film_id":"main","output_film":"old.bin"}}`,
		"bdpt fallback": `{"render":{"bdpt_fallback_policy":"path"}}`,
		"film width":    `{"films":[{"id":"main","camera_id":"camera","width":800}]}`,
		"camera widths": `{"cameras":[{"id":"camera","widths":[800,600]}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			var script schema.StudioScript
			if err := json.Unmarshal([]byte(source), &script); err == nil {
				t.Fatal("expected removed field to be rejected")
			}
		})
	}
}

func TestStudioSchemaValidatesRenderConfiguration(t *testing.T) {
	for name, source := range map[string]string{
		"dimension":          `{"renders":[{"dimension":1}]}`,
		"threads":            `{"renders":[{"thread_num":-1}]}`,
		"samples":            `{"renders":[{"samples":-1}]}`,
		"integrator":         `{"renders":[{"integrator":"magic"}]}`,
		"rgb spectrum alias": `{"renders":[{"spectrum_mode":"rgb"}]}`,
		"spectrum mode":      `{"renders":[{"spectrum_mode":"magic"}]}`,
		"wavelength samples": `{"renders":[{"wavelength_samples":-1}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			var script schema.StudioScript
			if err := json.Unmarshal([]byte(source), &script); err == nil {
				t.Fatal("expected invalid Studio render configuration to fail")
			}
		})
	}
}

func TestIntermediateScriptUsesRenderTarget(t *testing.T) {
	adapted, err := adaptTestScript(&schema.StudioScript{
		Cameras: []schema.StudioCameraScript{{ID: "main", Type: "3d"}},
		Films:   []schema.StudioFilmScript{{ID: "main-film", CameraID: "main", Shape: []int{800, 600}, OutputFilm: "main.bin"}},
		Render:  schema.StudioRenderScript{FilmID: "main-film"},
	}, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	data, err := json.Marshal(adapted)
	if err != nil {
		t.Fatalf("marshal intermediate script: %v", err)
	}
	var intermediate map[string]json.RawMessage
	if err := json.Unmarshal(data, &intermediate); err != nil {
		t.Fatalf("inspect intermediate script: %v", err)
	}
	if _, exists := intermediate["render"]; exists {
		t.Fatal("legacy render field leaked into Engine intermediate script")
	}
	if _, exists := intermediate["renders"]; !exists {
		t.Fatal("Engine intermediate script must contain renders")
	}
	if adapted.Dimension != 3 {
		t.Fatalf("intermediate scene dimension = %d, want 3", adapted.Dimension)
	}
	if _, exists := adapted.Renders[0]["dimension"]; exists {
		t.Fatal("scene dimension leaked into an Engine render job")
	}
	film := intermediateRenderFilm(t, adapted, 0)
	if len(adapted.Renders) != 1 || adapted.Renders[0]["camera_id"] != "main" || film.Shape[0] != 800 || adapted.Renders[0]["output"] != "main.bin" {
		t.Fatalf("unexpected intermediate script: %+v", adapted)
	}
	if len(adapted.Cameras) != 1 || adapted.Cameras[0].ID != "main" {
		t.Fatalf("camera was not emitted independently: %+v", adapted.Cameras)
	}
}

func TestStudioReusesCameraAcrossRenderTargets(t *testing.T) {
	adapted, err := adaptTestScript(&schema.StudioScript{
		Cameras: []schema.StudioCameraScript{{ID: "main", Type: "3d"}},
		Films: []schema.StudioFilmScript{
			{ID: "preview", CameraID: "main", Shape: []int{320, 200}, OutputFilm: "preview.bin"},
			{ID: "final", CameraID: "main", Shape: []int{1920, 1080}, OutputFilm: "final.bin"},
		},
		Renders: []schema.StudioRenderScript{{FilmID: "preview"}, {FilmID: "final"}},
	}, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if len(adapted.Cameras) != 1 || adapted.Cameras[0].ID != "main" {
		t.Fatalf("camera was cloned per Film: %+v", adapted.Cameras)
	}
	if adapted.Renders[0]["camera_id"] != "main" || adapted.Renders[1]["camera_id"] != "main" {
		t.Fatalf("render targets do not share the authored camera: %+v", adapted.Renders)
	}
	assertIntSlice(t, intermediateRenderFilm(t, adapted, 0).Shape, []int{320, 200})
	assertIntSlice(t, intermediateRenderFilm(t, adapted, 1).Shape, []int{1920, 1080})
}

func TestStudioConfiguresSpectralBinCount(t *testing.T) {
	adapted, err := adaptTestScript(&schema.StudioScript{
		Cameras: []schema.StudioCameraScript{{ID: "main", Type: "3d"}},
		Films: []schema.StudioFilmScript{{
			ID: "main-film", CameraID: "main", Shape: []int{2, 2}, SpectralBinCount: 128, OutputFilm: "main.bin",
		}},
		Render: schema.StudioRenderScript{FilmID: "main-film"},
	}, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if got := intermediateRenderFilm(t, adapted, 0).SpectralBinCount; got != 128 {
		t.Fatalf("spectral_bin_count = %d, want 128", got)
	}
}

func TestStudioRejectsInvalidSpectralBinCount(t *testing.T) {
	for _, count := range []int{-1, schema.MaxSpectralBinCount + 1} {
		var script schema.StudioScript
		source := fmt.Sprintf(`{"films":[{"id":"film","camera_id":"camera","shape":[1,1],"spectral_bin_count":%d}]}`, count)
		if err := json.Unmarshal([]byte(source), &script); err == nil {
			t.Fatalf("expected spectral_bin_count %d to be rejected", count)
		}
	}
}

func TestStudioExpandsLegacyRenderDefaultsIntoEveryEngineJob(t *testing.T) {
	adapted, err := adaptTestScript(&schema.StudioScript{
		Render: schema.StudioRenderScript{
			Integrator: "bdpt",
			Samples:    8,
			FilmID:     "test-film",
		},
		Renders: []schema.StudioRenderScript{
			{Samples: 32},
			{},
		},
	}, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if len(adapted.Renders) != 2 {
		t.Fatalf("expected two Engine jobs, got %d", len(adapted.Renders))
	}
	if adapted.Renders[0]["integrator"] != "bdpt" || adapted.Renders[0]["samples"] != int64(32) {
		t.Fatalf("unexpected first Engine job: %v", adapted.Renders[0])
	}
	if adapted.Renders[1]["integrator"] != "bdpt" || adapted.Renders[1]["samples"] != int64(8) {
		t.Fatalf("unexpected second Engine job: %v", adapted.Renders[1])
	}
}

func TestStudioPreservesExplicitWavelengthCount(t *testing.T) {
	adapted, err := adaptTestScript(&schema.StudioScript{
		Render: schema.StudioRenderScript{
			WavelengthSamples: 1,
		},
	}, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if adapted.Renders[0]["wavelength_samples"] != 1 {
		t.Fatalf("wavelength samples = %v, want 1", adapted.Renders[0]["wavelength_samples"])
	}
}

func TestStudioPreservesExplicitWavelengthCountAfterCLIOverrides(t *testing.T) {
	config := studioConfig{
		provided: map[string]bool{
			"wavelength-samples": true,
		},
		wavelengthSamples: 1,
	}
	intermediate := &schema.IntermediateScript{Renders: []map[string]interface{}{{}}}

	config.applyEngineOverrides(intermediate, "", 0)

	if intermediate.Renders[0]["wavelength_samples"] != 1 {
		t.Fatalf("wavelength samples = %v, want 1", intermediate.Renders[0]["wavelength_samples"])
	}
}

func TestStudioRequiresExplicitFilmSelection(t *testing.T) {
	script := &schema.StudioScript{
		Cameras: []schema.StudioCameraScript{{ID: "camera", Type: "3d"}},
		Films:   []schema.StudioFilmScript{{ID: "film", CameraID: "camera", Shape: []int{400, 400}}},
	}
	if _, err := adapt.AdaptScript(script, []string{"scene.json"}, 3); err == nil {
		t.Fatal("expected missing render.film_id to fail")
	}
}

func TestExperimentScriptsUseCurrentStudioSchema(t *testing.T) {
	root := filepath.Join("..", "experiment")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var script schema.StudioScript
		if err := json.Unmarshal(data, &script); err != nil {
			t.Errorf("%s: %v", path, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTriangularPrismScriptCompositionsAdapt(t *testing.T) {
	root := filepath.Join("..", "experiment", "material", "triangular_prism_dispersion")
	for name, files := range map[string][]string{
		"complete scene": {
			filepath.Join(root, "main.json"),
			filepath.Join(root, "prism.json"),
			filepath.Join(root, "prism-scene.json"),
			filepath.Join(root, "moissanite.json"),
			filepath.Join(root, "moissanite-scene.json"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			script, err := storage.ReadStudioScriptFiles(files)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := adapt.AdaptScript(script, files, 3); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFlattenNestedGroupAndInheritFields(t *testing.T) {
	script := &schema.StudioScript{
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
								"shape":  "polynomial",
								"degree": 3,
								"terms": []interface{}{
									map[string]interface{}{"exponents": []interface{}{3, 0, 0}, "coefficient": 1},
									map[string]interface{}{"exponents": []interface{}{0, 0, 0}, "coefficient": -1},
								},
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

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if len(adapted.Objects) != 2 {
		t.Fatalf("expected two flattened objects, got %d", len(adapted.Objects))
	}

	polynomial := adapted.Objects[0]
	if polynomial["id"] != "outer/inner/surface" {
		t.Fatalf("unexpected polynomial id: %v", polynomial["id"])
	}
	if polynomial["material_id"] != "glass" {
		t.Fatalf("expected inherited material_id, got %v", polynomial["material_id"])
	}
	if _, ok := polynomial["center"]; ok {
		t.Fatal("polynomial intermediate object should not keep center")
	}
	if _, ok := polynomial["scale"]; ok {
		t.Fatal("polynomial intermediate object should not keep scale")
	}
	transform, ok := polynomial["transform"].([][]float64)
	if !ok {
		t.Fatalf("expected polynomial transform, got %T", polynomial["transform"])
	}
	assertFloatSlice(t, transform[1], []float64{-5.0 / 3.0, 1.0 / 3.0, 0, 0})
	assertFloatSlice(t, transform[2], []float64{0, 0, 1.0 / 6.0, 0})

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

func TestGroupBasisRotatesTriangleAndComposesCenter(t *testing.T) {
	script := &schema.StudioScript{
		Objects: []map[string]interface{}{
			{
				"id":     "rotated",
				"shape":  "group",
				"center": []interface{}{2, 3, 4},
				"basis": []interface{}{
					[]interface{}{0, -1, 0},
					[]interface{}{1, 0, 0},
					[]interface{}{0, 0, 1},
				},
				"objects": []interface{}{
					map[string]interface{}{
						"id": "facet", "shape": "triangle",
						"p1":          []interface{}{1, 0, 0},
						"p2":          []interface{}{0, 1, 0},
						"p3":          []interface{}{0, 0, 1},
						"material_id": "glass",
					},
				},
			},
		},
	}

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	facet := adapted.Objects[0]
	for key, want := range map[string][]float64{
		"p1": {2, 4, 4},
		"p2": {1, 3, 4},
		"p3": {2, 3, 5},
	} {
		got, ok := facet[key].([]float64)
		if !ok || len(got) != len(want) {
			t.Fatalf("%s: expected vector, got %T %v", key, facet[key], facet[key])
		}
		for axis := range want {
			if math.Abs(got[axis]-want[axis]) > 1e-10 {
				t.Fatalf("%s[%d]: expected %g, got %g", key, axis, want[axis], got[axis])
			}
		}
	}
}

func TestChildFieldOverridesGroupInheritance(t *testing.T) {
	script := &schema.StudioScript{
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

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if adapted.Objects[0]["material_id"] != "inner" {
		t.Fatalf("expected child material override, got %v", adapted.Objects[0]["material_id"])
	}
}

func TestGroupDoesNotRequireMaterialID(t *testing.T) {
	script := &schema.StudioScript{
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

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
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
	script := &schema.StudioScript{
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

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
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

	script, err := storage.ReadStudioScriptFiles([]string{firstPath, secondPath})
	if err != nil {
		t.Fatalf("read merged studio scripts: %v", err)
	}
	if len(script.Objects) != 1 {
		t.Fatalf("expected one merged array, got %d", len(script.Objects))
	}

	adapted, err := adaptTestScript(script, []string{firstPath, secondPath}, 3)
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
	      "center": [2, 0, 0],
	      "objects": [
	        { "id": "right", "shape": "sphere", "center": [1, 0, 0], "r": 0.25, "material_id": "mat" }
	      ]
	    }
	  ]
	}`), 0o644); err != nil {
		t.Fatalf("write second group script: %v", err)
	}

	script, err := storage.ReadStudioScriptFiles([]string{firstPath, secondPath})
	if err != nil {
		t.Fatalf("read merged group scripts: %v", err)
	}
	adapted, err := adaptTestScript(script, []string{firstPath, secondPath}, 3)
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

func TestStudioBindsTopLevelGroupFragmentToNestedObject(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.json")
	scenePath := filepath.Join(dir, "scene.json")
	if err := os.WriteFile(modelPath, []byte(`{
	  "objects": [{
	    "id": "stone", "shape": "group", "material_id": "glass",
	    "objects": [{ "id": "facet", "shape": "sphere", "center": [0, 0, 0], "r": 0.25 }]
	  }]
	}`), 0o644); err != nil {
		t.Fatalf("write model script: %v", err)
	}
	if err := os.WriteFile(scenePath, []byte(`{
	  "objects": [{
	    "id": "rig", "shape": "group", "center": [2, 0, 0],
	    "objects": [
	      { "id": "stone", "shape": "group" },
	      { "id": "marker", "shape": "sphere", "center": [1, 0, 0], "r": 0.1 }
	    ]
	  }]
	}`), 0o644); err != nil {
		t.Fatalf("write scene script: %v", err)
	}

	for name, paths := range map[string][]string{
		"model before scene": {modelPath, scenePath},
		"scene before model": {scenePath, modelPath},
	} {
		t.Run(name, func(t *testing.T) {
			script, err := storage.ReadStudioScriptFiles(paths)
			if err != nil {
				t.Fatalf("read scripts: %v", err)
			}
			if len(script.Objects) != 1 || script.Objects[0]["id"] != "rig" {
				t.Fatalf("expected only the bound rig at top level, got %v", script.Objects)
			}
			adapted, err := adaptTestScript(script, paths, 3)
			if err != nil {
				t.Fatalf("adapt bound fragment: %v", err)
			}
			if len(adapted.Objects) != 2 {
				t.Fatalf("expected bound facet and marker, got %d objects", len(adapted.Objects))
			}
			if adapted.Objects[0]["id"] != "rig/stone/facet" || adapted.Objects[1]["id"] != "rig/marker" {
				t.Fatalf("unexpected bound ids: %v, %v", adapted.Objects[0]["id"], adapted.Objects[1]["id"])
			}
			assertFloatSlice(t, adapted.Objects[0]["center"], []float64{2, 0, 0})
		})
	}
}

func TestStudioMergesDuplicateNestedGroupsInOneObjectList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scene.json")
	if err := os.WriteFile(path, []byte(`{
	  "objects": [{
	    "id": "rig", "shape": "group",
	    "objects": [
	      { "id": "part", "shape": "group", "objects": [
	        { "id": "left", "shape": "sphere", "center": [0, 0, 0], "r": 0.1 }
	      ]},
	      { "id": "part", "shape": "group", "objects": [
	        { "id": "right", "shape": "sphere", "center": [1, 0, 0], "r": 0.1 }
	      ]}
	    ]
	  }]
	}`), 0o644); err != nil {
		t.Fatalf("write scene script: %v", err)
	}

	script, err := storage.ReadStudioScriptFiles([]string{path})
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	adapted, err := adaptTestScript(script, []string{path}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if len(adapted.Objects) != 2 || adapted.Objects[0]["id"] != "rig/part/left" || adapted.Objects[1]["id"] != "rig/part/right" {
		t.Fatalf("unexpected normalized objects: %v", adapted.Objects)
	}
}

func TestStudioRejectsConflictingGroupParametersAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "group-a.json")
	secondPath := filepath.Join(dir, "group-b.json")
	if err := os.WriteFile(firstPath, []byte(`{
	  "objects": [{ "id": "rig", "shape": "group", "center": [0, 0, 0] }]
	}`), 0o644); err != nil {
		t.Fatalf("write first group script: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte(`{
	  "objects": [{ "id": "rig", "shape": "group", "center": [1, 0, 0] }]
	}`), 0o644); err != nil {
		t.Fatalf("write second group script: %v", err)
	}

	_, err := storage.ReadStudioScriptFiles([]string{firstPath, secondPath})
	if err == nil || !strings.Contains(err.Error(), `object id "rig" has conflicting field "center"`) {
		t.Fatalf("expected group parameter conflict, got %v", err)
	}
}

func TestStudioRejectsConflictingArrayParametersAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "array-a.json")
	secondPath := filepath.Join(dir, "array-b.json")
	if err := os.WriteFile(firstPath, []byte(`{
	  "objects": [{ "id": "grid", "shape": "array", "counts": [2] }]
	}`), 0o644); err != nil {
		t.Fatalf("write first array script: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte(`{
	  "objects": [{ "id": "grid", "shape": "array", "counts": [3] }]
	}`), 0o644); err != nil {
		t.Fatalf("write second array script: %v", err)
	}

	_, err := storage.ReadStudioScriptFiles([]string{firstPath, secondPath})
	if err == nil || !strings.Contains(err.Error(), `object id "grid" has conflicting field "counts"`) {
		t.Fatalf("expected array parameter conflict, got %v", err)
	}
}

func TestStudioRejectsObjectIDWithConflictingParents(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "parent-a.json")
	secondPath := filepath.Join(dir, "parent-b.json")
	if err := os.WriteFile(firstPath, []byte(`{
	  "objects": [{
	    "id": "parent-a", "shape": "group",
	    "objects": [{ "id": "shared", "shape": "sphere", "center": [0, 0, 0], "r": 1 }]
	  }]
	}`), 0o644); err != nil {
		t.Fatalf("write first parent script: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte(`{
	  "objects": [{
	    "id": "parent-b", "shape": "group",
	    "objects": [{ "id": "shared", "shape": "sphere", "center": [0, 0, 0], "r": 1 }]
	  }]
	}`), 0o644); err != nil {
		t.Fatalf("write second parent script: %v", err)
	}

	_, err := storage.ReadStudioScriptFiles([]string{firstPath, secondPath})
	if err == nil || !strings.Contains(err.Error(), `object id "shared" has conflicting parents`) {
		t.Fatalf("expected parent conflict, got %v", err)
	}
}

func TestStudioAdaptsTriangleCenterAndGroupPlacement(t *testing.T) {
	script := &schema.StudioScript{
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

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
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

func TestStudioExpandsQuadrilateralIntoTwoTriangles(t *testing.T) {
	script := &schema.StudioScript{
		Objects: []map[string]interface{}{
			{
				"id":          "panel-group",
				"shape":       "group",
				"center":      []interface{}{10, 0, 0},
				"scale":       2,
				"material_id": "mat",
				"objects": []interface{}{
					map[string]interface{}{
						"id":     "panel",
						"shape":  "quadrilateral",
						"center": []interface{}{1, 1, 1},
						"p1":     []interface{}{0, 0, 0},
						"p2":     []interface{}{1, 0, 0},
						"p3":     []interface{}{1, 1, 0},
						"p4":     []interface{}{0, 1, 0},
					},
				},
			},
		},
	}

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt quadrilateral: %v", err)
	}
	if len(adapted.Objects) != 2 {
		t.Fatalf("expected two triangles, got %d objects", len(adapted.Objects))
	}

	first, second := adapted.Objects[0], adapted.Objects[1]
	if first["shape"] != "triangle" || second["shape"] != "triangle" {
		t.Fatalf("engine objects must both be triangles, got %v and %v", first["shape"], second["shape"])
	}
	if first["id"] != "panel-group/panel/triangle-1" || second["id"] != "panel-group/panel/triangle-2" {
		t.Fatalf("unexpected triangle ids: %v, %v", first["id"], second["id"])
	}
	assertFloatSlice(t, first["p1"], []float64{12, 2, 2})
	assertFloatSlice(t, first["p2"], []float64{14, 2, 2})
	assertFloatSlice(t, first["p3"], []float64{14, 4, 2})
	assertFloatSlice(t, second["p1"], []float64{14, 4, 2})
	assertFloatSlice(t, second["p2"], []float64{12, 4, 2})
	assertFloatSlice(t, second["p3"], []float64{12, 2, 2})
	for _, triangle := range adapted.Objects {
		if triangle["material_id"] != "mat" {
			t.Fatalf("triangle did not inherit material: %v", triangle["material_id"])
		}
		if _, ok := triangle["p4"]; ok {
			t.Fatal("engine triangle must not contain p4")
		}
		if _, ok := triangle["center"]; ok {
			t.Fatal("engine triangle must not contain center")
		}
	}
}

func TestStudioQuadrilateralRequiresP4(t *testing.T) {
	script := &schema.StudioScript{
		Objects: []map[string]interface{}{
			{
				"id":    "panel",
				"shape": "quadrilateral",
				"p1":    []interface{}{0, 0, 0},
				"p2":    []interface{}{1, 0, 0},
				"p3":    []interface{}{1, 1, 0},
			},
		},
	}

	_, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err == nil || !strings.Contains(err.Error(), `missing required field "p4"`) {
		t.Fatalf("expected missing p4 error, got %v", err)
	}
}

func TestStudioAdaptsBasicShapesWithGroupPlacement(t *testing.T) {
	script := &schema.StudioScript{
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

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
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
	script := &schema.StudioScript{
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

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
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
	script := &schema.StudioScript{
		Objects: []map[string]interface{}{
			{
				"id":     "cube",
				"shape":  "hypercube",
				"center": []interface{}{1, 2, 3, 4},
				"size":   []interface{}{2, 2, 2, 2},
			},
		},
	}

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 4)
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
	script := &schema.StudioScript{
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

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
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
	script := &schema.StudioScript{
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

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
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

func TestStudioNormalizesImplicitEquationFieldAlias(t *testing.T) {
	script := &schema.StudioScript{
		Objects: []map[string]interface{}{
			{
				"id":    "lp",
				"shape": "implicit equation",
				"field": map[string]interface{}{
					"type":   "lp_norm",
					"power":  1,
					"radius": 1,
				},
				"bounds": map[string]interface{}{
					"pmin": []interface{}{-1, -1, -1},
					"pmax": []interface{}{1, 1, 1},
				},
			},
		},
	}

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt implicit equation alias: %v", err)
	}
	field, ok := adapted.Objects[0]["field"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected field object, got %T", adapted.Objects[0]["field"])
	}
	if field["type"] != "lp_power_sum" {
		t.Fatalf("expected lp_norm alias to normalize to lp_power_sum, got %v", field["type"])
	}
}

func TestStudioAdaptsCameraLookAtFromRawFields(t *testing.T) {
	script := &schema.StudioScript{}
	cameras := []schema.StudioCameraScript{
		{
			Type:        "3d",
			Position:    []float64{-4, 0, 1},
			LookAt:      []float64{0, 0, 0},
			Up:          []float64{0, 0, 1},
			FieldOfView: 60,
		},
	}

	script.Cameras = cameras
	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt camera: %v", err)
	}
	camera := adapted.Cameras[0]
	if len(camera.Coordinates) != 3 {
		t.Fatalf("expected three canonical camera coordinates, got %v", camera.Coordinates)
	}
	assertDirectFloatSlice(t, camera.Coordinates[0], []float64{4 / math.Sqrt(17), 0, -1 / math.Sqrt(17)})
}

func TestStudioDoesNotEmitResumeFilmToIntermediateScript(t *testing.T) {
	script := &schema.StudioScript{
		Render: schema.StudioRenderScript{FilmID: "test-film"},
		Films: []schema.StudioFilmScript{{
			ID: "test-film", CameraID: "test-camera", Shape: []int{400, 400},
			OutputFilm: "final.bin", OutputImage: "final.png", ResumeFilm: "existing.bin",
		}},
	}

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if _, ok := adapted.Renders[0]["resume_film"]; ok {
		t.Fatal("resume_film must stay in studio and not be emitted to engine intermediate scripts")
	}
	if _, ok := adapted.Renders[0]["output_image"]; ok {
		t.Fatal("output_image must stay in studio and not be emitted to engine intermediate scripts")
	}
	if adapted.Renders[0]["output"] != "final.bin" {
		t.Fatalf("expected output on the Engine render target, got %v", adapted.Renders[0]["output"])
	}
}

func TestStudioKeepsColorPipelineOutOfEngineIntermediateScript(t *testing.T) {
	script := &schema.StudioScript{
		Render: schema.StudioRenderScript{FilmID: "test-film"},
		Films: []schema.StudioFilmScript{{
			ID: "test-film", CameraID: "test-camera", Shape: []int{400, 400},
			Exposure: 0.75, ToneMapping: "aces", Gamma: 2.2, ColorSpace: "acescg",
		}},
	}
	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	for _, field := range []string{"exposure", "tone_mapping", "tanh_omega", "gamma", "color_space", "working_space"} {
		if _, ok := adapted.Renders[0][field]; ok {
			t.Fatalf("Studio-only field %q leaked into Engine script", field)
		}
	}
}

func TestStudioEmitsIntegratorToIntermediateScript(t *testing.T) {
	script := &schema.StudioScript{
		Render: schema.StudioRenderScript{Integrator: "bdpt"},
	}
	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	if adapted.Renders[0]["integrator"] != "bdpt" {
		t.Fatalf("expected bdpt integrator in intermediate script, got %v", adapted.Renders[0]["integrator"])
	}
}

func TestStudioEmitsPixelWindowsToIntermediateScript(t *testing.T) {
	script := &schema.StudioScript{
		Render: schema.StudioRenderScript{FilmID: "test-film"},
		Films: []schema.StudioFilmScript{{
			ID: "test-film", CameraID: "test-camera", Shape: []int{400, 400},
			PixelWindows: []schema.PixelWindowScript{
				{Min: []int{100, 600}, Max: []int{150, 650}},
			},
		}},
	}

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}

	windows := intermediateRenderFilm(t, adapted, 0).PixelWindows
	if len(windows) != 1 {
		t.Fatalf("expected one pixel window, got %d", len(windows))
	}
	assertIntSlice(t, windows[0].Min, []int{100, 600})
	assertIntSlice(t, windows[0].Max, []int{150, 650})
}

func TestStudioEngineArgsOnlyPassScriptPath(t *testing.T) {
	config := studioConfig{
		provided: map[string]bool{
			"resume-film": true,
			"output-film": true,
		},
		resumeFilm: "existing.bin",
		outputFilm: "final.bin",
	}

	args := config.engineArgs("intermediate.json")
	if len(args) != 2 || args[0] != "--script" || args[1] != "intermediate.json" {
		t.Fatalf("Engine must receive only the script path: %v", args)
	}
}

func TestStudioEngineArgsDoNotForwardColorPipeline(t *testing.T) {
	config := studioConfig{
		provided: map[string]bool{
			"exposure": true, "tone-mapping": true, "tanh-omega": true, "gamma": true, "color-space": true,
		},
		exposure: 0.75, toneMapping: "spectral_tanh", tanhOmega: 2, gamma: 2.2, colorSpace: "acescg",
	}
	args := config.engineArgs("intermediate.json")
	for _, flag := range []string{"--exposure", "--tone-mapping", "--tanh-omega", "--gamma", "--color-space"} {
		if containsString(args, flag) {
			t.Fatalf("Studio-only flag %q leaked into Engine args: %v", flag, args)
		}
	}
}

func TestStudioEmbedsFilmShapeOverrideInIntermediateScript(t *testing.T) {
	config := studioConfig{
		provided: map[string]bool{"width": true, "height": true},
		width:    1920,
		height:   1080,
	}
	intermediate := &schema.IntermediateScript{Renders: []map[string]interface{}{{"film": schema.EngineFilmScript{}}}}
	config.applyEngineOverrides(intermediate, "", 0)
	assertIntSlice(t, intermediateRenderFilm(t, intermediate, 0).Shape, []int{1920, 1080})
}

func TestStudioEmbedsRenderOverridesInIntermediateScript(t *testing.T) {
	config := studioConfig{
		provided: map[string]bool{
			"integrator":         true,
			"camera-id":          true,
			"threads":            true,
			"samples":            true,
			"output-film":        true,
			"wavelength-samples": true,
		},
		integrator:        "bdpt",
		cameraID:          "camera-override",
		threadNum:         6,
		samples:           48,
		outputFilm:        "override.bin",
		wavelengthSamples: 8,
	}
	intermediate := &schema.IntermediateScript{
		Renders: []map[string]interface{}{{}},
		Cameras: []schema.EngineCameraScript{{}},
	}

	config.applyEngineOverrides(intermediate, "", 0)

	render := intermediate.Renders[0]
	if render["integrator"] != "bdpt" || render["camera_id"] != "camera-override" ||
		render["thread_num"] != 6 || render["samples"] != int64(48) ||
		render["wavelength_samples"] != 8 {
		t.Fatalf("render overrides were not embedded in JSON: %v", render)
	}
	if intermediate.Renders[0]["output"] != "override.bin" {
		t.Fatalf("target output override was not embedded in JSON: %+v", intermediate.Renders[0])
	}
}

func TestStudioAcceptsLegacyEngineRenderFlags(t *testing.T) {
	config, err := parseStudioConfig([]string{
		"--integrator", "bdpt",
		"--camera-id", "main",
		"--widths", "16,12,8",
	})
	if err != nil {
		t.Fatalf("parse legacy Engine flags in Studio: %v", err)
	}
	if config.integrator != "bdpt" || config.cameraID != "main" {
		t.Fatalf("unexpected render overrides: %+v", config)
	}
	assertIntSlice(t, config.widths, []int{16, 12, 8})
}

func TestStudioAdaptAttachesFilmShapeToRenderTarget(t *testing.T) {
	adapted, err := adaptTestScript(&schema.StudioScript{
		Render:  schema.StudioRenderScript{FilmID: "test-film"},
		Films:   []schema.StudioFilmScript{{ID: "test-film", CameraID: "test-camera", Shape: []int{1280, 720}}},
		Cameras: []schema.StudioCameraScript{{ID: "test-camera"}},
	}, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt script: %v", err)
	}
	assertIntSlice(t, intermediateRenderFilm(t, adapted, 0).Shape, []int{1280, 720})
	if adapted.Renders[0]["camera_id"] != "test-camera" {
		t.Fatalf("render camera_id = %v, want test-camera", adapted.Renders[0]["camera_id"])
	}
	if _, exists := adapted.Renders[0]["width"]; exists {
		t.Fatal("legacy width leaked into canonical Engine script")
	}
	if _, exists := adapted.Renders[0]["height"]; exists {
		t.Fatal("legacy height leaked into canonical Engine script")
	}
}

func adaptTestScript(script *schema.StudioScript, source []string, dimension int) (*schema.IntermediateScript, error) {
	if len(script.Cameras) == 0 {
		script.Cameras = []schema.StudioCameraScript{{ID: "test-camera", Type: "n_dim"}}
	} else if script.Cameras[0].ID == "" {
		script.Cameras[0].ID = "test-camera"
	}
	if len(script.Films) == 0 {
		script.Films = []schema.StudioFilmScript{{
			ID: "test-film", CameraID: script.Cameras[0].ID, Shape: []int{400, 400},
		}}
	}
	if script.Render.FilmID == "" {
		script.Render.FilmID = script.Films[0].ID
	}
	return adapt.AdaptScript(script, source, dimension)
}

func intermediateRenderFilm(t testing.TB, script *schema.IntermediateScript, index int) schema.EngineFilmScript {
	t.Helper()
	if script == nil || index < 0 || index >= len(script.Renders) {
		t.Fatalf("render index %d does not exist", index)
	}
	film, ok := script.Renders[index]["film"].(schema.EngineFilmScript)
	if !ok {
		t.Fatalf("render[%d] film has unexpected type %T", index, script.Renders[index]["film"])
	}
	return film
}

func TestStudioOwnsColorAndWavelengthSampleCLI(t *testing.T) {
	config, err := parseStudioConfig([]string{"--color-space", "acescg", "--wavelength-samples", "4"})
	if err != nil {
		t.Fatalf("parse Studio color pipeline: %v", err)
	}
	if config.colorSpace != "acescg" || config.wavelengthSamples != 4 {
		t.Fatalf("unexpected Studio color pipeline: %+v", config)
	}
	if _, err := parseStudioConfig([]string{"--working-space", "acescg"}); err == nil {
		t.Fatal("expected removed working-space alias to fail")
	}
	if _, err := parseStudioConfig([]string{"--spectrum-mode", "sampled"}); err == nil {
		t.Fatal("expected removed spectrum-mode flag to fail")
	}
}

func TestStudioParsesStandaloneFilmConversionConfig(t *testing.T) {
	config, err := parseStudioConfig([]string{
		"--input-film", filepath.Join("renders", "beauty.BIN"),
		"--exposure", "1.25",
		"--tone-mapping", "aces",
		"--gamma", "2.2",
		"--color-space", "acescg",
	})
	if err != nil {
		t.Fatalf("parse Film conversion config: %v", err)
	}
	if len(config.scriptPaths) != 0 {
		t.Fatalf("Film conversion unexpectedly resolved scene scripts: %v", config.scriptPaths)
	}
	if got, want := config.filmConversionImagePath(), filepath.Join("renders", "beauty.png"); got != want {
		t.Fatalf("conversion image path = %q, want %q", got, want)
	}
	if config.exposure != 1.25 || config.toneMapping != "aces" || config.gamma != 2.2 || config.colorSpace != "acescg" {
		t.Fatalf("unexpected Film conversion options: %+v", config)
	}
}

func TestStudioRejectsRenderInputsInFilmConversionMode(t *testing.T) {
	for name, args := range map[string][]string{
		"scene flag":       {"--input-film", "film.bin", "--script", "scene.json"},
		"positional scene": {"--input-film", "film.bin", "scene.json"},
		"render flag":      {"--input-film", "film.bin", "--samples", "4"},
		"empty output":     {"--input-film", "film.bin", "--output-image", ""},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseStudioConfig(args); err == nil {
				t.Fatal("expected incompatible Film conversion config to fail")
			}
		})
	}
}

func TestStudioParsesSpectralTanhFilmConversion(t *testing.T) {
	config, err := parseStudioConfig([]string{
		"--input-film", "render.bin",
		"--tone-mapping", "spectral_tanh",
		"--tanh-omega", "3.5",
	})
	if err != nil {
		t.Fatalf("parse spectral tanh conversion: %v", err)
	}
	if config.toneMapping != "spectral_tanh" || config.tanhOmega != 3.5 {
		t.Fatalf("unexpected spectral tanh options: %+v", config)
	}
	if _, err := parseStudioConfig([]string{"--input-film", "render.bin", "--tanh-omega", "0"}); err == nil {
		t.Fatal("expected non-positive tanh omega to fail")
	}
}

func TestRunFilmConversionWritesConfiguredPNG(t *testing.T) {
	dir := t.TempDir()
	filmPath := filepath.Join(dir, "render.bin")
	imagePath := filepath.Join(dir, "image.png")
	film := studiofilm.NewFilm(2, 1)
	film.InitSpectralBins(64, 380, 750)
	film.Samples = 1
	for bin := range film.SpectralBins {
		for pixel := range film.SpectralBins[bin].Data {
			film.SpectralBins[bin].Data[pixel] = 1.0 / 64
		}
	}
	if err := film.SaveToFile(filmPath); err != nil {
		t.Fatalf("save input Film: %v", err)
	}

	config, err := parseStudioConfig([]string{
		"--input-film", filmPath,
		"--output-image", imagePath,
		"--tone-mapping", "reinhard",
		"--gamma", "2.2",
	})
	if err != nil {
		t.Fatalf("parse Film conversion config: %v", err)
	}
	if err := runFilmConversion(config); err != nil {
		t.Fatalf("convert Film to PNG: %v", err)
	}

	file, err := os.Open(imagePath)
	if err != nil {
		t.Fatalf("open output PNG: %v", err)
	}
	defer file.Close()
	decoded, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatalf("decode output PNG: %v", err)
	}
	if decoded.Width != 2 || decoded.Height != 1 {
		t.Fatalf("output PNG size = %dx%d, want 2x1", decoded.Width, decoded.Height)
	}
}

func TestStudioResolvesPerRenderColorPipeline(t *testing.T) {
	film := schema.StudioFilmScript{
		Exposure: 0.5, ToneMapping: "spectral_tanh", TanhOmega: 3.5, Gamma: 2.4, ColorSpace: "acescg",
	}
	output := studioRenderOutputFromFilm(film, studioConfig{provided: map[string]bool{}}, "")
	if output.Options.Exposure != 0.5 || output.Options.Gamma != 2.4 ||
		output.Options.TanhOmega != 3.5 || string(output.Options.ToneMapping) != "spectral_tanh" ||
		string(output.Options.ColorSpace) != "acescg" {
		t.Fatalf("unexpected Studio color pipeline: %+v", output.Options)
	}
}

func TestStudioEmbedsPixelWindowsInIntermediateScript(t *testing.T) {
	config := studioConfig{
		provided: map[string]bool{"pixel-window": true},
		pixelWindows: []schema.PixelWindowScript{
			{Min: []int{100, 600}, Max: []int{150, 650}},
			{Min: []int{2, 6}, Max: []int{4, 8}},
		},
	}

	intermediate := &schema.IntermediateScript{Renders: []map[string]interface{}{{"film": schema.EngineFilmScript{}}}}
	config.applyEngineOverrides(intermediate, "", 0)
	windows := intermediateRenderFilm(t, intermediate, 0).PixelWindows
	if len(windows) != 2 {
		t.Fatalf("expected two embedded pixel windows, got %v", windows)
	}
	assertIntSlice(t, windows[0].Min, []int{100, 600})
	assertIntSlice(t, windows[1].Max, []int{4, 8})
}

func TestParseStudioConfigAcceptsPixelWindows(t *testing.T) {
	config, err := parseStudioConfig([]string{
		"--pixel-window", "100-150,600-650",
		"--pixel-window", "2:4,6:8",
	})
	if err != nil {
		t.Fatalf("parse studio config: %v", err)
	}

	if len(config.pixelWindows) != 2 {
		t.Fatalf("expected two pixel windows, got %d", len(config.pixelWindows))
	}
	assertIntSlice(t, config.pixelWindows[0].Min, []int{100, 600})
	assertIntSlice(t, config.pixelWindows[0].Max, []int{150, 650})
	assertIntSlice(t, config.pixelWindows[1].Min, []int{2, 6})
	assertIntSlice(t, config.pixelWindows[1].Max, []int{4, 8})
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

func TestStudioEmbedsEndlessSampleAndFilmOverrides(t *testing.T) {
	config := studioConfig{
		provided: map[string]bool{"samples": true},
		samples:  10,
	}

	intermediate := &schema.IntermediateScript{
		Renders: []map[string]interface{}{{}},
		Cameras: []schema.EngineCameraScript{{}},
	}
	config.applyEngineOverrides(intermediate, "checkpoint.bin", 100)
	if intermediate.Renders[0]["samples"] != int64(100) {
		t.Fatalf("expected endless sample override in JSON: %v", intermediate.Renders[0])
	}
	if intermediate.Renders[0]["output"] != "checkpoint.bin" {
		t.Fatalf("expected checkpoint output override in JSON: %+v", intermediate.Renders[0])
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
	script := &schema.StudioScript{
		Objects: []map[string]interface{}{
			{
				"id":     "bad-cube",
				"shape":  "hypercube",
				"center": []interface{}{0, 0, 0},
				"size":   []interface{}{2, 3, 2},
			},
		},
	}

	if _, err := adaptTestScript(script, []string{"scene.json"}, 3); err == nil {
		t.Fatal("expected unequal hypercube extents to fail")
	}
}

func TestStudioRejectsLegacyPolynomialShapeKinds(t *testing.T) {
	for _, kind := range []string{"quadratic equation", "cubic equation", "four-order equation", "polynomial surface"} {
		script := &schema.StudioScript{Objects: []map[string]interface{}{{"id": "legacy", "shape": kind}}}
		if _, err := adaptTestScript(script, []string{"scene.json"}, 3); err == nil {
			t.Fatalf("expected legacy polynomial shape %q to fail", kind)
		}
	}
}

func TestStudioAdaptsPolynomialCenterScaleBasisToTransform(t *testing.T) {
	script := &schema.StudioScript{
		Objects: []map[string]interface{}{
			{
				"id":     "surface",
				"shape":  "polynomial",
				"degree": 1,
				"terms": []interface{}{
					map[string]interface{}{"exponents": []interface{}{0, 0, 1}, "coefficient": 1},
				},
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

	adapted, err := adaptTestScript(script, []string{"scene.json"}, 3)
	if err != nil {
		t.Fatalf("adapt polynomial: %v", err)
	}
	object := adapted.Objects[0]
	if _, ok := object["center"]; ok {
		t.Fatal("polynomial intermediate object should not keep center")
	}
	if _, ok := object["scale"]; ok {
		t.Fatal("polynomial intermediate object should not keep scale")
	}
	if _, ok := object["basis"]; ok {
		t.Fatal("polynomial intermediate object should not keep basis")
	}

	transform, ok := object["transform"].([][]float64)
	if !ok {
		t.Fatalf("expected transform matrix, got %T", object["transform"])
	}
	assertFloatSlice(t, transform[1], []float64{0, 0, 0, 1.0 / 3.0})
	assertFloatSlice(t, transform[2], []float64{0, 0, 1, 0})
	assertFloatSlice(t, transform[3], []float64{2, -1, 0, 0})
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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

func assertIntSlice(t *testing.T, values, expected []int) {
	t.Helper()
	if len(values) != len(expected) {
		t.Fatalf("expected %d values, got %d: %v", len(expected), len(values), values)
	}
	for i := range values {
		if values[i] != expected[i] {
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
