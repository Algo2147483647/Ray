package optics

import (
	"math"
	"testing"
)

func TestConvertToMonochromeMonteCarlo(t *testing.T) {
	const numSamples = 200000

	wavelengths := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		ray := &Ray{
			Color: RGB{1.0, 1.0, 1.0},
		}
		ray.ConvertToMonochrome()

		wavelengths[i] = ray.WaveLength
		for ch := 0; ch < 3; ch++ {
			if ray.Color[ch] != 1 {
				t.Fatalf("expected spectral ray throughput to stay scalar-white before film conversion, got %v", ray.Color)
			}
		}
		if ray.SpectralPower != 1 {
			t.Fatalf("expected spectral power to start at 1, got %f", ray.SpectralPower)
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
		sum[0] += weight[0]
		sum[1] += weight[1]
		sum[2] += weight[2]
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
		sum[0] += xyz[0]
		sum[1] += xyz[1]
		sum[2] += xyz[2]
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
		Color:            RGB{0.2, 0.3, 0.4},
		RGBCompatibility: RGB{0.5, 0.6, 0.7},
		SpectralPower:    0.25,
	}
	ray.WaveLength = 510
	ray.WavelengthPDF = UniformWavelengthPDF()

	ray.Init()

	if ray.WaveLength != 0 || ray.WavelengthPDF != 0 {
		t.Fatalf("expected spectral state reset, got wavelength=%f pdf=%f", ray.WaveLength, ray.WavelengthPDF)
	}
	for i := 0; i < 3; i++ {
		if ray.Color[i] != 1 {
			t.Fatalf("expected throughput reset to white, got %v", ray.Color)
		}
		if ray.RGBCompatibility[i] != 1 {
			t.Fatalf("expected RGB compatibility reset to white, got %v", ray.RGBCompatibility)
		}
	}
	if ray.SpectralPower != 1 {
		t.Fatalf("expected spectral power reset to 1, got %f", ray.SpectralPower)
	}
	if ray.SpectralPath {
		t.Fatal("expected spectral path marker to reset")
	}
	if ray.RGBCompatibilityPath {
		t.Fatal("expected RGB compatibility path marker to reset")
	}
}

func TestRaySetSpectralSamplePreservesSamplerPDF(t *testing.T) {
	ray := &Ray{}

	ray.SetSpectralSample(WavelengthSample{LambdaNM: 520, PDF: 0.0123})

	if ray.WaveLength != 520 {
		t.Fatalf("unexpected wavelength: %f", ray.WaveLength)
	}
	if math.Abs(ray.WavelengthPDF-0.0123) > 1e-12 {
		t.Fatalf("expected sampler pdf to be preserved, got %f", ray.WavelengthPDF)
	}
}

func TestWavelengthToXYZHasExpectedPrimaryRegions(t *testing.T) {
	blue := WavelengthToXYZ(450)
	green := WavelengthToXYZ(555)
	red := WavelengthToXYZ(610)

	if blue[2] <= blue[0] || blue[2] <= blue[1] {
		t.Fatalf("expected 450nm to be dominated by Z, got %v", blue)
	}
	if green[1] <= green[0] || green[1] <= green[2] {
		t.Fatalf("expected 555nm to be dominated by Y, got %v", green)
	}
	if red[0] <= red[1] || red[0] <= red[2] {
		t.Fatalf("expected 610nm to be dominated by X, got %v", red)
	}
}
