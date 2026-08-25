package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/model/optics"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math"
)

const minRussianRouletteSurvival = 0.05

func applySpectrum(ray *renderray.Ray, spectrum optics.Spectrum) {
	if ray == nil {
		return
	}
	if ray.Path.Wavelength != nil {
		if !spectrum.HasSamples() {
			spectrum = spectrum.UpliftRGBToSampled([]float64{ray.Path.Wavelength.LambdaNM})
		}
		ray.Path.Throughput = ray.Path.Throughput.Mul(spectrum)
		return
	}
	if spectrum.HasSamples() {
		terminateRay(ray)
		return
	}
	ray.Path.Throughput = ray.Path.Throughput.Mul(spectrum)
}

// accumulateEmission adds beta * Le to the path result without changing beta.
func accumulateEmission(ray *renderray.Ray, emitted optics.Spectrum) {
	if ray == nil || emitted.IsZero() {
		return
	}
	if ray.Path.Wavelength != nil {
		if !emitted.HasSamples() {
			emitted = emitted.UpliftRGBToSampled([]float64{ray.Path.Wavelength.LambdaNM})
		}
	} else if emitted.HasSamples() {
		return
	}
	ray.Path.Radiance = ray.Path.Radiance.Add(ray.Path.Throughput.Mul(emitted))
}

func terminateRay(ray *renderray.Ray) {
	if ray == nil {
		return
	}
	ray.Path.Throughput = ray.Path.Throughput.MulScalar(0)
	return
}

func russianRouletteSurvivalProbability(ray *renderray.Ray) float64 {
	if ray == nil {
		return 0
	}
	return clampSurvival(finiteNonNegative(ray.Path.Throughput.MaxComponent()))
}

func scaleRayThroughput(ray *renderray.Ray, scale float64) {
	if ray == nil || scale == 1 {
		return
	}
	ray.Path.Throughput = ray.Path.Throughput.MulScalar(scale)
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
