package ray_tracing

import (
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/model"
	rendercamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

type pixelKernel interface {
	sampleSpectral(*Handler, model.RenderTarget, *object.ObjectTree, *optics.Ray, optics.WavelengthSample, ...int) rendercamera.SpectralSample
}

type pathTracingKernel struct{}

func (pathTracingKernel) sampleSpectral(
	h *Handler,
	target model.RenderTarget,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	wavelength optics.WavelengthSample,
	index ...int,
) rendercamera.SpectralSample {
	target.Camera.GenerateRay(ray, target.Film.Shape, index...)
	ray.SetSpectralSample(wavelength)
	h.TraceRay(objTree, ray, 0)
	return rendercamera.SpectralSample{
		WavelengthNM: wavelength.LambdaNM,
		Value: optics.SpectralSampleRadiance(
			optics.SpectralRayToScalar(ray),
			ray.Path.Wavelength.PDF,
		),
	}
}

func (h *Handler) tracePixel(
	kernel pixelKernel,
	context *RenderContext,
	pixel int,
	index ...int,
) {
	for _, sample := range h.traceSpectral(
		kernel,
		context.Target,
		context.ObjectTree,
		context.Samples,
		index...,
	) {
		context.Accumulator.AddSpectral(pixel, sample.WavelengthNM, sample.Value)
	}
}

func (h *Handler) TraceSpectral(
	target model.RenderTarget,
	objTree *object.ObjectTree,
	samples int64,
	index ...int,
) []rendercamera.SpectralSample {
	return h.traceSpectral(pathTracingKernel{}, target, objTree, samples, index...)
}

func (h *Handler) traceSpectral(
	kernel pixelKernel,
	target model.RenderTarget,
	objTree *object.ObjectTree,
	samples int64,
	index ...int,
) []rendercamera.SpectralSample {
	ray := h.RayPool.Get().(*optics.Ray)
	ray.Space = h.Space
	defer h.RayPool.Put(ray)

	wavelengthSampler := h.wavelengthSampler()
	spectralSamples := make([]rendercamera.SpectralSample, 0, h.estimatedSpectralSampleCount(samples))

	wavelengthSamples := h.wavelengthSampleCount()
	for s := int64(0); s < samples; s++ {
		for wavelengthIndex := 0; wavelengthIndex < wavelengthSamples; wavelengthIndex++ {
			u := (float64(wavelengthIndex) + rand.Float64()) / float64(wavelengthSamples)
			spectralSamples = append(spectralSamples, kernel.sampleSpectral(
				h, target, objTree, ray, wavelengthSampler.Sample(u), index...,
			))
		}
	}

	normalizeSpectralSamples(spectralSamples)
	return spectralSamples
}

func (h *Handler) TraceSpectralSample(
	target model.RenderTarget,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	wavelengthSampler optics.WavelengthSampler,
	u float64,
	index ...int,
) rendercamera.SpectralSample {
	return pathTracingKernel{}.sampleSpectral(
		h, target, objTree, ray, wavelengthSampler.Sample(u), index...,
	)
}

func (h *Handler) wavelengthSampleCount() int {
	return h.WavelengthSamples
}

func (h *Handler) estimatedSpectralSampleCount(samples int64) int {
	if samples <= 0 {
		return 0
	}
	return int(samples) * h.wavelengthSampleCount()
}

func normalizeSpectralSamples(samples []rendercamera.SpectralSample) {
	if len(samples) == 0 {
		return
	}

	scale := 1.0 / float64(len(samples))
	for i := range samples {
		samples[i].Value *= scale
	}
}
