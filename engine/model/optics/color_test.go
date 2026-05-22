package optics

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestSpectralRayToXYZUsesScalarPower(t *testing.T) {
	ray := &Ray{
		WaveLength:       555,
		WavelengthPDF:    UniformWavelengthPDF(),
		SpectralPower:    2,
		SpectralPath:     true,
		RGBCompatibility: mat.NewVecDense(3, []float64{1, 1, 1}),
	}
	color := mat.NewVecDense(3, []float64{1, 1, 1})

	got := SpectralRayToXYZ(color, ray)
	want := SpectralPowerToXYZ(555, UniformWavelengthPDF(), 2)

	for ch := 0; ch < 3; ch++ {
		if math.Abs(got.AtVec(ch)-want.AtVec(ch)) > 1e-12 {
			t.Fatalf("unexpected XYZ channel %d: got %f want %f", ch, got.AtVec(ch), want.AtVec(ch))
		}
	}
}

func TestSpectralRayToXYZPreservesChromaticRGBThroughput(t *testing.T) {
	ray := &Ray{
		WaveLength:           610,
		WavelengthPDF:        UniformWavelengthPDF(),
		SpectralPower:        1,
		SpectralPath:         true,
		RGBCompatibilityPath: true,
		RGBCompatibility:     mat.NewVecDense(3, []float64{0.8, 0.1, 0.05}),
	}
	color := mat.NewVecDense(3, []float64{1, 1, 1})

	got := SpectralRayToXYZ(color, ray)
	gotR, gotG, gotB := XYZToLinearSRGB(got.AtVec(0), got.AtVec(1), got.AtVec(2))

	if gotR <= gotG || gotR <= gotB {
		t.Fatalf("expected chromatic RGB throughput to remain red-dominant, got linear RGB [%f %f %f]", gotR, gotG, gotB)
	}
}

func TestSpectralRayToXYZUsesRGBCompatibilityForNonSpectralPath(t *testing.T) {
	ray := &Ray{
		WaveLength:           610,
		WavelengthPDF:        UniformWavelengthPDF(),
		SpectralPower:        1,
		SpectralPath:         false,
		RGBCompatibilityPath: true,
		RGBCompatibility:     mat.NewVecDense(3, []float64{0.8, 0.1, 0.05}),
	}
	color := mat.NewVecDense(3, []float64{1, 1, 1})

	got := SpectralRayToXYZ(color, ray)
	gotR, gotG, gotB := XYZToLinearSRGB(got.AtVec(0), got.AtVec(1), got.AtVec(2))

	if math.Abs(gotR-0.8) > 1e-6 || math.Abs(gotG-0.1) > 1e-6 || math.Abs(gotB-0.05) > 1e-6 {
		t.Fatalf("expected RGB compatibility to bypass spectral conversion, got linear RGB [%f %f %f]", gotR, gotG, gotB)
	}
}

func TestSpectralRayToXYZIgnoresCompatibilityWithoutExplicitFlag(t *testing.T) {
	ray := &Ray{
		WaveLength:       610,
		WavelengthPDF:    UniformWavelengthPDF(),
		SpectralPower:    1,
		SpectralPath:     true,
		RGBCompatibility: mat.NewVecDense(3, []float64{0.1, 0.9, 0.1}),
	}

	got := SpectralRayToXYZ(mat.NewVecDense(3, []float64{1, 1, 1}), ray)
	want := SpectralPowerToXYZ(610, UniformWavelengthPDF(), 1)

	for ch := 0; ch < 3; ch++ {
		if math.Abs(got.AtVec(ch)-want.AtVec(ch)) > 1e-12 {
			t.Fatalf("channel %d used RGB compatibility without explicit flag: got %f want %f", ch, got.AtVec(ch), want.AtVec(ch))
		}
	}
}

func TestSpectralRayToScalarAppliesRGBCompatibilitySpectrally(t *testing.T) {
	ray := &Ray{
		WaveLength:           610,
		WavelengthPDF:        UniformWavelengthPDF(),
		SpectralPower:        2,
		SpectralPath:         true,
		RGBCompatibilityPath: true,
		RGBCompatibility:     mat.NewVecDense(3, []float64{0.8, 0.1, 0.05}),
	}

	got := SpectralRayToScalar(mat.NewVecDense(3, []float64{1, 1, 1}), ray)
	want := 2 * NewRGBSpectrum(0.8, 0.1, 0.05).RGBPowerAtWavelength(610)
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("unexpected scalar spectral contribution: got %f want %f", got, want)
	}
}

func TestACEScgRoundTripLinearSRGB(t *testing.T) {
	r, g, b := LinearSRGBToACEScg(0.25, 0.5, 0.75)
	gotR, gotG, gotB := ACEScgToLinearSRGB(r, g, b)

	if math.Abs(gotR-0.25) > 1e-6 || math.Abs(gotG-0.5) > 1e-6 || math.Abs(gotB-0.75) > 1e-6 {
		t.Fatalf("ACEScg round trip drifted: got [%f %f %f]", gotR, gotG, gotB)
	}
}
