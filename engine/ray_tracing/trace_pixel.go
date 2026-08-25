package ray_tracing

import (
	"math/rand/v2"

	renderfilm "github.com/Algo2147483647/ray/engine/model/film"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

type pixelKernel interface {
	sampleSpectral(*Handler, *RenderJob, *optics.Ray, optics.WavelengthSample, ...int) renderfilm.SpectralSample
}

type pathTracingKernel struct{}

func (pathTracingKernel) sampleSpectral(
	h *Handler,
	job *RenderJob,
	ray *optics.Ray,
	wavelength optics.WavelengthSample,
	index ...int,
) renderfilm.SpectralSample {
	job.camera.GenerateRay(ray, job.film.Shape, index...)
	ray.SetSpectralSample(wavelength)
	h.TraceRay(job.objectTree, ray, 0)
	return renderfilm.SpectralSample{
		WavelengthNM: wavelength.LambdaNM,
		Value: optics.SpectralSampleRadiance(
			optics.SpectralRayToScalar(ray),
			ray.Path.Wavelength.PDF,
		),
	}
}

func (h *Handler) tracePixel(
	kernel pixelKernel,
	job *RenderJob,
	pixel int,
	index ...int,
) {
	for _, sample := range h.traceSpectral(
		kernel,
		job,
		index...,
	) {
		job.accumulator.AddSpectral(pixel, sample.WavelengthNM, sample.Value)
	}
}

func (h *Handler) TraceSpectral(job RenderJob, index ...int) []renderfilm.SpectralSample {
	return h.traceSpectral(pathTracingKernel{}, &job, index...)
}

func (h *Handler) traceSpectral(
	kernel pixelKernel,
	job *RenderJob,
	index ...int,
) []renderfilm.SpectralSample {
	ray := h.RayPool.Get().(*optics.Ray)
	ray.Space = h.Space
	defer h.RayPool.Put(ray)

	wavelengthSampler := h.wavelengthSampler()
	spectralSamples := make([]renderfilm.SpectralSample, 0, estimatedSpectralSampleCount(job.samples, job.wavelengthSamples))

	wavelengthSamples := job.wavelengthSamples
	for s := int64(0); s < job.samples; s++ {
		for wavelengthIndex := 0; wavelengthIndex < wavelengthSamples; wavelengthIndex++ {
			u := (float64(wavelengthIndex) + rand.Float64()) / float64(wavelengthSamples)
			spectralSamples = append(spectralSamples, kernel.sampleSpectral(
				h, job, ray, wavelengthSampler.Sample(u), index...,
			))
		}
	}

	normalizeSpectralSamples(spectralSamples)
	return spectralSamples
}

func (h *Handler) TraceSpectralSample(
	job RenderJob,
	ray *optics.Ray,
	wavelengthSampler optics.WavelengthSampler,
	u float64,
	index ...int,
) renderfilm.SpectralSample {
	return pathTracingKernel{}.sampleSpectral(
		h, &job, ray, wavelengthSampler.Sample(u), index...,
	)
}

func estimatedSpectralSampleCount(samples int64, wavelengthSamples int) int {
	if samples <= 0 {
		return 0
	}
	return int(samples) * wavelengthSamples
}

func normalizeSpectralSamples(samples []renderfilm.SpectralSample) {
	if len(samples) == 0 {
		return
	}

	scale := 1.0 / float64(len(samples))
	for i := range samples {
		samples[i].Value *= scale
	}
}
