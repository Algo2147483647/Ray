package bxdf_test

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/material/microfacet"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
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

func TestLambertSamples4DUpperHemisphere(t *testing.T) {
	lambert := bxdf.NewLambert(optics.NewSpectrum(0.6, 0.4, 0.2))
	wo := maths.NewDirectionFromComponents([]float64{0.1, -0.2, 0.3, 0.9}).Normalize()

	sample := lambert.Sample(bxdf.ShadingContext{}, wo, maths.Sample2D{U: 0.3, V: 0.7})

	if sample.Wi.Len() != 4 {
		t.Fatalf("expected 4D sampled direction, got %dD", sample.Wi.Len())
	}
	if !maths.IsUpperHemisphere(sample.Wi) {
		t.Fatalf("expected upper hemisphere sample, got %+v", sample.Wi)
	}
	if math.Abs(sample.Wi.Length()-1) > 1e-12 {
		t.Fatalf("expected normalized direction, got length %f", sample.Wi.Length())
	}
	if sample.PDF <= 0 || !sample.F.IsFinite() {
		t.Fatalf("expected finite positive Lambert sample, got %+v", sample)
	}
}

func TestSpecularReflectionReflects4DLocalDirection(t *testing.T) {
	specular := bxdf.NewSpecularReflection(optics.NewSpectrum(1, 1, 1))
	wo := maths.NewDirectionFromComponents([]float64{0.1, -0.2, 0.3, 0.9}).Normalize()

	sample := specular.Sample(bxdf.ShadingContext{}, wo, maths.Sample2D{})

	if sample.Wi.Len() != 4 {
		t.Fatalf("expected 4D reflected direction, got %dD", sample.Wi.Len())
	}
	for i := 0; i < sample.Wi.Len()-1; i++ {
		if math.Abs(sample.Wi.Component(i)+wo.Component(i)) > 1e-12 {
			t.Fatalf("component %d was not reflected: got %f want %f", i, sample.Wi.Component(i), -wo.Component(i))
		}
	}
	normalAxis := sample.Wi.Len() - 1
	if math.Abs(sample.Wi.Component(normalAxis)-wo.Component(normalAxis)) > 1e-12 {
		t.Fatalf("normal component changed: got %f want %f", sample.Wi.Component(normalAxis), wo.Component(normalAxis))
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

func TestRoughDielectricTransmissionSamplesOppositeHemisphere(t *testing.T) {
	transmission := bxdf.NewRoughDielectricTransmission(
		optics.NewSpectrum(0.9, 0.85, 0.8),
		1,
		1.5,
		0.2,
	)
	ctx := bxdf.ShadingContext{
		TransportMode: bxdf.TransportRadiance,
		EtaIncident:   1,
		EtaTransmit:   1.5,
	}
	wo := maths.NewDirection(0.15, -0.1, 0.98).Normalize()

	sample := transmission.Sample(ctx, wo, maths.Sample2D{U: 0.37, V: 0.61})

	if sample.PDF <= 0 {
		t.Fatalf("expected positive PDF, got %+v", sample)
	}
	if !sample.F.IsFinite() || !sample.F.IsNonNegative() {
		t.Fatalf("expected finite non-negative sample value, got %+v", sample.F)
	}
	if maths.SameHemisphere(sample.Wi, wo) {
		t.Fatalf("expected transmission to cross hemispheres, got wi=%+v wo=%+v", sample.Wi, wo)
	}
	if sample.Flags&bxdf.TransmissionEvent == 0 {
		t.Fatalf("expected transmission event flag, got %v", sample.Flags)
	}
	if sample.Flags&bxdf.DeltaTransmission != 0 {
		t.Fatalf("rough transmission should not be marked as delta, got %v", sample.Flags)
	}
	if math.Abs(sample.Eta-1.5) > 1e-12 {
		t.Fatalf("expected sample eta to carry transmitted-side IOR, got %f", sample.Eta)
	}
}

func TestRoughDielectricTransmissionSamplesFromInsideToOutside(t *testing.T) {
	transmission := bxdf.NewRoughDielectricTransmission(
		optics.NewSpectrum(1, 1, 1),
		1,
		1.5,
		0.3,
	)
	ctx := bxdf.ShadingContext{
		TransportMode: bxdf.TransportRadiance,
		EtaIncident:   1.5,
		EtaTransmit:   1,
	}
	wo := maths.NewDirection(-0.1, 0.12, -0.98).Normalize()

	sample := transmission.Sample(ctx, wo, maths.Sample2D{U: 0.44, V: 0.19})

	if sample.PDF <= 0 {
		t.Fatalf("expected inside-to-outside sample, got %+v", sample)
	}
	if !sample.F.AlmostEqual(transmission.Eval(ctx, sample.Wi, wo), 1e-12) {
		t.Fatalf("sample/eval mismatch: sample=%+v eval=%+v", sample.F, transmission.Eval(ctx, sample.Wi, wo))
	}
	if got := transmission.PDF(ctx, sample.Wi, wo); math.Abs(got-sample.PDF) > 1e-12 {
		t.Fatalf("sample/pdf mismatch: sample=%f pdf=%f", sample.PDF, got)
	}
	if maths.SameHemisphere(sample.Wi, wo) {
		t.Fatalf("expected inside-to-outside transmission to cross hemispheres, got wi=%+v wo=%+v", sample.Wi, wo)
	}
	if math.Abs(sample.Eta-1) > 1e-12 {
		t.Fatalf("expected sample eta to carry outside IOR, got %f", sample.Eta)
	}
}

func TestRoughDielectricTransmissionDefaultsNilTransmittance(t *testing.T) {
	transmission := bxdf.NewRoughDielectricTransmissionParameter(nil, 1, medium.NewConstant(1.5), 0.25)
	ctx := bxdf.ShadingContext{
		TransportMode: bxdf.TransportRadiance,
		EtaIncident:   1,
		EtaTransmit:   1.5,
	}
	wi := maths.NewDirection(0.02, -0.15, -0.98).Normalize()
	wo := maths.NewDirection(0.05, 0.1, 0.99).Normalize()

	got := transmission.Eval(ctx, wi, wo)

	if !got.IsFinite() || !got.IsNonNegative() {
		t.Fatalf("expected nil transmittance to default to finite non-negative value, got %+v", got)
	}
	if got.MaxComponent() == 0 {
		t.Fatalf("expected nil transmittance fallback to contribute non-zero transmission, got %+v", got)
	}
}

func TestRoughDielectricTransmissionUsesDispersiveIOR(t *testing.T) {
	transmission := bxdf.NewRoughDielectricTransmissionParameter(
		spectrum_parameter.NewConstantParameter(1),
		1,
		medium.NewCauchy(1.5046, 0.0042, 0),
		0.35,
	)
	ctx := bxdf.ShadingContext{
		TransportMode: bxdf.TransportRadiance,
		SpectrumMode:  optics.SpectrumModeHeroWavelength,
		WavelengthNM:  450,
		WavelengthsNM: []float64{450},
	}
	wo := maths.NewDirection(0.05, 0.2, 0.97).Normalize()

	sample := transmission.Sample(ctx, wo, maths.Sample2D{U: 0.2, V: 0.4})

	if sample.PDF <= 0 {
		t.Fatalf("expected sampled dispersive transmission, got %+v", sample)
	}
	if sample.WavelengthNM != 450 {
		t.Fatalf("expected sample wavelength to be propagated, got %f", sample.WavelengthNM)
	}
}
