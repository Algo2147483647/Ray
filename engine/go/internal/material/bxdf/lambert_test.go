package bxdf

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/go/internal/material/core"
	"github.com/Algo2147483647/ray/engine/go/internal/material/validation"
)

func TestLambertEvalPDFAndSample(t *testing.T) {
	ctx := core.ShadingContext{SpectrumMode: core.SpectrumRGB}
	bxdf := NewLambert(core.NewSpectrum(0.8, 0.4, 0.2))
	wi := core.NewDirection(0, 0, 1)
	wo := core.NewDirection(0, 0, 1)

	f := bxdf.Eval(ctx, wi, wo)
	expected := core.NewSpectrum(0.8/math.Pi, 0.4/math.Pi, 0.2/math.Pi)
	if !f.AlmostEqual(expected, 1e-12) {
		t.Fatalf("unexpected eval: got %+v want %+v", f, expected)
	}

	pdf := bxdf.PDF(ctx, wi, wo)
	if math.Abs(pdf-1/math.Pi) > 1e-12 {
		t.Fatalf("unexpected pdf: got %f want %f", pdf, 1/math.Pi)
	}

	sample := bxdf.Sample(ctx, wo, core.Sample2D{U: 0.25, V: 0.5})
	if sample.PDF <= 0 {
		t.Fatalf("expected positive sample pdf, got %f", sample.PDF)
	}
	if !sample.F.AlmostEqual(bxdf.Eval(ctx, sample.Wi, wo), 1e-12) {
		t.Fatalf("sample returned eval mismatch: got %+v", sample.F)
	}
}

func TestLambertRejectsInvalidHemisphere(t *testing.T) {
	ctx := core.ShadingContext{SpectrumMode: core.SpectrumRGB}
	bxdf := NewLambert(core.ConstantSpectrum(1))
	wi := core.NewDirection(0, 0, -1)
	wo := core.NewDirection(0, 0, 1)

	if got := bxdf.Eval(ctx, wi, wo); got.MaxComponent() != 0 {
		t.Fatalf("expected zero eval for invalid hemisphere, got %+v", got)
	}
	if got := bxdf.PDF(ctx, wi, wo); got != 0 {
		t.Fatalf("expected zero pdf for invalid hemisphere, got %f", got)
	}
}

func TestLambertValidation(t *testing.T) {
	ctx := core.ShadingContext{SpectrumMode: core.SpectrumRGB}
	bxdf := NewLambert(core.NewSpectrum(0.9, 0.7, 0.5))

	err := validation.CheckBasicPhysicalValidity(bxdf, ctx, validation.Options{
		DirectionSamples: 512,
		Tolerance:        2e-2,
	})
	if err != nil {
		t.Fatal(err)
	}
}
