package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
	"math"
	"testing"
)

func TestUniformWavelengthSampler(t *testing.T) {
	sampler := NewUniformWavelengthSampler()

	first := sampler.Sample(0)
	middle := sampler.Sample(0.5)
	last := sampler.Sample(1)

	if first.LambdaNM <= optics.WavelengthMin || first.LambdaNM >= optics.WavelengthMax {
		t.Fatalf("first wavelength out of visible range: %f", first.LambdaNM)
	}
	if math.Abs(middle.LambdaNM-((optics.WavelengthMin+optics.WavelengthMax)/2)) > 1e-9 {
		t.Fatalf("unexpected middle wavelength: %f", middle.LambdaNM)
	}
	if last.LambdaNM <= optics.WavelengthMin || last.LambdaNM >= optics.WavelengthMax {
		t.Fatalf("last wavelength out of visible range: %f", last.LambdaNM)
	}
	if math.Abs(middle.PDF-optics.UniformWavelengthPDF()) > 1e-12 {
		t.Fatalf("unexpected wavelength pdf: got %f want %f", middle.PDF, optics.UniformWavelengthPDF())
	}
}

func TestSpectralRayToXYZUsesScalarPower(t *testing.T) {
	ray := &optics.Ray{
		WaveLength:       555,
		WavelengthPDF:    optics.UniformWavelengthPDF(),
		SpectralPower:    2,
		SpectralPath:     true,
		RGBCompatibility: mat.NewVecDense(3, []float64{1, 1, 1}),
	}
	color := mat.NewVecDense(3, []float64{1, 1, 1})

	got := spectralRayToXYZ(color, ray)
	want := optics.SpectralPowerToXYZ(555, optics.UniformWavelengthPDF(), 2)

	for ch := 0; ch < 3; ch++ {
		if math.Abs(got.AtVec(ch)-want.AtVec(ch)) > 1e-12 {
			t.Fatalf("unexpected XYZ channel %d: got %f want %f", ch, got.AtVec(ch), want.AtVec(ch))
		}
	}
}

func TestSpectralRayToXYZPreservesChromaticRGBThroughput(t *testing.T) {
	ray := &optics.Ray{
		WaveLength:       610,
		WavelengthPDF:    optics.UniformWavelengthPDF(),
		SpectralPower:    1,
		SpectralPath:     true,
		RGBCompatibility: mat.NewVecDense(3, []float64{0.8, 0.1, 0.05}),
	}
	color := mat.NewVecDense(3, []float64{1, 1, 1})

	got := spectralRayToXYZ(color, ray)
	gotR, gotG, gotB := xyzToLinearSRGBForTest(got.AtVec(0), got.AtVec(1), got.AtVec(2))

	if gotR <= gotG || gotR <= gotB {
		t.Fatalf("expected chromatic RGB throughput to remain red-dominant, got linear RGB [%f %f %f]", gotR, gotG, gotB)
	}
}

func TestSpectralRayToXYZUsesRGBCompatibilityForNonSpectralPath(t *testing.T) {
	ray := &optics.Ray{
		WaveLength:       610,
		WavelengthPDF:    optics.UniformWavelengthPDF(),
		SpectralPower:    1,
		SpectralPath:     false,
		RGBCompatibility: mat.NewVecDense(3, []float64{0.8, 0.1, 0.05}),
	}
	color := mat.NewVecDense(3, []float64{1, 1, 1})

	got := spectralRayToXYZ(color, ray)
	gotR, gotG, gotB := xyzToLinearSRGBForTest(got.AtVec(0), got.AtVec(1), got.AtVec(2))

	if math.Abs(gotR-0.8) > 1e-6 || math.Abs(gotG-0.1) > 1e-6 || math.Abs(gotB-0.05) > 1e-6 {
		t.Fatalf("expected RGB compatibility to bypass spectral conversion, got linear RGB [%f %f %f]", gotR, gotG, gotB)
	}
}

func xyzToLinearSRGBForTest(x, y, z float64) (float64, float64, float64) {
	return 3.2404542*x - 1.5371385*y - 0.4985314*z,
		-0.9692660*x + 1.8760108*y + 0.0415560*z,
		0.0556434*x - 0.2040259*y + 1.0572252*z
}
