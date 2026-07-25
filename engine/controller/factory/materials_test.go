package factory

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
)

func TestParseRoughConductorWeight(t *testing.T) {
	script := &parser.Script{
		Materials: []map[string]interface{}{
			{
				"id": "warm-metal",
				"surface": map[string]interface{}{
					"type":      "rough_conductor",
					"eta":       []interface{}{0.17, 0.35, 1.5},
					"k":         []interface{}{3.1, 2.7, 1.9},
					"roughness": 0.3,
					"weight": map[string]interface{}{
						"type":  "rgb",
						"value": []interface{}{1.0, 0.7, 0.25},
					},
				},
			},
		},
	}

	materials, err := ParseMaterials(script)
	if err != nil {
		t.Fatalf("ParseMaterials failed: %v", err)
	}

	single, ok := materials["warm-metal"].Surface.(bsdf.Single)
	if !ok {
		t.Fatalf("expected single BSDF, got %T", materials["warm-metal"].Surface)
	}
	got, ok := single.BxDF.(bxdf.RoughConductor)
	if !ok {
		t.Fatalf("expected rough conductor, got %T", single.BxDF)
	}

	bounds := got.Weight.Bounds().Max
	if bounds.RGBChannel(0) <= bounds.RGBChannel(1) || bounds.RGBChannel(1) <= bounds.RGBChannel(2) {
		t.Fatalf("expected warm gold weight to be red-dominant, got %+v", bounds)
	}
}

func TestParseRoughDielectricTransmission(t *testing.T) {
	script := &parser.Script{
		Materials: []map[string]interface{}{
			{
				"id": "frosted-glass",
				"surface": map[string]interface{}{
					"type":          "rough_dielectric_transmission",
					"transmittance": []interface{}{0.9, 0.95, 1.0},
					"eta_outside":   1.0,
					"ior": map[string]interface{}{
						"type": "constant",
						"eta":  1.5,
					},
					"roughness": 0.45,
				},
			},
		},
	}

	materials, err := ParseMaterials(script)
	if err != nil {
		t.Fatalf("ParseMaterials failed: %v", err)
	}

	single, ok := materials["frosted-glass"].Surface.(bsdf.Single)
	if !ok {
		t.Fatalf("expected single BSDF, got %T", materials["frosted-glass"].Surface)
	}
	got, ok := single.BxDF.(bxdf.RoughDielectricTransmission)
	if !ok {
		t.Fatalf("expected rough dielectric transmission, got %T", single.BxDF)
	}
	if got.Alpha <= 0 || got.Alpha > 1 {
		t.Fatalf("expected clamped alpha in (0,1], got %f", got.Alpha)
	}
}

func TestParseCylindricalGridCutout(t *testing.T) {
	script := &parser.Script{
		Materials: []map[string]interface{}{
			{
				"id": "silver-mesh",
				"surface": map[string]interface{}{
					"type":           "cylindrical_grid_cutout",
					"origin":         []interface{}{0, 0, 1.76},
					"axis":           []interface{}{0, 0, 1},
					"meridian_count": 16,
					"line_width":     0.01,
					"ring_gap":       0.05,
				},
			},
		},
	}

	materials, err := ParseMaterials(script)
	if err != nil {
		t.Fatalf("ParseMaterials failed: %v", err)
	}

	got, ok := materials["silver-mesh"].Surface.(bsdf.CylindricalGridCutout)
	if !ok {
		t.Fatalf("expected cylindrical grid cutout, got %T", materials["silver-mesh"].Surface)
	}
	if got.MeridianCount != 16 {
		t.Fatalf("expected 16 meridians, got %d", got.MeridianCount)
	}
	if got.RingSpacing <= got.RingLineWidth {
		t.Fatalf("expected configurable ring gap to exceed line width, spacing=%f line=%f", got.RingSpacing, got.RingLineWidth)
	}
	if !got.OnGridLine(bxdf.ShadingContext{HitPoint: maths.NewDirection(1, 0, 1.76)}) {
		t.Fatal("expected point on reference meridian to be a grid line")
	}
	if got.OnGridLine(bxdf.ShadingContext{HitPoint: maths.NewDirection(1, 0.1, 1.78)}) {
		t.Fatal("expected point away from meridians and rings to be a cutout gap")
	}
}
