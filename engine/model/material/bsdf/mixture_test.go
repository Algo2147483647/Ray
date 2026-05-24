package bsdf

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

func TestWeightedMixturePreservesSelectedDeltaSample(t *testing.T) {
	mixture := NewWeightedMixture(
		WeightedBxDF{Weight: 1, BxDF: bxdf.NewLambert(optics.NewSpectrum(0.6, 0.6, 0.6))},
		WeightedBxDF{Weight: 1, BxDF: bxdf.NewSpecularReflection(optics.NewSpectrum(0.8, 0.7, 0.6))},
	)
	wo := maths.NewDirection(0, 0, 1)

	sample := mixture.Sample(bxdf.ShadingContext{}, wo, maths.Sample2D{U: 0.75, V: 0.5})

	if sample.Flags&bxdf.DeltaReflection == 0 {
		t.Fatalf("expected selected specular component flag to be preserved, got %v", sample.Flags)
	}
	if sample.PDF != 0.5 {
		t.Fatalf("expected delta PDF to be scaled by selection probability, got %f", sample.PDF)
	}
	if math.Abs(sample.F.RGBChannel(0)-0.4) > 1e-12 ||
		math.Abs(sample.F.RGBChannel(1)-0.35) > 1e-12 ||
		math.Abs(sample.F.RGBChannel(2)-0.3) > 1e-12 {
		t.Fatalf("expected delta F to be preserved and selection-scaled, got %+v", sample.F)
	}
}

func TestWeightedMixtureKeepsNonDeltaSampleFlags(t *testing.T) {
	mixture := NewWeightedMixture(
		WeightedBxDF{Weight: 1, BxDF: bxdf.NewLambert(optics.NewSpectrum(0.6, 0.6, 0.6))},
		WeightedBxDF{Weight: 1, BxDF: bxdf.NewSpecularReflection(optics.NewSpectrum(0.8, 0.7, 0.6))},
	)
	wo := maths.NewDirection(0, 0, 1)

	sample := mixture.Sample(bxdf.ShadingContext{}, wo, maths.Sample2D{U: 0.25, V: 0.5})

	if sample.Flags != bxdf.DeltaNone {
		t.Fatalf("expected selected lambert flags to remain non-delta, got %v", sample.Flags)
	}
	if sample.PDF <= 0 {
		t.Fatalf("expected finite mixture PDF for lambert sample, got %f", sample.PDF)
	}
	if sample.F.IsZero() {
		t.Fatal("expected finite mixture F for lambert sample")
	}
}
