package bxdf

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/material/core"
	"github.com/Algo2147483647/ray/engine/material/ior"
)

func TestSpecularReflectionSampleThroughputWeight(t *testing.T) {
	bxdf := NewSpecularReflection(core.NewSpectrum(0.8, 0.7, 0.6))
	wo := core.NewDirection(0.3, 0.4, math.Sqrt(1-0.3*0.3-0.4*0.4))

	sample := bxdf.Sample(core.ShadingContext{}, wo, core.Sample2D{})
	if sample.PDF != 1 {
		t.Fatalf("expected discrete delta pdf 1, got %f", sample.PDF)
	}
	if sample.Wi.X != -wo.X || sample.Wi.Y != -wo.Y || sample.Wi.Z != wo.Z {
		t.Fatalf("unexpected reflection direction: %+v", sample.Wi)
	}

	weight := sample.F.MulScalar(core.AbsCosTheta(sample.Wi) / sample.PDF)
	expected := core.NewSpectrum(0.8, 0.7, 0.6)
	if !weight.AlmostEqual(expected, 1e-12) {
		t.Fatalf("unexpected throughput weight: got %+v want %+v", weight, expected)
	}
}

func TestSpecularReflectionSampledReflectance(t *testing.T) {
	bxdf := NewSpecularReflectionParameter(core.NewSampledParameter(
		[]float64{400, 500, 600, 700},
		[]float64{0.2, 0.4, 0.6, 0.8},
	))
	ctx := core.ShadingContext{WavelengthsNM: []float64{450, 550, 650}}
	wo := core.NewDirection(0, 0, 1)

	sample := bxdf.Sample(ctx, wo, core.Sample2D{})
	expected := core.NewSampledSpectrum([]float64{0.3, 0.5, 0.7})
	if !sample.F.AlmostEqual(expected, 1e-12) {
		t.Fatalf("unexpected sampled specular reflectance: got %+v want %+v", sample.F, expected)
	}
}

func TestSpecularDielectricChoosesReflectionAndTransmission(t *testing.T) {
	bxdf := NewSpecularDielectricConstant(core.ConstantSpectrum(1), core.ConstantSpectrum(1), 1, 1.5)
	wo := core.NewDirection(0, 0, 1)
	f := FresnelDielectric(1, 1, 1.5)

	reflection := bxdf.Sample(core.ShadingContext{}, wo, core.Sample2D{U: 0})
	if reflection.Flags&core.DeltaReflection == 0 {
		t.Fatalf("expected reflection sample, got flags %v", reflection.Flags)
	}
	if math.Abs(reflection.PDF-f) > 1e-12 {
		t.Fatalf("unexpected reflection probability: got %f want %f", reflection.PDF, f)
	}

	transmission := bxdf.Sample(core.ShadingContext{}, wo, core.Sample2D{U: 1})
	if transmission.Flags&core.DeltaTransmission == 0 {
		t.Fatalf("expected transmission sample, got flags %v", transmission.Flags)
	}
	if math.Abs(transmission.Eta-1.5) > 1e-12 {
		t.Fatalf("expected transmitted medium eta 1.5, got %f", transmission.Eta)
	}
	if transmission.Wi.Z >= 0 {
		t.Fatalf("expected transmitted ray below surface, got %+v", transmission.Wi)
	}

	exit := bxdf.Sample(core.ShadingContext{CurrentIOR: 1.5}, wo, core.Sample2D{U: 1})
	if exit.Flags&core.DeltaTransmission == 0 {
		t.Fatalf("expected exit transmission sample, got flags %v", exit.Flags)
	}
	if math.Abs(exit.Eta-1) > 1e-12 {
		t.Fatalf("expected exited medium eta 1, got %f", exit.Eta)
	}
}

func TestSpecularDielectricUsesRendererWavelengthForDispersion(t *testing.T) {
	bxdf := NewSpecularDielectric(
		core.ConstantSpectrum(1),
		core.ConstantSpectrum(1),
		1,
		ior.NewCauchy(1.5, 0.004, 0),
	)
	wo := core.NewDirection(0, 0, 1)

	fallback := bxdf.Sample(core.ShadingContext{}, wo, core.Sample2D{U: 1, V: 0.25})
	expectedFallbackEta := ior.NewCauchy(1.5, 0.004, 0).Evaluate(ior.DefaultWavelengthNM)
	if fallback.WavelengthNM != 0 {
		t.Fatalf("expected bxdf not to sample wavelength on its own, got %f", fallback.WavelengthNM)
	}
	if math.Abs(fallback.Eta-expectedFallbackEta) > 1e-12 {
		t.Fatalf("unexpected fallback eta: got %f want %f", fallback.Eta, expectedFallbackEta)
	}

	continued := bxdf.Sample(core.ShadingContext{WavelengthNM: 610}, wo, core.Sample2D{U: 1, V: 0})
	expectedEta := ior.NewCauchy(1.5, 0.004, 0).Evaluate(610)
	if math.Abs(continued.WavelengthNM-610) > 1e-12 {
		t.Fatalf("expected existing wavelength to be preserved, got %f", continued.WavelengthNM)
	}
	if math.Abs(continued.Eta-expectedEta) > 1e-12 {
		t.Fatalf("unexpected dispersive eta: got %f want %f", continued.Eta, expectedEta)
	}
}

func TestSpecularDielectricUsesContextBoundaryEta(t *testing.T) {
	bxdf := NewSpecularDielectricConstant(core.ConstantSpectrum(1), core.ConstantSpectrum(1), 1, 1.5)
	wo := core.NewDirection(0, 0, 1)

	sample := bxdf.Sample(core.ShadingContext{
		EtaIncident:    1.5,
		EtaTransmit:    1.33,
		IncidentMedium: core.MediumID(2),
		TransmitMedium: core.MediumID(3),
		Entering:       true,
	}, wo, core.Sample2D{U: 1})

	if sample.Flags&core.DeltaTransmission == 0 {
		t.Fatalf("expected transmission sample, got flags %v", sample.Flags)
	}
	if math.Abs(sample.Eta-1.33) > 1e-12 {
		t.Fatalf("expected context eta 1.33, got %f", sample.Eta)
	}
	if sample.TransmitMedium != core.MediumID(3) {
		t.Fatalf("expected transmitted medium 3, got %d", sample.TransmitMedium)
	}
}
