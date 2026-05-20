package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
)

func TestPrepareMediumContextKeepsLegacyIORWithoutBoundary(t *testing.T) {
	ray := &renderray.Ray{RefractionIndex: 1.5}
	ray.MediumStack.Reset(medium.MediumAir)
	ctx := bxdf.ShadingContext{CurrentIOR: ray.RefractionIndex}

	prepareMediumContext(&ctx, medium.NewRegistry(), ray, medium.Boundary{}, true)

	if ctx.CurrentIOR != 1.5 {
		t.Fatalf("legacy current IOR changed: got %f want 1.5", ctx.CurrentIOR)
	}
	if ctx.EtaIncident != 0 || ctx.EtaTransmit != 0 {
		t.Fatalf("expected inactive boundary eta fields to stay empty, got %f -> %f", ctx.EtaIncident, ctx.EtaTransmit)
	}
}

func TestPrepareMediumContextUsesBoundaryEta(t *testing.T) {
	registry := medium.NewRegistry()
	glassID, err := registry.RegisterHomogeneous("glass", medium.NewConstant(1.5))
	if err != nil {
		t.Fatalf("register glass: %v", err)
	}
	ray := &renderray.Ray{}
	ray.MediumStack.Reset(medium.MediumAir)
	ctx := bxdf.ShadingContext{}

	prepareMediumContext(&ctx, registry, ray, medium.NewBoundary(medium.MediumAir, glassID), true)

	if ctx.IncidentMedium != medium.MediumAir || ctx.TransmitMedium != glassID {
		t.Fatalf("unexpected boundary media: %d -> %d", ctx.IncidentMedium, ctx.TransmitMedium)
	}
	if ctx.EtaIncident != 1 || ctx.EtaTransmit != 1.5 {
		t.Fatalf("unexpected eta pair: %f -> %f", ctx.EtaIncident, ctx.EtaTransmit)
	}
	if !ctx.Entering {
		t.Fatal("expected entering boundary")
	}
}

func TestPrepareMediumContextUsesPriorityResolver(t *testing.T) {
	registry := medium.NewRegistry()
	waterID, err := registry.RegisterHomogeneous("water", medium.NewConstant(1.33))
	if err != nil {
		t.Fatalf("register water: %v", err)
	}
	glassID, err := registry.RegisterHomogeneous("glass", medium.NewConstant(1.5))
	if err != nil {
		t.Fatalf("register glass: %v", err)
	}

	ray := &renderray.Ray{}
	ray.MediumStack.Reset(medium.MediumAir)
	ray.MediumStack.EnterBoundary(medium.Boundary{Inside: glassID, Priority: 10})
	ctx := bxdf.ShadingContext{}

	prepareMediumContext(&ctx, registry, ray, medium.Boundary{Inside: waterID, Priority: 1}, true)

	if ctx.IncidentMedium != glassID || ctx.TransmitMedium != glassID {
		t.Fatalf("expected lower-priority water boundary to stay optically in glass, got %d -> %d", ctx.IncidentMedium, ctx.TransmitMedium)
	}
	if ctx.EtaIncident != 1.5 || ctx.EtaTransmit != 1.5 {
		t.Fatalf("unexpected priority eta pair: %f -> %f", ctx.EtaIncident, ctx.EtaTransmit)
	}
}

func TestApplyMediumTransmissionThinBoundaryDoesNotMutateStack(t *testing.T) {
	registry := medium.NewRegistry()
	glassID, err := registry.RegisterHomogeneous("glass", medium.NewConstant(1.5))
	if err != nil {
		t.Fatalf("register glass: %v", err)
	}
	ray := &renderray.Ray{}
	ray.MediumStack.Reset(medium.MediumAir)
	boundary := medium.Boundary{Outside: medium.MediumAir, Inside: glassID, Thin: true, Priority: 4}
	ctx := bxdf.ShadingContext{Entering: true, TransmitMedium: glassID}

	applyMediumTransmission(registry, ray, ctx, boundary, bxdf.BxDFSample{
		Flags:          bxdf.DeltaTransmission,
		TransmitMedium: glassID,
	})

	if got := ray.MediumStack.Current(); got != medium.MediumAir {
		t.Fatalf("thin boundary should not push stack medium, got %d", got)
	}
	if got := ray.RefractionIndex; got != 1 {
		t.Fatalf("thin boundary should leave current ray IOR at air, got %f", got)
	}
}
