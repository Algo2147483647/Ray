package optics

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestConvertToMonochromeMonteCarlo(t *testing.T) {
	const numSamples = 200000

	wavelengths := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		ray := &Ray{
			Color: mat.NewVecDense(3, []float64{1.0, 1.0, 1.0}),
		}
		ray.ConvertToMonochrome()

		wavelengths[i] = ray.WaveLength
		for ch := 0; ch < 3; ch++ {
			if ray.Color.AtVec(ch) != 1 {
				t.Fatalf("expected spectral ray throughput to stay scalar-white before film conversion, got %v", ray.Color.RawVector().Data)
			}
		}
	}

	validateDistribution(t, wavelengths)
}

func validateDistribution(t *testing.T, wavelengths []float64) {
	minWavelength, maxWavelength := math.Inf(1), math.Inf(-1)
	for _, w := range wavelengths {
		if w < minWavelength {
			minWavelength = w
		}
		if w > maxWavelength {
			maxWavelength = w
		}
	}

	if minWavelength < WavelengthMin || maxWavelength > WavelengthMax {
		t.Errorf("Wavelength out of range: [%f, %f], expected range: [%f, %f]",
			minWavelength, maxWavelength, WavelengthMin, WavelengthMax)
	}
}

func TestRGBWeightWhitePoint(t *testing.T) {
	const samples = 10000
	var sum [3]float64
	for i := 0; i < samples; i++ {
		t := (float64(i) + 0.5) / samples
		weight := RGBWeight(WavelengthMin + t*(WavelengthMax-WavelengthMin))
		sum[0] += weight.AtVec(0)
		sum[1] += weight.AtVec(1)
		sum[2] += weight.AtVec(2)
	}
	for ch, total := range sum {
		mean := total / samples
		if math.Abs(mean-1) > 1e-3 {
			t.Fatalf("channel %d white point mean = %f, want 1", ch, mean)
		}
	}
}

func TestSpectralPowerToXYZWhitePoint(t *testing.T) {
	const samples = 10000
	var sum [3]float64
	for i := 0; i < samples; i++ {
		t := (float64(i) + 0.5) / samples
		xyz := SpectralPowerToXYZ(WavelengthMin+t*(WavelengthMax-WavelengthMin), UniformWavelengthPDF(), 1)
		sum[0] += xyz.AtVec(0)
		sum[1] += xyz.AtVec(1)
		sum[2] += xyz.AtVec(2)
	}

	want := d65WhiteXYZ
	for ch, total := range sum {
		mean := total / samples
		if math.Abs(mean-want[ch]) > 1e-3 {
			t.Fatalf("channel %d normalized XYZ white point = %f, want %f", ch, mean, want[ch])
		}
	}
}

func TestRayInitResetsReusedThroughput(t *testing.T) {
	ray := &Ray{
		Color: mat.NewVecDense(3, []float64{0.2, 0.3, 0.4}),
	}
	ray.WaveLength = 510
	ray.WavelengthPDF = UniformWavelengthPDF()

	ray.Init()

	if ray.WaveLength != 0 || ray.WavelengthPDF != 0 {
		t.Fatalf("expected spectral state reset, got wavelength=%f pdf=%f", ray.WaveLength, ray.WavelengthPDF)
	}
	for i := 0; i < 3; i++ {
		if ray.Color.AtVec(i) != 1 {
			t.Fatalf("expected throughput reset to white, got %v", ray.Color.RawVector().Data)
		}
	}
}

func TestWavelengthToXYZHasExpectedPrimaryRegions(t *testing.T) {
	blue := WavelengthToXYZ(450)
	green := WavelengthToXYZ(555)
	red := WavelengthToXYZ(610)

	if blue.AtVec(2) <= blue.AtVec(0) || blue.AtVec(2) <= blue.AtVec(1) {
		t.Fatalf("expected 450nm to be dominated by Z, got %v", blue.RawVector().Data)
	}
	if green.AtVec(1) <= green.AtVec(0) || green.AtVec(1) <= green.AtVec(2) {
		t.Fatalf("expected 555nm to be dominated by Y, got %v", green.RawVector().Data)
	}
	if red.AtVec(0) <= red.AtVec(1) || red.AtVec(0) <= red.AtVec(2) {
		t.Fatalf("expected 610nm to be dominated by X, got %v", red.RawVector().Data)
	}
}
