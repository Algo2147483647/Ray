package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"gonum.org/v1/gonum/mat"
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

func TestApplyMediumAbsorptionUsesBeerLambertRGB(t *testing.T) {
	registry := medium.NewRegistry()
	waterID, err := registry.RegisterHomogeneousWithCoefficients(
		"water",
		medium.NewConstant(1.33),
		medium.ConstantCoefficient(0.5),
		nil,
	)
	if err != nil {
		t.Fatalf("register water: %v", err)
	}
	ray := &renderray.Ray{Color: mat.NewVecDense(3, []float64{1, 1, 1})}
	ray.Init()
	ray.MediumStack.Reset(waterID)

	applyMediumAbsorption(registry, ray, 2, bxdf.ShadingContext{})

	want := math.Exp(-1)
	for ch := 0; ch < 3; ch++ {
		if math.Abs(ray.Color.AtVec(ch)-want) > 1e-12 {
			t.Fatalf("channel %d: got %f want %f", ch, ray.Color.AtVec(ch), want)
		}
	}
}

func TestApplyMediumAbsorptionUsesSpectralPowerForSampledSigmaA(t *testing.T) {
	registry := medium.NewRegistry()
	filterID, err := registry.RegisterHomogeneousWithCoefficients(
		"filter",
		medium.NewConstant(1),
		sampledCoefficient{value: 0.25},
		nil,
	)
	if err != nil {
		t.Fatalf("register filter: %v", err)
	}
	ray := &renderray.Ray{}
	ray.Init()
	ray.SetSpectralWavelength(550)
	ray.MediumStack.Reset(filterID)

	applyMediumAbsorption(registry, ray, 4, bxdf.ShadingContext{
		SpectrumMode:  renderray.SpectrumModeHeroWavelength,
		WavelengthNM:  550,
		WavelengthsNM: []float64{550},
	})

	want := math.Exp(-1)
	if math.Abs(ray.SpectralPower-want) > 1e-12 {
		t.Fatalf("got spectral power %f want %f", ray.SpectralPower, want)
	}
	if !ray.SpectralPath {
		t.Fatal("expected sampled absorption to mark spectral path")
	}
}

type sampledCoefficient struct {
	value float64
}

func (c sampledCoefficient) Eval(medium.WavelengthContext) medium.CoefficientSpectrum {
	return medium.NewSampledCoefficientSpectrum([]float64{c.value})
}
