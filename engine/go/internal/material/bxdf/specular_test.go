package bxdf

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/go/internal/material/core"
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

func TestSpecularDielectricChoosesReflectionAndTransmission(t *testing.T) {
	bxdf := NewSpecularDielectric(core.ConstantSpectrum(1), core.ConstantSpectrum(1), 1, 1.5)
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
