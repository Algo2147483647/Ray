package ray_tracing

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/emission"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
)

func TestSegmentTransmittanceAppliesBeerLambertToRGBAndSampledSpectra(t *testing.T) {
	registry := medium.NewRegistry()
	absorbingID, err := registry.RegisterHomogeneousWithCoefficients(
		"absorbing",
		medium.NewConstant(1),
		medium.ConstantCoefficient(0.5),
		nil,
	)
	if err != nil {
		t.Fatalf("register absorbing medium: %v", err)
	}

	rgb := evaluateSegmentTransmittance(
		registry,
		absorbingID,
		2,
		bxdf.ShadingContext{SpectrumMode: optics.SpectrumModeRGB},
	).ApplyToSpectrum(optics.ConstantSpectrum(2))
	want := 2 * math.Exp(-1)
	for channel := range 3 {
		if math.Abs(rgb.RGBChannel(channel)-want) > 1e-12 {
			t.Fatalf("RGB channel %d = %g, want %g", channel, rgb.RGBChannel(channel), want)
		}
	}

	sampled := evaluateSegmentTransmittance(
		registry,
		absorbingID,
		2,
		bxdf.ShadingContext{
			SpectrumMode:  optics.SpectrumModeHeroWavelength,
			WavelengthNM:  550,
			WavelengthsNM: []float64{550},
		},
	).ApplyToSpectrum(optics.NewSampledSpectrum([]float64{2}))
	if !sampled.HasSamples() || math.Abs(sampled.Sample(0)-want) > 1e-12 {
		t.Fatalf("sampled transmittance = %+v, want %g", sampled, want)
	}
}

func TestBuildLightSubpathAppliesHomogeneousAbsorptionBeforeVertex(t *testing.T) {
	const sigmaA = 0.25
	registry := absorbingAirRegistry(t, sigmaA)
	emitterShape := shape.NewCircle(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		0.1,
	)
	emitter := &object.Object{
		Shape: emitterShape,
		Material: &material.Material{
			Emission: emission.NewConstant(optics.ConstantSpectrum(1)),
		},
	}
	upperReceiver := &object.Object{
		Shape: &shape.Plane{
			A: mat.NewVecDense(3, []float64{0, 0, 1}),
			B: -1,
		},
		Material: &material.Material{},
	}
	lowerReceiver := &object.Object{
		Shape: &shape.Plane{
			A: mat.NewVecDense(3, []float64{0, 0, 1}),
			B: 1,
		},
		Material: &material.Material{},
	}
	tree := &object.ObjectTree{Media: registry}
	tree.AddObject(emitter)
	tree.AddObject(upperReceiver)
	tree.AddObject(lowerReceiver)
	tree.Build()

	handler := NewHandler()
	handler.SceneGeometry = geometry.Euclidean()
	handler.SpectrumMode = optics.SpectrumModeRGB
	handler.MaxRayLevel = 1
	lights, totalArea := collectAreaLights(tree)
	if len(lights) != 1 {
		t.Fatalf("area light count = %d, want 1", len(lights))
	}

	for sample := 0; sample < 32; sample++ {
		path := handler.buildLightSubpath(tree, lights, totalArea, 0, 0)
		if len(path) != 2 {
			t.Fatalf("light path vertex count = %d, want 2", len(path))
		}
		distance := math.Sqrt(squaredDistance(path[0].Point, path[1].Point))
		want := 2 * math.Pi * emitterShape.SurfaceArea() * math.Exp(-sigmaA*distance)
		for channel := range 3 {
			if got := path[1].Beta.RGBChannel(channel); math.Abs(got-want) > 1e-10 {
				t.Fatalf("sample %d channel %d beta = %g, want %g", sample, channel, got, want)
			}
		}
	}
}

func TestLightTracingRenderAttenuatesCameraConnectionStatistically(t *testing.T) {
	const (
		sigmaA   = 0.5
		distance = 2.0
		samples  = 4096
	)

	unabsorbed := renderDirectAreaLight(t, 0, samples)
	absorbed := renderDirectAreaLight(t, sigmaA, samples)
	if unabsorbed <= 0 || absorbed <= 0 {
		t.Fatalf("expected positive light-tracing energy, got unabsorbed=%g absorbed=%g", unabsorbed, absorbed)
	}

	gotRatio := absorbed / unabsorbed
	wantRatio := math.Exp(-sigmaA * distance)
	if math.Abs(gotRatio-wantRatio) > 1e-2 {
		t.Fatalf("absorbed/unabsorbed ratio = %g, want approximately %g", gotRatio, wantRatio)
	}
}

func renderDirectAreaLight(t *testing.T, sigmaA float64, samples int64) float64 {
	t.Helper()
	registry := absorbingAirRegistry(t, sigmaA)
	emitter := &object.Object{
		Shape: shape.NewCircle(
			mat.NewVecDense(3, []float64{0, 0, 2}),
			mat.NewVecDense(3, []float64{0, 0, -1}),
			0.001,
		),
		Material: &material.Material{
			Emission: emission.NewConstant(optics.ConstantSpectrum(2)),
		},
	}
	tree := &object.ObjectTree{Media: registry}
	tree.AddObject(emitter)
	tree.Build()

	renderCamera := &camera.Camera3D{
		Position:     mat.NewVecDense(3, []float64{0, 0, 0}),
		Direction:    mat.NewVecDense(3, []float64{0, 0, 1}),
		Up:           mat.NewVecDense(3, []float64{0, 1, 0}),
		Width:        1,
		Height:       1,
		FieldOfViews: []float64{60, 60},
	}
	handler := NewHandler()
	handler.IntegratorKind = IntegratorLightTracing
	handler.SceneGeometry = geometry.Euclidean()
	handler.SpectrumMode = optics.SpectrumModeHeroWavelength
	handler.ThreadNum = 1
	handler.MaxRayLevel = 0
	film := camera.NewFilm(1, 1)

	if err := handler.TraceScene(renderCamera, tree, film, samples, nil); err != nil {
		t.Fatalf("light-tracing render: %v", err)
	}
	if film.Samples != samples {
		t.Fatalf("film samples = %d, want %d", film.Samples, samples)
	}
	var energy float64
	for bin := range film.SpectralBins {
		energy += film.SpectralBins[bin].Data[0]
	}
	return energy
}

func absorbingAirRegistry(t *testing.T, sigmaA float64) *medium.Registry {
	t.Helper()
	registry := medium.NewRegistry()
	registry.Set(
		medium.MediumAir,
		"air",
		medium.NewHomogeneousWithCoefficients(
			medium.MediumAir,
			"air",
			medium.NewConstant(1),
			medium.ConstantCoefficient(sigmaA),
			nil,
		),
	)
	return registry
}
