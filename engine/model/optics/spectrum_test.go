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

func TestRGBReflectanceUpliftIsEnergyBounded(t *testing.T) {
	floor := NewRGBSpectrum(0.78, 0.76, 0.71)

	got := floor.UpliftRGBReflectanceToSampled([]float64{450, 550, 610})

	if !got.HasSamples() || len(got.Samples) != 3 {
		t.Fatalf("expected three uplifted spectral samples, got %+v", got)
	}
	for _, sample := range got.Samples {
		if sample < 0 || sample > floor.MaxComponent() {
			t.Fatalf("expected reflectance uplift to stay within [0,maxRGB], got samples %v", got.Samples)
		}
	}
}
