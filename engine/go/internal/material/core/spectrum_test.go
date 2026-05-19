package core

import "testing"

func TestSampledSpectrumArithmeticPreservesChannels(t *testing.T) {
	a := NewSampledSpectrum([]float64{1, 2, 3, 4})
	b := NewSampledSpectrum([]float64{0.5, 0.25, 0.125, 0.0625})

	got := a.Mul(b).Add(ConstantSpectrum(1)).DivScalar(2)
	want := NewSampledSpectrum([]float64{0.75, 0.75, 0.6875, 0.625})
	want.R = 0.75
	want.G = 0.75
	want.B = 0.6875

	if !got.AlmostEqual(want, 1e-12) {
		t.Fatalf("unexpected sampled arithmetic: got %+v want %+v", got, want)
	}
}

func TestSampledSpectrumValidationChecksChannels(t *testing.T) {
	if NewSampledSpectrum([]float64{1, -0.1}).IsNonNegative() {
		t.Fatal("expected negative sampled channel to fail non-negative check")
	}
	if !NewSampledSpectrum([]float64{1, 2, 3}).IsFinite() {
		t.Fatal("expected finite sampled spectrum")
	}
}
