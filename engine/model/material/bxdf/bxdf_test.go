package bxdf_test

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/microfacet"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
	"github.com/Algo2147483647/ray/engine/utils/maths"
)

func TestLambertBasicPhysicalValidity(t *testing.T) {
	lambert := bxdf.NewLambert(optics.NewSpectrum(0.6, 0.4, 0.2))
	opts := material.Options{DirectionSamples: 64, Tolerance: 1e-4}

	if err := material.CheckBasicPhysicalValidity(lambert, bxdf.ShadingContext{}, opts); err != nil {
		t.Fatalf("lambert validity failed: %v", err)
	}
}

func TestSpecularReflectionSampleIsDeltaDiscrete(t *testing.T) {
	specular := bxdf.NewSpecularReflection(optics.NewSpectrum(0.8, 0.7, 0.6))
	wo := maths.NewDirection(0.2, -0.1, 0.97).Normalize()

	sample := specular.Sample(bxdf.ShadingContext{}, wo, maths.Sample2D{U: 0.3, V: 0.7})

	if sample.Flags&bxdf.DeltaReflection == 0 {
		t.Fatalf("expected delta reflection flag, got %v", sample.Flags)
	}
	if sample.PDF != 1 {
		t.Fatalf("expected discrete sample PDF of 1, got %f", sample.PDF)
	}
	if got := specular.PDF(bxdf.ShadingContext{}, sample.Wi, wo); got != 0 {
		t.Fatalf("expected solid-angle PDF for delta reflection to stay 0, got %f", got)
	}
}

func TestFresnelConductorProducesChannelColor(t *testing.T) {
	eta := optics.NewSpectrum(0.2, 0.5, 1.5)
	k := optics.NewSpectrum(3.5, 2.4, 1.8)

	got := microfacet.FresnelConductor(0.6, eta, k)

	if got.RGBChannel(0) <= got.RGBChannel(2) {
		t.Fatalf("expected gold-like conductor Fresnel to be red-dominant, got %+v", got)
	}
	for ch := 0; ch < 3; ch++ {
		if got.RGBChannel(ch) < 0 || got.RGBChannel(ch) > 1 || math.IsNaN(got.RGBChannel(ch)) {
			t.Fatalf("unexpected Fresnel channel %d: %+v", ch, got)
		}
	}
}

func TestRoughConductorKeepsSampledColorFromSampledParameters(t *testing.T) {
	conductor := bxdf.NewRoughConductorParameter(
		spectrum_parameter.NewSampledParameter([]float64{450, 610}, []float64{1.5, 0.2}),
		spectrum_parameter.NewSampledParameter([]float64{450, 610}, []float64{1.8, 3.5}),
		0.35,
	)
	ctx := bxdf.ShadingContext{
		SpectrumMode:  optics.SpectrumModeHeroWavelength,
		WavelengthNM:  610,
		WavelengthsNM: []float64{450, 610},
	}
	wi := maths.NewDirection(0.25, 0.1, 0.96).Normalize()
	wo := maths.NewDirection(-0.15, 0.05, 0.98).Normalize()

	got := conductor.Eval(ctx, wi, wo)

	if !got.HasSamples() || len(got.Samples) != 2 {
		t.Fatalf("expected sampled rough conductor response, got %+v", got)
	}
	if got.Samples[0] == got.Samples[1] {
		t.Fatalf("expected sampled rough conductor color to vary by wavelength, got %v", got.Samples)
	}
	if got.Samples[1] <= got.Samples[0] {
		t.Fatalf("expected gold-like rough conductor to keep stronger red response, got %v", got.Samples)
	}
}
