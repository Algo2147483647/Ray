package optics

import (
	"math"
	"testing"
)

func TestConvertToMonochromeMonteCarlo(t *testing.T) {
	const numSamples = 200000

	wavelengths := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		ray := &Ray{}
		ray.ConvertToMonochrome()

		wavelengths[i] = ray.Path.Wavelength.LambdaNM
		if ray.Path.Throughput != 1 {
			t.Fatalf("expected unit sampled throughput, got %+v", ray.Path)
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

func TestSpectralPowerToXYZUsesCommonLuminanceNormalization(t *testing.T) {
	const samples = 10000
	var sum [3]float64
	for i := 0; i < samples; i++ {
		t := (float64(i) + 0.5) / samples
		xyz := SpectralPowerToXYZ(WavelengthMin+t*(WavelengthMax-WavelengthMin), UniformWavelengthPDF(), 1)
		sum[0] += xyz[0]
		sum[1] += xyz[1]
		sum[2] += xyz[2]
	}

	want := [3]float64{
		spectralXYZWhitePoint[0] / spectralXYZWhitePoint[1],
		1,
		spectralXYZWhitePoint[2] / spectralXYZWhitePoint[1],
	}
	for ch, total := range sum {
		mean := total / samples
		if math.Abs(mean-want[ch]) > 1e-3 {
			t.Fatalf("channel %d normalized XYZ white point = %f, want %f", ch, mean, want[ch])
		}
	}
}

func TestSpectralPowerToXYZKeeps6500KBlackbodyNearNeutral(t *testing.T) {
	const (
		samples     = 20000
		temperature = 6500.0
		c2NMK       = 1.438776877e7
	)
	var xyz XYZ
	for i := 0; i < samples; i++ {
		t := (float64(i) + 0.5) / samples
		wavelength := WavelengthMin + t*(WavelengthMax-WavelengthMin)
		power := 1 / (math.Pow(wavelength, 5) * math.Expm1(c2NMK/(wavelength*temperature)))
		sample := SpectralPowerToXYZ(wavelength, UniformWavelengthPDF(), power)
		xyz = xyz.Add(sample)
	}
	xyz = xyz.MulScalar(1 / float64(samples))
	r, g, b := XYZToLinearSRGB(xyz[0], xyz[1], xyz[2])
	maximum := math.Max(r, math.Max(g, b))
	minimum := math.Min(r, math.Min(g, b))
	if maximum <= 0 || (maximum-minimum)/maximum > 0.1 {
		t.Fatalf("6500 K blackbody is not near neutral: RGB=[%g %g %g]", r, g, b)
	}
}

func TestRayInitResetsReusedThroughput(t *testing.T) {
	ray := &Ray{Path: PathState{
		Throughput: 0.25,
		Wavelength: &WavelengthSample{LambdaNM: 510, PDF: UniformWavelengthPDF()},
	}}

	ray.Init()

	if ray.Path.Wavelength != nil {
		t.Fatalf("expected wavelength to be reset, got %+v", ray.Path)
	}
	if ray.Path.Throughput != 1 {
		t.Fatalf("expected throughput reset to one, got %+v", ray.Path.Throughput)
	}
}

func TestRaySetSpectralSamplePreservesSamplerPDF(t *testing.T) {
	ray := &Ray{}

	ray.SetSpectralSample(WavelengthSample{LambdaNM: 520, PDF: 0.0123})

	if ray.Path.Wavelength == nil || ray.Path.Wavelength.LambdaNM != 520 {
		t.Fatalf("unexpected wavelength: %+v", ray.Path.Wavelength)
	}
	if math.Abs(ray.Path.Wavelength.PDF-0.0123) > 1e-12 {
		t.Fatalf("expected sampler pdf to be preserved, got %f", ray.Path.Wavelength.PDF)
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
