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
		WaveLength:    555,
		WavelengthPDF: optics.UniformWavelengthPDF(),
	}
	color := mat.NewVecDense(3, []float64{2, 2, 2})

	got := spectralRayToXYZ(color, ray)
	want := optics.SpectralPowerToXYZ(555, optics.UniformWavelengthPDF(), 2)

	for ch := 0; ch < 3; ch++ {
		if math.Abs(got.AtVec(ch)-want.AtVec(ch)) > 1e-12 {
			t.Fatalf("unexpected XYZ channel %d: got %f want %f", ch, got.AtVec(ch), want.AtVec(ch))
		}
	}
}
