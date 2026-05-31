package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
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

func TestApplySurfaceSampleUpdatesMediumForNonDeltaTransmission(t *testing.T) {
	registry := medium.NewRegistry()
	glassID, err := registry.RegisterHomogeneous("glass", medium.NewConstant(1.5))
	if err != nil {
		t.Fatalf("register glass: %v", err)
	}
	ray := &renderray.Ray{}
	ray.Init()
	ray.MediumStack.Reset(medium.MediumAir)
	ctx := bxdf.ShadingContext{
		Entering:       true,
		TransmitMedium: glassID,
	}
	obj := &object.Object{
		MediumBoundary: medium.Boundary{Outside: medium.MediumAir, Inside: glassID},
	}

	applySurfaceSample(registry, ray, ctx, obj, bxdf.BxDFSample{
		Wi:             maths.NewDirection(0, 0, -1),
		F:              renderray.NewSpectrum(1, 1, 1),
		PDF:            1,
		Flags:          bxdf.TransmissionEvent,
		Eta:            1.5,
		TransmitMedium: glassID,
	})

	if got := ray.MediumStack.Current(); got != glassID {
		t.Fatalf("expected non-delta transmission to enter glass, got medium %d", got)
	}
	if got := ray.RefractionIndex; got != 1.5 {
		t.Fatalf("expected ray IOR to update to glass, got %f", got)
	}
}

