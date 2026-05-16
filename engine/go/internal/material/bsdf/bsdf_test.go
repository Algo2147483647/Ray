package bsdf

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/go/internal/material/bxdf"
	"github.com/Algo2147483647/ray/engine/go/internal/material/core"
	"github.com/Algo2147483647/ray/engine/go/internal/material/validation"
)

func TestSingleDelegatesToBxDF(t *testing.T) {
	ctx := core.ShadingContext{SpectrumMode: core.SpectrumRGB}
	lambert := bxdf.NewLambert(core.NewSpectrum(0.6, 0.3, 0.1))
	surface := NewSingle(lambert)
	wi := core.NewDirection(0, 0, 1)
	wo := core.NewDirection(0, 0, 1)

	if !surface.Eval(ctx, wi, wo).AlmostEqual(lambert.Eval(ctx, wi, wo), 1e-12) {
		t.Fatal("single BSDF should delegate Eval to its BxDF")
	}
	if math.Abs(surface.PDF(ctx, wi, wo)-lambert.PDF(ctx, wi, wo)) > 1e-12 {
		t.Fatal("single BSDF should delegate PDF to its BxDF")
	}
}

func TestWeightedMixtureEvalAndPDF(t *testing.T) {
	ctx := core.ShadingContext{SpectrumMode: core.SpectrumRGB}
	a := bxdf.NewLambert(core.NewSpectrum(1.0, 0.0, 0.0))
	b := bxdf.NewLambert(core.NewSpectrum(0.0, 0.0, 1.0))
	mix := NewWeightedMixture(
		WeightedBxDF{Weight: 1, BxDF: a},
		WeightedBxDF{Weight: 3, BxDF: b},
	)
	wi := core.NewDirection(0, 0, 1)
	wo := core.NewDirection(0, 0, 1)

	expected := core.NewSpectrum(0.25/math.Pi, 0, 0.75/math.Pi)
	if got := mix.Eval(ctx, wi, wo); !got.AlmostEqual(expected, 1e-12) {
		t.Fatalf("unexpected mixture eval: got %+v want %+v", got, expected)
	}

	if got := mix.PDF(ctx, wi, wo); math.Abs(got-1/math.Pi) > 1e-12 {
		t.Fatalf("unexpected mixture pdf: got %f", got)
	}
}

func TestWeightedMixtureSampleReturnsMixtureEvalAndPDF(t *testing.T) {
	ctx := core.ShadingContext{SpectrumMode: core.SpectrumRGB}
	mix := NewWeightedMixture(
		WeightedBxDF{Weight: 1, BxDF: bxdf.NewLambert(core.NewSpectrum(1.0, 0.0, 0.0))},
		WeightedBxDF{Weight: 1, BxDF: bxdf.NewLambert(core.NewSpectrum(0.0, 1.0, 0.0))},
	)
	wo := core.NewDirection(0, 0, 1)

	sample := mix.Sample(ctx, wo, core.Sample2D{U: 0.75, V: 0.25})
	if !sample.F.AlmostEqual(mix.Eval(ctx, sample.Wi, wo), 1e-12) {
		t.Fatalf("sample should return mixture eval, got %+v", sample.F)
	}
	if math.Abs(sample.PDF-mix.PDF(ctx, sample.Wi, wo)) > 1e-12 {
		t.Fatalf("sample should return mixture pdf, got %f", sample.PDF)
	}
}

func TestWeightedMixtureValidation(t *testing.T) {
	ctx := core.ShadingContext{SpectrumMode: core.SpectrumRGB}
	mix := NewWeightedMixture(
		WeightedBxDF{Weight: 1, BxDF: bxdf.NewLambert(core.NewSpectrum(0.8, 0.2, 0.2))},
		WeightedBxDF{Weight: 2, BxDF: bxdf.NewLambert(core.NewSpectrum(0.2, 0.4, 0.7))},
	)

	err := validation.CheckBasicPhysicalValidity(mix, ctx, validation.Options{
		DirectionSamples: 512,
		Tolerance:        2e-2,
	})
	if err != nil {
		t.Fatal(err)
	}
}
