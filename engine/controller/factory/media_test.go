package factory

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
)

type mediaTestWavelengthContext struct {
	wavelengths []float64
}

func (c mediaTestWavelengthContext) SpectralWavelengthNM() float64 {
	if len(c.wavelengths) == 0 {
		return 0
	}
	return c.wavelengths[0]
}

func (c mediaTestWavelengthContext) SpectralWavelengthsNM() []float64 {
	return c.wavelengths
}

func TestParseMediaRegistryKeepsAbsorptionAndScattering(t *testing.T) {
	script := &parser.Script{
		Media: map[string]map[string]interface{}{
			"tinted-water": {
				"type": "homogeneous",
				"ior": map[string]interface{}{
					"type": "constant",
					"eta":  1.33,
				},
				"sigma_a": []interface{}{0.2, 0.4, 0.8},
				"sigma_s": []interface{}{0.01, 0.02, 0.03},
			},
		},
	}

	registry, err := ParseMediaRegistry(script)
	if err != nil {
		t.Fatalf("ParseMediaRegistry failed: %v", err)
	}
	id, ok := registry.ID("tinted-water")
	if !ok {
		t.Fatal("expected medium id")
	}

	sigmaA := registry.SigmaA(id, mediaTestWavelengthContext{})
	sigmaS := registry.SigmaS(id, mediaTestWavelengthContext{})

	if math.Abs(sigmaA.RGBChannel(0)-0.2) > 1e-12 ||
		math.Abs(sigmaA.RGBChannel(1)-0.4) > 1e-12 ||
		math.Abs(sigmaA.RGBChannel(2)-0.8) > 1e-12 {
		t.Fatalf("unexpected sigma_a: %+v", sigmaA)
	}
	if math.Abs(sigmaS.RGBChannel(0)-0.01) > 1e-12 ||
		math.Abs(sigmaS.RGBChannel(1)-0.02) > 1e-12 ||
		math.Abs(sigmaS.RGBChannel(2)-0.03) > 1e-12 {
		t.Fatalf("unexpected sigma_s: %+v", sigmaS)
	}
}

func TestParseMediaRegistryEvaluatesSampledAbsorptionAtWavelength(t *testing.T) {
	script := &parser.Script{
		Media: map[string]map[string]interface{}{
			"filter": {
				"sigma_a": map[string]interface{}{
					"type":           "sampled",
					"wavelengths_nm": []interface{}{500.0, 600.0},
					"values":         []interface{}{0.1, 0.5},
				},
			},
		},
	}

	registry, err := ParseMediaRegistry(script)
	if err != nil {
		t.Fatalf("ParseMediaRegistry failed: %v", err)
	}
	id, ok := registry.ID("filter")
	if !ok {
		t.Fatal("expected medium id")
	}

	sigmaA := registry.SigmaA(id, mediaTestWavelengthContext{wavelengths: []float64{550}})

	if !sigmaA.HasSamples() || math.Abs(sigmaA.Sample(0)-0.3) > 1e-12 {
		t.Fatalf("expected interpolated sampled sigma_a of 0.3, got %+v", sigmaA)
	}
}
