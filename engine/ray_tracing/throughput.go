package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/model/optics"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math"
)

const minRussianRouletteSurvival = 0.05

func applySpectrum(ray *renderray.Ray, spectrum optics.Spectrum) {
	if ray.WaveLength > 0 {
		if spectrum.HasSamples() {
			ray.SpectralPower *= spectrum.Sample(0)
			ray.SpectralPath = true
			return
		}
		ray.RGBCompatibility = ray.RGBCompatibility.Mul(spectrum.RGB)
		ray.RGBCompatibilityPath = true
		return
	}

	if spectrum.HasSamples() {
		ray.Color = optics.RGB{}
		return
	}
	ray.Color = ray.Color.Mul(spectrum.RGB)
}

func terminateRay(ray *renderray.Ray) {
	if ray == nil {
		return
	}
	ray.Color = optics.RGB{}
	ray.SpectralPower = 0
	ray.SpectralPath = false
	ray.RGBCompatibilityPath = false
	ray.RGBCompatibility = optics.RGB{}
	return
}

func russianRouletteSurvivalProbability(ray *renderray.Ray) float64 {
	if ray == nil {
		return 0
	}
	if ray.WaveLength > 0 {
		throughput := finiteNonNegative(ray.SpectralPower)
		if ray.RGBCompatibilityPath {
			throughput *= maxRGBChannel(ray.RGBCompatibility)
		}
		return clampSurvival(throughput)
	}
	return clampSurvival(maxRGBChannel(ray.Color))
}

func scaleRayThroughput(ray *renderray.Ray, scale float64) {
	if ray == nil || scale == 1 {
		return
	}
	if ray.WaveLength > 0 {
		ray.SpectralPower *= scale
		return
	}
	ray.Color = ray.Color.MulScalar(scale)
}

func maxRGBChannel(v optics.RGB) float64 {
	maxValue := 0.0
	for i := 0; i < 3; i++ {
		maxValue = math.Max(maxValue, finiteNonNegative(v[i]))
	}
	return maxValue
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
