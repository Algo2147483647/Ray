package core

import "testing"

func TestSampledSpectrumArithmeticPreservesChannels(t *testing.T) {
	a := NewSampledSpectrum([]float64{1, 2, 3, 4})
	b := NewSampledSpectrum([]float64{0.5, 0.25, 0.125, 0.0625})

	got := a.Mul(b).Add(ConstantSpectrum(1)).DivScalar(2)
	want := NewSampledSpectrum([]float64{0.75, 0.75, 0.6875, 0.625})

	if !got.AlmostEqual(want, 1e-12) {
		t.Fatalf("unexpected sampled arithmetic: got %+v want %+v", got, want)
	}
}

func TestSampledSpectrumDoesNotMirrorRGBChannels(t *testing.T) {
	got := NewSampledSpectrum([]float64{0.2, 0.5, 0.8})
	if got.Kind != SpectrumKindSampled {
		t.Fatalf("expected sampled spectrum kind, got %v", got.Kind)
	}
	if got.RGBChannel(0) != 0 || got.RGBChannel(1) != 0 || got.RGBChannel(2) != 0 {
		t.Fatalf("sampled spectrum should not populate RGB channels: %+v", got)
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
