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
	if script.Materials[0].Surface.Type != SurfaceLambert.Kind {
		t.Fatalf("surface type = %q", script.Materials[0].Surface.Type)
	}
	lambert, ok := script.Materials[0].Surface.Definition.(*LambertSurfaceSpec)
	if !ok || len(lambert.Albedo) == 0 {
		t.Fatal("spectral parameter leaf was not preserved")
	}
	if len(script.Objects) != 1 || script.Objects[0].Shape != ShapeSphere.Kind {
		t.Fatalf("unexpected objects: %+v", script.Objects)
	}
	sphere, ok := script.Objects[0].Definition.(*SphereSpec)
	if !ok || sphere.R == nil || *sphere.R != 1 {
		t.Fatalf("sphere definition = %#v", script.Objects[0].Definition)
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

func TestTypedSpecsMarshalAsFlatDiscriminatedObjects(t *testing.T) {
	var object ObjectSpec
	if err := json.Unmarshal([]byte(`{"id":"ball","material_id":"matte","shape":"sphere","center":[1,2,3],"r":2}`), &object); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if !strings.Contains(text, `"shape":"sphere"`) || !strings.Contains(text, `"center":[1,2,3]`) || strings.Contains(text, "Definition") {
		t.Fatalf("object encoded as %s", text)
	}
	var objectRoundTrip ObjectSpec
	if err := json.Unmarshal(encoded, &objectRoundTrip); err != nil {
		t.Fatal(err)
	}
	if _, ok := objectRoundTrip.Definition.(*SphereSpec); !ok {
		t.Fatalf("round-trip definition = %T", objectRoundTrip.Definition)
	}

	var material MaterialSpec
	if err := json.Unmarshal([]byte(`{"id":"matte","surface":{"type":"lambert","albedo":[0.2,0.3,0.4]}}`), &material); err != nil {
		t.Fatal(err)
	}
	encoded, err = json.Marshal(material)
	if err != nil {
		t.Fatal(err)
	}
	text = string(encoded)
	if !strings.Contains(text, `"type":"lambert"`) || !strings.Contains(text, `"albedo":[0.2,0.3,0.4]`) || strings.Contains(text, "Definition") {
		t.Fatalf("material encoded as %s", text)
	}
}

func TestMaterialVariantsRejectFieldsFromOtherVariants(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		contains string
	}{
		{"lambert rejects conductor field", `{"type":"lambert","albedo":[1,1,1],"roughness":0.2}`, `unsupported surface field "roughness"`},
		{"conductor rejects dielectric field", `{"type":"rough_conductor","eta":[1,1,1],"k":[1,1,1],"eta_inside":1.5}`, `unsupported surface field "eta_inside"`},
		{"constant emission rejects palette", `{"type":"constant","color":[1,1,1],"palette":[[1,0,0]]}`, `unsupported emission field "palette"`},
		{"normal palette rejects radiance", `{"type":"normal_palette","radiance":[1,1,1]}`, `unsupported emission field "radiance"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			if strings.Contains(test.data, `"type":"constant"`) || strings.Contains(test.data, `"type":"normal_palette"`) {
				var spec EmissionSpec
				err = json.Unmarshal([]byte(test.data), &spec)
			} else {
				var spec SurfaceSpec
				err = json.Unmarshal([]byte(test.data), &spec)
			}
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("error = %v, want %q", err, test.contains)
			}
		})
	}
}

func TestEmissionSpecRejectsGeometrySpecificLegacyTypes(t *testing.T) {
	for _, kind := range []string{"cell_palette", "uv_klein"} {
		var spec EmissionSpec
		if err := json.Unmarshal([]byte(`{"type":"`+kind+`"}`), &spec); err == nil {
			t.Fatalf("expected legacy emission type %q to fail", kind)
		}
	}
}

func TestEngineVariantProtocolRejectsAliases(t *testing.T) {
	for _, kind := range []string{"hypercuboid", "hypersphere", "finite cylinder"} {
		var spec ObjectSpec
		data := `{"material_id":"m","shape":"` + kind + `"}`
		if err := json.Unmarshal([]byte(data), &spec); err == nil {
			t.Fatalf("expected shape alias %q to fail", kind)
		}
	}
	var surface SurfaceSpec
	if err := json.Unmarshal([]byte(`{"type":"wire_mesh"}`), &surface); err == nil {
		t.Fatal("expected surface alias wire_mesh to fail")
	}
}
