package optics

import (
	"math"
	"testing"
)

func TestUniformWavelengthSampler(t *testing.T) {
	sampler := NewUniformWavelengthSampler()

	first := sampler.Sample(0)
	middle := sampler.Sample(0.5)
	last := sampler.Sample(1)

	if first.LambdaNM <= WavelengthMin || first.LambdaNM >= WavelengthMax {
		t.Fatalf("first wavelength out of visible range: %f", first.LambdaNM)
	}
	if math.Abs(middle.LambdaNM-((WavelengthMin+WavelengthMax)/2)) > 1e-9 {
		t.Fatalf("unexpected middle wavelength: %f", middle.LambdaNM)
	}
	if last.LambdaNM <= WavelengthMin || last.LambdaNM >= WavelengthMax {
		t.Fatalf("last wavelength out of visible range: %f", last.LambdaNM)
	}
	if math.Abs(middle.PDF-UniformWavelengthPDF()) > 1e-12 {
		t.Fatalf("unexpected wavelength pdf: got %f want %f", middle.PDF, UniformWavelengthPDF())
	}
}

func TestWeightedWavelengthSamplerBiasesTowardWeightedRegion(t *testing.T) {
	sampler := NewWeightedWavelengthSampler(WavelengthMin, WavelengthMax, 16, func(wavelengthNM float64) float64 {
		if wavelengthNM > 600 {
			return 10
		}
		return 1
	})

	low := sampler.Sample(0.1)
	high := sampler.Sample(0.9)

	if high.LambdaNM <= 600 {
		t.Fatalf("expected high quantile to land in heavily weighted red region, got %f", high.LambdaNM)
	}
	if high.PDF <= low.PDF {
		t.Fatalf("expected weighted red region to have larger pdf, got low=%f high=%f", low.PDF, high.PDF)
	}
}

func TestRGBImportanceWavelengthSamplerFavorsAuthoredColor(t *testing.T) {
	sampler := NewRGBImportanceWavelengthSampler(NewRGBSpectrum(0.9, 0.05, 0.02), 32)

	got := sampler.Sample(0.75)

	if got.LambdaNM < 560 {
		t.Fatalf("expected red RGB importance sampler to favor longer wavelengths, got %f", got.LambdaNM)
	}
	if got.PDF <= 0 {
		t.Fatalf("expected positive pdf")
	}
}
