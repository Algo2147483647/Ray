package spectrum_parameter

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/model/optics"
)

type testWavelengthContext struct {
	wavelengths []float64
}

func (c testWavelengthContext) SpectralWavelengthNM() float64 {
	if len(c.wavelengths) == 0 {
		return 0
	}
	return c.wavelengths[0]
}

func (c testWavelengthContext) SpectralWavelengthsNM() []float64 {
	return c.wavelengths
}

func TestRGBParameterKeepsAuthoredRGBInSpectralContext(t *testing.T) {
	red := NewRGBParameter(optics.NewSpectrum(0.82, 0.08, 0.045))

	ctx := testWavelengthContext{wavelengths: []float64{450, 610}}

	got := red.Eval(ctx)

	if got.HasSamples() {
		t.Fatalf("expected authored RGB to remain RGB in hybrid spectral mode, got samples %v", got.Samples)
	}
	if got.RGBChannel(0) <= got.RGBChannel(1) || got.RGBChannel(0) <= got.RGBChannel(2) {
		t.Fatalf("expected red authored color to keep its dominant red channel, got %v", got)
	}
}
