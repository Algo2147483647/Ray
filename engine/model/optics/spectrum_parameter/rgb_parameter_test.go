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

func TestRGBParameterUpliftsAuthoredRGBInSpectralContext(t *testing.T) {
	red := NewRGBParameter(optics.NewSpectrum(0.82, 0.08, 0.045))

	ctx := testWavelengthContext{wavelengths: []float64{450, 610}}

	got := red.Eval(ctx)

	if !got.HasSamples() || len(got.Samples) != 2 {
		t.Fatalf("expected authored RGB to uplift into sampled spectrum, got %+v", got)
	}
	if got.Samples[1] <= got.Samples[0] {
		t.Fatalf("expected red authored color to be stronger at red wavelength, got %v", got.Samples)
	}
}
