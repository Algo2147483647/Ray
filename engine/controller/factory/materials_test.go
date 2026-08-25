package factory

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/emission"
)

func mustMaterialSpecs(t *testing.T, definitions ...map[string]interface{}) []parser.MaterialSpec {
	t.Helper()
	data, err := json.Marshal(definitions)
	if err != nil {
		t.Fatal(err)
	}
	var specs []parser.MaterialSpec
	if err := json.Unmarshal(data, &specs); err != nil {
		t.Fatal(err)
	}
	return specs
}

func TestParseCosinePowerEmission(t *testing.T) {
	script := &parser.Script{Materials: mustMaterialSpecs(t,
		map[string]interface{}{
			"id": "spot-panel",
			"emission": map[string]interface{}{
				"type":     "constant",
				"exitance": []interface{}{12.0, 8.0, 4.0},
				"distribution": map[string]interface{}{
					"type": "cosine_power", "half_angle_degrees": 30.0, "sidedness": "front",
				},
			},
		},
	)}
	materials, err := ParseMaterials(script)
	if err != nil {
		t.Fatalf("ParseMaterials failed: %v", err)
	}
	emitter, ok := materials["spot-panel"].Emission.(emission.SurfaceEmitter)
	if !ok {
		t.Fatalf("expected SurfaceEmitter, got %T", materials["spot-panel"].Emission)
	}
	distribution, ok := emitter.Distribution.(emission.CosinePower)
	if !ok {
		t.Fatalf("expected CosinePower, got %T", emitter.Distribution)
	}
	wantExponent := math.Log(0.5) / math.Log(math.Cos(math.Pi/6))
	if math.Abs(distribution.Exponent-wantExponent) > 1e-12 || distribution.Sidedness != emission.FrontSide {
		t.Fatalf("unexpected distribution: %+v", distribution)
	}
	if emitter.Quantity != emission.TotalExitance {
		t.Fatalf("quantity = %v, want total exitance", emitter.Quantity)
	}
}

func TestParseEmissionRejectsAmbiguousOrInvalidDirection(t *testing.T) {
	tests := []struct {
		name     string
		emission map[string]interface{}
		contains string
	}{
		{
			name: "radiance and exitance",
			emission: map[string]interface{}{
				"type": "constant", "radiance": []interface{}{1, 1, 1}, "exitance": []interface{}{1, 1, 1},
			},
			contains: "mutually exclusive",
		},
		{
			name: "two direction parameters",
			emission: map[string]interface{}{
				"type": "constant", "radiance": []interface{}{1, 1, 1},
				"distribution": map[string]interface{}{
					"type": "cosine_power", "exponent": 2.0, "half_angle_degrees": 30.0,
				},
			},
			contains: "exactly one",
		},
		{
			name: "negative exponent",
			emission: map[string]interface{}{
				"type": "constant", "radiance": []interface{}{1, 1, 1},
				"distribution": map[string]interface{}{"type": "cosine_power", "exponent": -1.0},
			},
			contains: "finite and >= 0",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			script := &parser.Script{Materials: mustMaterialSpecs(t, map[string]interface{}{"id": "invalid", "emission": test.emission})}
			_, err := ParseMaterials(script)
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("error = %v, want substring %q", err, test.contains)
			}
		})
	}
}

