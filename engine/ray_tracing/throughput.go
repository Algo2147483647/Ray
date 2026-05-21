package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/model/optics"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
)

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
		return
	}

	color := ray.Color
	if spectrum.HasSamples() {
		power := spectrum.Average()
		color.SetVec(0, color.AtVec(0)*power)
		color.SetVec(1, color.AtVec(1)*power)
		color.SetVec(2, color.AtVec(2)*power)
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
