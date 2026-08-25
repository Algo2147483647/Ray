package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestScriptDecodesTypedMaterialAndObjectSpecs(t *testing.T) {
	data := []byte(`{
		"dimension": 3,
		"materials": [{
			"id": "matte",
			"surface": {"type": "lambert", "albedo": [0.4, 0.5, 0.6]}
		}],
		"objects": [{
			"id": "ball",
			"material_id": "matte",
			"shape": "sphere",
			"center": [0, 0, 0],
			"r": 1
		}]
	}`)
	var script Script
	if err := json.Unmarshal(data, &script); err != nil {
		t.Fatal(err)
	}
	if len(script.Materials) != 1 || script.Materials[0].Surface == nil {
		t.Fatalf("unexpected materials: %+v", script.Materials)
	}
	if script.Materials[0].Surface.Type != SurfaceLambert {
		t.Fatalf("surface type = %q", script.Materials[0].Surface.Type)
	}
	if len(script.Materials[0].Surface.Albedo) == 0 {
		t.Fatal("spectral parameter leaf was not preserved")
	}
	if len(script.Objects) != 1 || script.Objects[0].Shape != ShapeSphere {
		t.Fatalf("unexpected objects: %+v", script.Objects)
	}
	if script.Objects[0].R == nil || *script.Objects[0].R != 1 {
		t.Fatalf("sphere radius = %v", script.Objects[0].R)
	}
}

func TestTypedSpecsRejectUnknownFieldsAndDiscriminators(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		contains string
	}{
		{
			name:     "unknown object field",
			json:     `{"material_id":"m","shape":"sphere","centre":[0,0,0],"r":1}`,
			contains: `unsupported object field "centre"`,
		},
		{
			name:     "unknown shape",
			json:     `{"material_id":"m","shape":"magic"}`,
			contains: `unsupported shape "magic"`,
		},
		{
			name:     "unknown surface field",
			json:     `{"id":"m","surface":{"type":"lambert","colour":[1,1,1]}}`,
			contains: `unsupported surface field "colour"`,
		},
		{
			name:     "unknown surface discriminator",
			json:     `{"id":"m","surface":{"type":"magic"}}`,
			contains: `unsupported surface type "magic"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var target interface{}
			if strings.Contains(test.name, "object") || test.name == "unknown shape" {
				target = &ObjectSpec{}
			} else {
				target = &MaterialSpec{}
			}
			err := json.Unmarshal([]byte(test.json), target)
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("error = %v, want %q", err, test.contains)
			}
		})
	}
}