func TestParseRoughConductorWeight(t *testing.T) {
	script := &parser.Script{
		Materials: mustMaterialSpecs(t,
			map[string]interface{}{
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
		),
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
		Materials: mustMaterialSpecs(t,
			map[string]interface{}{
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
		),
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

func TestParseWeightedMixture(t *testing.T) {
	script := &parser.Script{
		Materials: mustMaterialSpecs(t,
			map[string]interface{}{
				"id": "glazed-porcelain",
				"surface": map[string]interface{}{
					"type": "weighted_mixture",
					"components": []interface{}{
						map[string]interface{}{
							"weight": 0.8,
							"surface": map[string]interface{}{
								"type":   "lambert",
								"albedo": []interface{}{0.78, 0.78, 0.78},
							},
						},
						map[string]interface{}{
							"weight": 0.2,
							"surface": map[string]interface{}{
								"type":        "rough_dielectric_reflection",
								"reflectance": []interface{}{1.0, 1.0, 1.0},
								"eta_outside": 1.0,
								"eta_inside":  1.5,
								"roughness":   0.14,
							},
						},
					},
				},
			},
		),
	}

	materials, err := ParseMaterials(script)
	if err != nil {
		t.Fatalf("ParseMaterials failed: %v", err)
	}
	mixture, ok := materials["glazed-porcelain"].Surface.(bsdf.WeightedMixture)
	if !ok {
		t.Fatalf("expected weighted mixture, got %T", materials["glazed-porcelain"].Surface)
	}
	if len(mixture.Components) != 2 {
		t.Fatalf("component count = %d, want 2", len(mixture.Components))
	}
	if mixture.Components[0].Weight != 0.8 || mixture.Components[1].Weight != 0.2 {
		t.Fatalf("unexpected component weights: %+v", mixture.Components)
	}
	glaze, ok := mixture.Components[1].BxDF.(bsdf.Single)
	if !ok {
		t.Fatalf("expected glaze component to be a single BSDF, got %T", mixture.Components[1].BxDF)
	}
	if _, ok := glaze.BxDF.(bxdf.RoughDielectricReflection); !ok {
		t.Fatalf("expected rough dielectric glaze, got %T", glaze.BxDF)
	}
}

func TestParseWeightedMixtureRejectsInvalidComponents(t *testing.T) {
	for name, components := range map[string]interface{}{
		"not an array": "invalid",
		"empty":        []interface{}{},
		"zero weight": []interface{}{
			map[string]interface{}{
				"weight": 0.0,
				"surface": map[string]interface{}{
					"type":   "lambert",
					"albedo": []interface{}{1.0, 1.0, 1.0},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			definition := map[string]interface{}{
				"materials": []interface{}{map[string]interface{}{
					"id": "invalid-mixture",
					"surface": map[string]interface{}{
						"type":       "weighted_mixture",
						"components": components,
					},
				}},
			}
			data, err := json.Marshal(definition)
			if err != nil {
				t.Fatal(err)
			}
			var script parser.Script
			if err := json.Unmarshal(data, &script); err != nil {
				return // Invalid structure was rejected at the typed protocol boundary.
			}
			if _, err := ParseMaterials(&script); err == nil {
				t.Fatal("expected invalid weighted mixture to fail")
			}
		})
	}
}

func TestParseCylindricalGridCutout(t *testing.T) {
	script := &parser.Script{
		Materials: mustMaterialSpecs(t,
			map[string]interface{}{
				"id": "silver-mesh",
				"surface": map[string]interface{}{
					"type":             "cylindrical_grid_cutout",
					"origin":           []interface{}{0, 0, 1.76},
					"axis":             []interface{}{0, 0, 1},
					"line_width":       0.01,
					"gap_width":        0.05,
					"gap_height":       0.04,
					"reference_radius": 1.0,
				},
			},
		),
	}

	materials, err := ParseMaterials(script)
	if err != nil {
		t.Fatalf("ParseMaterials failed: %v", err)
	}

	got, ok := materials["silver-mesh"].Surface.(bsdf.CylindricalGridCutout)
	if !ok {
		t.Fatalf("expected cylindrical grid cutout, got %T", materials["silver-mesh"].Surface)
	}
	if got.GapWidth != 0.05 {
		t.Fatalf("expected gap width 0.05, got %f", got.GapWidth)
	}
	if got.GapHeight != 0.04 {
		t.Fatalf("expected gap height 0.04, got %f", got.GapHeight)
	}
	if got.ReferenceRadius != 1.0 {
		t.Fatalf("expected reference radius 1.0, got %f", got.ReferenceRadius)
	}
	if !got.OnGridLine(bxdf.ShadingContext{HitPoint: maths.NewDirection(1, 0, 1.76)}) {
		t.Fatal("expected point on reference meridian to be a grid line")
	}
	if !got.OnGridLine(bxdf.ShadingContext{HitPoint: maths.NewDirection(2, 0, 1.76)}) {
		t.Fatal("expected reference meridian to remain straight across radii")
	}
	if got.OnGridLine(bxdf.ShadingContext{HitPoint: maths.NewDirection(1, 0.02, 1.78)}) {
		t.Fatal("expected point away from meridians and rings to be a cutout gap")
	}
}