func TestPrepareSurfaceInteractionSamplesLambertIn4D(t *testing.T) {
	handler := &Handler{}
	ray := &renderray.Ray{
		Origin:    mat.NewVecDense(4, []float64{-1, 0, 0, 0}),
		Direction: mat.NewVecDense(4, []float64{1, 0, 0, 0}),
	}
	ray.Init()
	ray.Origin.CopyVec(mat.NewVecDense(4, []float64{-1, 0, 0, 0}))
	ray.Direction.CopyVec(mat.NewVecDense(4, []float64{1, 0, 0, 0}))

	obj := &object.Object{
		Material: &material.Material{
			Surface: bsdf.NewSingle(bxdf.NewLambert(renderray.NewSpectrum(1, 1, 1))),
		},
	}
	hit := &object.SurfaceHit{
		Distance:      1,
		Point:         mat.NewVecDense(4, []float64{0, 0, 0, 0}),
		ShadingNormal: mat.NewVecDense(4, []float64{-1, 0, 0, 0}),
		FrontFace:     true,
		Object:        obj,
	}

	si, ok := handler.prepareSurfaceInteraction(medium.NewRegistry(), ray, hit)
	if !ok {
		t.Fatal("expected 4D surface interaction to prepare")
	}
	if si.WoLocal.Len() != 4 {
		t.Fatalf("expected 4D local outgoing direction, got %dD", si.WoLocal.Len())
	}
	sample, ok := sampleSurface(obj, si.Context, si.WoLocal)
	if !ok {
		t.Fatal("expected 4D Lambert sample")
	}
	if sample.Wi.Len() != 4 {
		t.Fatalf("expected 4D sampled direction, got %dD", sample.Wi.Len())
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
	ray := &renderray.Ray{Color: renderray.RGB{1, 1, 1}}
	ray.Init()
	ray.MediumStack.Reset(waterID)

	applyMediumAbsorption(registry, ray, 2, bxdf.ShadingContext{})

	want := math.Exp(-1)
	for ch := 0; ch < 3; ch++ {
		if math.Abs(ray.Color[ch]-want) > 1e-12 {
			t.Fatalf("channel %d: got %f want %f", ch, ray.Color[ch], want)
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

func TestApplySpectrumMarksRGBCompatibilityExplicitly(t *testing.T) {
	ray := &renderray.Ray{}
	ray.Init()
	ray.SetSpectralWavelength(610)

	applySpectrum(ray, renderray.NewRGBSpectrum(0.8, 0.1, 0.05))

	if !ray.RGBCompatibilityPath {
		t.Fatal("expected RGB compatibility path to be marked explicitly")
	}
	if ray.SpectralPath {
		t.Fatal("RGB compatibility should not masquerade as spectral throughput")
	}
	if math.Abs(ray.SpectralPower-1) > 1e-12 {
		t.Fatalf("expected spectral power to stay scalar, got %f", ray.SpectralPower)
	}
}

func TestApplySpectrumRejectsSampledSpectrumWithoutWavelength(t *testing.T) {
	ray := &renderray.Ray{Color: renderray.RGB{1, 1, 1}}

	applySpectrum(ray, renderray.NewSampledSpectrum([]float64{0.2, 1.0}))

	for ch := 0; ch < 3; ch++ {
		if ray.Color[ch] != 0 {
			t.Fatalf("expected sampled spectrum without wavelength context to be rejected, got color %v", ray.Color)
		}
	}
}

func TestRussianRouletteSurvivalUsesRGBThroughputMax(t *testing.T) {
	ray := &renderray.Ray{Color: renderray.RGB{0.2, 0.8, 0.4}}

	if got := russianRouletteSurvivalProbability(ray); math.Abs(got-0.8) > 1e-12 {
		t.Fatalf("unexpected RGB survival probability: got %f want 0.8", got)
	}
}

func TestRussianRouletteSurvivalClampsLowThroughput(t *testing.T) {
	ray := &renderray.Ray{Color: renderray.RGB{0.001, 0.002, 0.003}}

	if got := russianRouletteSurvivalProbability(ray); math.Abs(got-minRussianRouletteSurvival) > 1e-12 {
		t.Fatalf("expected low throughput survival clamp, got %f", got)
	}
}

func TestRussianRouletteScalesRGBThroughput(t *testing.T) {
	ray := &renderray.Ray{Color: renderray.RGB{0.2, 0.4, 0.6}}

	scaleRayThroughput(ray, 2)

	if math.Abs(ray.Color[0]-0.4) > 1e-12 ||
		math.Abs(ray.Color[1]-0.8) > 1e-12 ||
		math.Abs(ray.Color[2]-1.2) > 1e-12 {
		t.Fatalf("unexpected scaled RGB throughput: %v", ray.Color)
	}
}

func TestRussianRouletteScalesSpectralThroughput(t *testing.T) {
	ray := &renderray.Ray{}
	ray.Init()
	ray.SetSpectralWavelength(550)
	ray.SpectralPower = 0.25

	scaleRayThroughput(ray, 4)

	if math.Abs(ray.SpectralPower-1) > 1e-12 {
		t.Fatalf("unexpected scaled spectral throughput: %f", ray.SpectralPower)
	}
}

func TestRussianRouletteDepthDefaultsToThirdBounce(t *testing.T) {
	handler := &Handler{}

	if handler.shouldApplyRussianRoulette(2) {
		t.Fatal("did not expect russian roulette before third bounce")
	}
	if !handler.shouldApplyRussianRoulette(3) {
		t.Fatal("expected russian roulette at third bounce")
	}
}

func TestTerminateBeforeBounceStopsPastMaxDepth(t *testing.T) {
	handler := &Handler{MaxRayLevel: 2}
	ray := &renderray.Ray{Color: renderray.RGB{1, 1, 1}}

	if !handler.terminateBeforeBounce(ray, 3) {
		t.Fatal("expected path to terminate beyond max depth")
	}
	if ray.Color != (renderray.RGB{}) {
		t.Fatalf("expected terminated ray color to be cleared, got %v", ray.Color)
	}
}

func TestTerminateBeforeBounceAllowsFullSurvivalRoulette(t *testing.T) {
	handler := &Handler{MaxRayLevel: 8, RussianRouletteDepth: 3}
	ray := &renderray.Ray{Color: renderray.RGB{1, 1, 1}}

	if handler.terminateBeforeBounce(ray, 3) {
		t.Fatal("did not expect roulette to terminate a unit-throughput RGB path")
	}
	if ray.Color != (renderray.RGB{1, 1, 1}) {
		t.Fatalf("unexpected ray throughput after guaranteed survival: %v", ray.Color)
	}
}

type sampledCoefficient struct {
	value float64
}

func (c sampledCoefficient) Eval(medium.WavelengthContext) medium.CoefficientSpectrum {
	return medium.NewSampledCoefficientSpectrum([]float64{c.value})
}
