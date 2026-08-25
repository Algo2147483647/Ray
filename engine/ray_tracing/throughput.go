package ray_tracing

import (
	"math"

	"github.com/Algo2147483647/ray/engine/model/optics"
)

const minRussianRouletteSurvival = 0.05

func applySpectrum(ray *optics.Ray, spectrum optics.Spectrum) {
	if ray == nil {
		return
	}
	power, ok := spectrumPower(ray, spectrum)
	if !ok {
		terminateRay(ray)
		return
	}
	ray.Path.Throughput *= power
}

// accumulateEmission adds beta * Le to the path result without changing beta.
func accumulateEmission(ray *optics.Ray, emitted optics.Spectrum) {
	if ray == nil || emitted.IsZero() {
		return
	}
	power, ok := spectrumPower(ray, emitted)
	if !ok {
		return
	}
	ray.Path.Radiance += ray.Path.Throughput * power
}

func terminateRay(ray *optics.Ray) {
	if ray == nil {
		return
	}
	ray.Path.Throughput = 0
	return
}

func russianRouletteSurvivalProbability(ray *optics.Ray) float64 {
	if ray == nil {
		return 0
	}
	return clampSurvival(finiteNonNegative(ray.Path.Throughput))
}

func scaleRayThroughput(ray *optics.Ray, scale float64) {
	if ray == nil || scale == 1 {
		return
	}
	ray.Path.Throughput *= scale
}

func spectrumPower(ray *optics.Ray, spectrum optics.Spectrum) (float64, bool) {
	if ray == nil || ray.Path.Wavelength == nil {
		return 0, false
	}
	return powerAtWavelength(spectrum, ray.Path.Wavelength.LambdaNM)
}

func powerAtWavelength(spectrum optics.Spectrum, wavelengthNM float64) (float64, bool) {
	power, ok := spectrum.PowerAt(wavelengthNM)
	return power, ok && validNonNegativePower(power)
}

func validNonNegativePower(power float64) bool {
	return !math.IsNaN(power) && !math.IsInf(power, 0) && power >= 0
}

func finiteNonNegative(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0
	}
	return v
}

func clampSurvival(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v < minRussianRouletteSurvival {
		return minRussianRouletteSurvival
	}
	if v > 1 {
		return 1
	}
	return v
}
