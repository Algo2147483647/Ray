package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"testing"

	"github.com/Algo2147483647/ray/engine/material/core"
	"github.com/Algo2147483647/ray/engine/material/ior"
	"github.com/Algo2147483647/ray/engine/material/medium"
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
