package optics

import "testing"

func TestSpectrumDoesNotImplicitlyAverageRGBAndSampledMul(t *testing.T) {
	rgb := NewRGBSpectrum(0.9, 0.1, 0.1)
	sampled := NewSampledSpectrum([]float64{2, 4})

	got := rgb.Mul(sampled)

	if got.HasSamples() || !got.IsZero() {
		t.Fatalf("expected mixed RGB/sample multiplication to be rejected as zero spectrum, got %+v", got)
	}
}

func TestSpectrumDoesNotImplicitlyAverageRGBAndSampledAdd(t *testing.T) {
	rgb := NewRGBSpectrum(0.9, 0.1, 0.1)
	sampled := NewSampledSpectrum([]float64{2, 4})

	got := rgb.Add(sampled)

	if got.HasSamples() || !got.IsZero() {
		t.Fatalf("expected mixed RGB/sample addition to be rejected as zero spectrum, got %+v", got)
	}
}

func TestRGBUpliftProducesWavelengthSamples(t *testing.T) {
	red := NewRGBSpectrum(0.8, 0.05, 0.02)

	got := red.UpliftRGBToSampled([]float64{450, 610})

	if !got.HasSamples() || len(got.Samples) != 2 {
		t.Fatalf("expected two uplifted spectral samples, got %+v", got)
	}
	if got.Samples[1] <= got.Samples[0] {
		t.Fatalf("expected red RGB uplift to be stronger at red wavelength, got %v", got.Samples)
	}
}
