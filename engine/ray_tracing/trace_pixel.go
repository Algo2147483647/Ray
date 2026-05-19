package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math/rand/v2"

	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/material/core"
	"github.com/Algo2147483647/ray/engine/model/camera"
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
		case core.SpectrumRGB:
			camera.GenerateRay(ray, index...)
			ray.DisableSpectralSampling()
			color.AddVec(color, h.TraceRay(objTree, ray, 0))
			totalTraces++

		case core.SpectrumRGBAndSpectral:
			wavelengthSamples := h.WavelengthSamples
			if wavelengthSamples <= 0 {
				wavelengthSamples = 4
			}
			for w := 0; w < wavelengthSamples; w++ {
				camera.GenerateRay(ray, index...)
				sample := wavelengthSampler.Sample((float64(w) + rand.Float64()) / float64(wavelengthSamples))
				ray.SetSpectralWavelength(sample.LambdaNM)
				color.AddVec(color, h.TraceRay(objTree, ray, 0))
				totalTraces++
			}

		default:
			camera.GenerateRay(ray, index...)
			sample := wavelengthSampler.Sample(rand.Float64())
			ray.SetSpectralWavelength(sample.LambdaNM)
			color.AddVec(color, h.TraceRay(objTree, ray, 0))
			totalTraces++
		}
	}

	if totalTraces == 0 {
		return color
	}
	return math_lib.ScaleVec(color, 1.0/float64(totalTraces), color)
}
