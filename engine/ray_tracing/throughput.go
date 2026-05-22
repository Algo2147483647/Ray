package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/model/optics"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
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
		ensureRGBCompatibility(ray)
		ray.RGBCompatibility.SetVec(0, ray.RGBCompatibility.AtVec(0)*spectrum.RGBChannel(0))
		ray.RGBCompatibility.SetVec(1, ray.RGBCompatibility.AtVec(1)*spectrum.RGBChannel(1))
		ray.RGBCompatibility.SetVec(2, ray.RGBCompatibility.AtVec(2)*spectrum.RGBChannel(2))
		ray.RGBCompatibilityPath = true
		return
	}

	color := ray.Color
	if spectrum.HasSamples() {
		color.Zero()
		return
	}
	color.SetVec(0, color.AtVec(0)*spectrum.RGBChannel(0))
	color.SetVec(1, color.AtVec(1)*spectrum.RGBChannel(1))
	color.SetVec(2, color.AtVec(2)*spectrum.RGBChannel(2))
}

func terminateRay(ray *renderray.Ray) *mat.VecDense {
	if ray == nil {
		return mat.NewVecDense(3, nil)
	}
	if ray.Color == nil {
		ray.Color = mat.NewVecDense(3, nil)
	} else {
		ray.Color.ScaleVec(0, ray.Color)
	}
	ray.SpectralPower = 0
	ray.SpectralPath = false
	ray.RGBCompatibilityPath = false
	if ray.RGBCompatibility != nil {
		ray.RGBCompatibility.ScaleVec(0, ray.RGBCompatibility)
	}
	return ray.Color
}

func ensureRGBCompatibility(ray *renderray.Ray) {
	if ray.RGBCompatibility == nil || ray.RGBCompatibility.Len() != 3 {
		ray.RGBCompatibility = mat.NewVecDense(3, []float64{1, 1, 1})
	}
}

func russianRouletteSurvivalProbability(ray *renderray.Ray) float64 {
	if ray == nil {
		return 0
	}
	if ray.WaveLength > 0 {
		throughput := finiteNonNegative(ray.SpectralPower)
		if ray.RGBCompatibilityPath && ray.RGBCompatibility != nil && ray.RGBCompatibility.Len() == 3 {
			throughput *= maxVecChannel(ray.RGBCompatibility)
		}
		return clampSurvival(throughput)
	}
	if ray.Color == nil || ray.Color.Len() == 0 {
		return 0
	}
	return clampSurvival(maxVecChannel(ray.Color))
}

func scaleRayThroughput(ray *renderray.Ray, scale float64) {
	if ray == nil || scale == 1 {
		return
	}
	if ray.WaveLength > 0 {
		ray.SpectralPower *= scale
		return
	}
	if ray.Color != nil {
		ray.Color.ScaleVec(scale, ray.Color)
	}
}

func maxVecChannel(v *mat.VecDense) float64 {
	if v == nil || v.Len() == 0 {
		return 0
	}
	maxValue := 0.0
	for i := 0; i < v.Len(); i++ {
		maxValue = math.Max(maxValue, finiteNonNegative(v.AtVec(i)))
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
