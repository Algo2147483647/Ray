package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math/rand/v2"

	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/object"
	"gonum.org/v1/gonum/mat"
)

func (h *Handler) TracePixel(camera camera.Camera, objTree *object.ObjectTree, samples int64, index ...int) *mat.VecDense {
	color := mat.NewVecDense(3, nil)
	ray := h.RayPool.Get().(*renderray.Ray) // new ray
	defer h.RayPool.Put(ray)

	wavelengthSampler := NewUniformWavelengthSampler()
	totalTraces := int64(0)

	for s := int64(0); s < samples; s++ {
		switch h.SpectrumMode {
		case bxdf.SpectrumRGB:
			camera.GenerateRay(ray, index...)
			ray.DisableSpectralSampling()
			color.AddVec(color, h.TraceRay(objTree, ray, 0))
			totalTraces++

		case bxdf.SpectrumRGBAndSpectral:
			wavelengthSamples := h.WavelengthSamples
			if wavelengthSamples <= 0 {
				wavelengthSamples = 4
			}
			for w := 0; w < wavelengthSamples; w++ {
				camera.GenerateRay(ray, index...)
				sample := wavelengthSampler.Sample((float64(w) + rand.Float64()) / float64(wavelengthSamples))
				ray.SetSpectralWavelength(sample.LambdaNM)
				color.AddVec(color, spectralRayToXYZ(h.TraceRay(objTree, ray, 0), ray))
				totalTraces++
			}

		default:
			camera.GenerateRay(ray, index...)
			sample := wavelengthSampler.Sample(rand.Float64())
			ray.SetSpectralWavelength(sample.LambdaNM)
			color.AddVec(color, spectralRayToXYZ(h.TraceRay(objTree, ray, 0), ray))
			totalTraces++
		}
	}

	if totalTraces == 0 {
		return color
	}
	return math_lib.ScaleVec(color, 1.0/float64(totalTraces), color)
}

func spectralRayToXYZ(color *mat.VecDense, ray *renderray.Ray) *mat.VecDense {
	if ray == nil || ray.WaveLength <= 0 {
		return mat.NewVecDense(3, nil)
	}
	power := ray.SpectralPower
	xyz := renderray.SpectralPowerToXYZ(ray.WaveLength, ray.WavelengthPDF, power)
	compatibility := ray.RGBCompatibility
	if compatibility == nil {
		compatibility = color
	}
	if !ray.SpectralPath {
		if compatibility == nil || compatibility.Len() < 3 {
			return linearSRGBToXYZ(power, power, power)
		}
		return linearSRGBToXYZ(
			power*compatibility.AtVec(0),
			power*compatibility.AtVec(1),
			power*compatibility.AtVec(2),
		)
	}
	if compatibility == nil || compatibility.Len() < 3 || isWhiteRGB(compatibility) {
		return xyz
	}
	r, g, b := xyzToLinearSRGB(xyz.AtVec(0), xyz.AtVec(1), xyz.AtVec(2))
	return linearSRGBToXYZ(
		r*compatibility.AtVec(0),
		g*compatibility.AtVec(1),
		b*compatibility.AtVec(2),
	)
}

func linearSRGBToXYZ(r, g, b float64) *mat.VecDense {
	return mat.NewVecDense(3, []float64{
		0.4124564*r + 0.3575761*g + 0.1804375*b,
		0.2126729*r + 0.7151522*g + 0.0721750*b,
		0.0193339*r + 0.1191920*g + 0.9503041*b,
	})
}

func xyzToLinearSRGB(x, y, z float64) (float64, float64, float64) {
	return 3.2404542*x - 1.5371385*y - 0.4985314*z,
		-0.9692660*x + 1.8760108*y + 0.0415560*z,
		0.0556434*x - 0.2040259*y + 1.0572252*z
}

func isWhiteRGB(v *mat.VecDense) bool {
	const eps = 1e-9
	return v.Len() >= 3 &&
		abs(v.AtVec(0)-1) <= eps &&
		abs(v.AtVec(1)-1) <= eps &&
		abs(v.AtVec(2)-1) <= eps
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
