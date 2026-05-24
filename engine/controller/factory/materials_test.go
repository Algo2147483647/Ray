package factory

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
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
