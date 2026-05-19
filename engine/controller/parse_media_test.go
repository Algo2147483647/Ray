package controller

import (
	"strings"
	"testing"

	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/material/core"
)

func TestLoadSceneParsesMediaBoundary(t *testing.T) {
	script := &Script{
		Media: map[string]map[string]interface{}{
			"glass": {
				"type": "homogeneous",
				"ior": map[string]interface{}{
					"type": "constant",
					"eta":  1.5,
				},
			},
		},
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
				"id":          "glass-sphere",
				"shape":       "sphere",
				"position":    []interface{}{0.0, 0.0, 0.0},
				"r":           1.0,
				"material_id": "mat",
				"medium_boundary": map[string]interface{}{
					"outside": "air",
					"inside":  "glass",
				},
			},
		},
	}

	scene := model.NewScene()
	if err := LoadSceneFromScript(script, scene); err != nil {
		t.Fatalf("load scene with media: %v", err)
	}
	if scene.ObjectTree.Media == nil {
		t.Fatal("expected media registry")
	}
	glassID, ok := scene.ObjectTree.Media.ID("glass")
	if !ok {
		t.Fatal("expected glass medium id")
	}
	if got := scene.ObjectTree.Objects[0].MediumBoundary.Inside; got != glassID {
		t.Fatalf("unexpected inside medium: got %d want %d", got, glassID)
	}
	if got := scene.ObjectTree.Media.IOR(glassID, core.ShadingContext{}); got != 1.5 {
		t.Fatalf("unexpected glass eta: got %f want 1.5", got)
	}
}

func TestLoadSceneRejectsUnknownBoundaryMedium(t *testing.T) {
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
				"id":          "bad-boundary",
				"shape":       "sphere",
				"position":    []interface{}{0.0, 0.0, 0.0},
				"r":           1.0,
				"material_id": "mat",
				"medium_boundary": map[string]interface{}{
					"inside": "missing",
				},
			},
		},
	}

	err := LoadSceneFromScript(script, model.NewScene())
	if err == nil {
		t.Fatal("expected unknown medium error")
	}
	if !strings.Contains(err.Error(), `unknown inside medium "missing"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
