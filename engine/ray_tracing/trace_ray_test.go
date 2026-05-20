package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/material/core"
	"github.com/Algo2147483647/ray/engine/model/material/ior"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
)

func TestPrepareMediumContextKeepsLegacyIORWithoutBoundary(t *testing.T) {
	ray := &renderray.Ray{RefractionIndex: 1.5}
	ray.MediumStack.Reset(core.MediumAir)
	ctx := core.ShadingContext{CurrentIOR: ray.RefractionIndex}

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
	glassID, err := registry.RegisterHomogeneous("glass", ior.NewConstant(1.5), nil, nil)
	if err != nil {
		t.Fatalf("register glass: %v", err)
	}
	ray := &renderray.Ray{}
	ray.MediumStack.Reset(core.MediumAir)
	ctx := core.ShadingContext{}

	prepareMediumContext(&ctx, registry, ray, medium.NewBoundary(core.MediumAir, glassID), true)

	if ctx.IncidentMedium != core.MediumAir || ctx.TransmitMedium != glassID {
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
	waterID, err := registry.RegisterHomogeneous("water", ior.NewConstant(1.33), nil, nil)
	if err != nil {
		t.Fatalf("register water: %v", err)
	}
	glassID, err := registry.RegisterHomogeneous("glass", ior.NewConstant(1.5), nil, nil)
	if err != nil {
		t.Fatalf("register glass: %v", err)
	}

	ray := &renderray.Ray{}
	ray.MediumStack.Reset(core.MediumAir)
	ray.MediumStack.EnterBoundary(medium.Boundary{Inside: glassID, Priority: 10})
	ctx := core.ShadingContext{}

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
	glassID, err := registry.RegisterHomogeneous("glass", ior.NewConstant(1.5), nil, nil)
	if err != nil {
		t.Fatalf("register glass: %v", err)
	}
	ray := &renderray.Ray{}
	ray.MediumStack.Reset(core.MediumAir)
	boundary := medium.Boundary{Outside: core.MediumAir, Inside: glassID, Thin: true, Priority: 4}
	ctx := core.ShadingContext{Entering: true, TransmitMedium: glassID}

	applyMediumTransmission(registry, ray, ctx, boundary, core.BxDFSample{
		Flags:          core.DeltaTransmission,
		TransmitMedium: glassID,
	})

	if got := ray.MediumStack.Current(); got != core.MediumAir {
		t.Fatalf("thin boundary should not push stack medium, got %d", got)
	}
	if got := ray.RefractionIndex; got != 1 {
		t.Fatalf("thin boundary should leave current ray IOR at air, got %f", got)
	}
}
