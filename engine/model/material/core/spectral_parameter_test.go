package core

import "testing"

func TestSampledParameterEvaluatesWavelengthPacket(t *testing.T) {
	parameter := NewSampledParameter(
		[]float64{400, 500, 600, 700},
		[]float64{0.1, 0.3, 0.7, 0.9},
	)

	got := parameter.Eval(ShadingContext{
		WavelengthsNM: []float64{450, 550, 650},
	})
	want := NewSampledSpectrum([]float64{0.2, 0.5, 0.8})
	want.R = 0.2
	want.G = 0.5
	want.B = 0.8

	if !got.AlmostEqual(want, 1e-12) {
		t.Fatalf("unexpected sampled parameter packet: got %+v want %+v", got, want)
	}
}
