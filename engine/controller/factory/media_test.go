package factory

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model/optics"
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

func TestParseMediaRegistryKeepsAbsorption(t *testing.T) {
	script := &parser.Script{
		Media: map[string]parser.MediumSpec{
			"tinted-water": {
				Type:   "homogeneous",
				IOR:    &parser.IORSpec{Type: "constant", Eta: float64Pointer(1.33)},
				SigmaA: json.RawMessage(`[0.2, 0.4, 0.8]`),
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

	sigmaA := registry.SigmaA(id, mediaTestWavelengthContext{wavelengths: []float64{550}})
	want := math.Min(0.8, optics.NewRGBSpectrum(0.2, 0.4, 0.8).RGBPowerAtWavelength(550))

	if math.Abs(sigmaA-want) > 1e-12 {
		t.Fatalf("sigma_a at 550nm = %g, want %g", sigmaA, want)
	}
}

func TestParseMediaRegistryRejectsSigmaS(t *testing.T) {
	script := &parser.Script{Media: map[string]parser.MediumSpec{
		"fog": {SigmaS: json.RawMessage(`[0.01, 0.02, 0.03]`)},
	}}
	if _, err := ParseMediaRegistry(script); err == nil {
		t.Fatal("sigma_s must be rejected until volume scattering is implemented")
	}
}

func TestParseMediaRegistryEvaluatesSampledAbsorptionAtWavelength(t *testing.T) {
	script := &parser.Script{
		Media: map[string]parser.MediumSpec{
			"filter": {
				SigmaA: json.RawMessage(`{
					"type": "sampled",
					"wavelengths_nm": [500.0, 600.0],
					"values": [0.1, 0.5]
				}`),
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

	if math.Abs(sigmaA-0.3) > 1e-12 {
		t.Fatalf("expected interpolated sampled sigma_a of 0.3, got %+v", sigmaA)
	}
}

func TestParseMediaRegistryRejectsUnknownSpectralVariantField(t *testing.T) {
	script := &parser.Script{Media: map[string]parser.MediumSpec{
		"filter": {
			SigmaA: json.RawMessage(`{"type":"constant","value":0.1,"mystery":true}`),
		},
	}}
	if _, err := ParseMediaRegistry(script); err == nil {
		t.Fatal("unknown spectral parameter field must be rejected")
	}
}
