package bxdf

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/material/core"
	"github.com/Algo2147483647/ray/engine/model/material/validation"
)

func TestRoughConductorEvalIsFiniteAndReciprocal(t *testing.T) {
	bxdf := NewRoughConductor(
		core.NewSpectrum(0.2, 0.9, 1.5),
		core.NewSpectrum(3.9, 2.5, 1.9),
		0.35,
	)
	ctx := core.ShadingContext{SpectrumMode: core.SpectrumRGB}
	wi := core.NewDirection(0.3, -0.1, math.Sqrt(0.9)).Normalize()
	wo := core.NewDirection(-0.2, 0.25, math.Sqrt(0.8975)).Normalize()

	f := bxdf.Eval(ctx, wi, wo)
	if !f.IsFinite() || !f.IsNonNegative() {
		t.Fatalf("rough conductor eval must be finite and non-negative, got %+v", f)
	}
	if !f.AlmostEqual(bxdf.Eval(ctx, wo, wi), 1e-12) {
		t.Fatalf("rough conductor should be reciprocal")
	}
}

func TestRoughConductorSamplePDFConsistency(t *testing.T) {
	bxdf := NewRoughConductor(
		core.NewSpectrum(0.2, 0.9, 1.5),
		core.NewSpectrum(3.9, 2.5, 1.9),
		0.4,
	)
	ctx := core.ShadingContext{SpectrumMode: core.SpectrumRGB}
	wo := core.NewDirection(0.2, -0.1, math.Sqrt(0.95)).Normalize()

	sample := bxdf.Sample(ctx, wo, core.Sample2D{U: 0.37, V: 0.82})
	if sample.PDF <= 0 {
		t.Fatalf("expected positive pdf, got %f", sample.PDF)
	}
	if math.Abs(sample.PDF-bxdf.PDF(ctx, sample.Wi, wo)) > 1e-12 {
		t.Fatalf("sample/pdf mismatch")
	}
	if !sample.F.AlmostEqual(bxdf.Eval(ctx, sample.Wi, wo), 1e-12) {
		t.Fatalf("sample/eval mismatch")
	}
}

func TestRoughConductorSampledEtaK(t *testing.T) {
	bxdf := NewRoughConductorParameter(
		core.NewSampledParameter([]float64{400, 500, 600, 700}, []float64{0.2, 0.3, 0.4, 0.5}),
		core.NewSampledParameter([]float64{400, 500, 600, 700}, []float64{3.0, 2.7, 2.4, 2.1}),
		0.25,
	)
	ctx := core.ShadingContext{WavelengthsNM: []float64{450, 550, 650}}
	wi := core.NewDirection(0.2, 0.1, 0.97).Normalize()
	wo := core.NewDirection(-0.1, 0.3, 0.95).Normalize()

	f := bxdf.Eval(ctx, wi, wo)
	if len(f.Samples) != 3 {
		t.Fatalf("expected sampled conductor response with 3 channels, got %+v", f)
	}
	if !f.IsFinite() || !f.IsNonNegative() {
		t.Fatalf("expected finite non-negative sampled conductor response, got %+v", f)
	}
}

func TestRoughConductorValidation(t *testing.T) {
	bxdf := NewRoughConductor(
		core.NewSpectrum(0.2, 0.9, 1.5),
		core.NewSpectrum(3.9, 2.5, 1.9),
		0.5,
	)
	err := validation.CheckBasicPhysicalValidity(bxdf, core.ShadingContext{SpectrumMode: core.SpectrumRGB}, validation.Options{
		DirectionSamples: 256,
		Tolerance:        5e-2,
	})
	if err != nil {
		t.Fatal(err)
	}
}
