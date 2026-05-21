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
