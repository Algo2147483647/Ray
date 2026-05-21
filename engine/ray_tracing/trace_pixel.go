package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math"
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
	if !isNeutralRGB(color) {
		return linearSRGBToXYZ(color.AtVec(0), color.AtVec(1), color.AtVec(2))
	}
	return renderray.SpectralPowerToXYZ(ray.WaveLength, ray.WavelengthPDF, averageVec(color))
}

func isNeutralRGB(v *mat.VecDense) bool {
	if v == nil || v.Len() < 3 {
		return true
	}
	const eps = 1e-9
	return math.Abs(v.AtVec(0)-v.AtVec(1)) <= eps &&
		math.Abs(v.AtVec(1)-v.AtVec(2)) <= eps
}

func linearSRGBToXYZ(r, g, b float64) *mat.VecDense {
	return mat.NewVecDense(3, []float64{
		0.4124564*r + 0.3575761*g + 0.1804375*b,
		0.2126729*r + 0.7151522*g + 0.0721750*b,
		0.0193339*r + 0.1191920*g + 0.9503041*b,
	})
}

func averageVec(v *mat.VecDense) float64 {
	if v == nil || v.Len() == 0 {
		return 0
	}
	sum := 0.0
	for i := 0; i < v.Len(); i++ {
		sum += v.AtVec(i)
	}
	return sum / float64(v.Len())
}
