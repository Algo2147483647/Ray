package ray_tracing

import (
	"math/rand/v2"

	rendercamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

const defaultWavelengthSamples = 4

type pixelKernel interface {
	sampleRGB(*Handler, rendercamera.Camera, *object.ObjectTree, *optics.Ray, ...int) optics.Color3
	sampleSpectral(*Handler, rendercamera.Camera, *object.ObjectTree, *optics.Ray, optics.WavelengthSample, ...int) rendercamera.SpectralSample
}

type pathTracingKernel struct{}

func (pathTracingKernel) sampleRGB(
	h *Handler,
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	index ...int,
) optics.Color3 {
	return h.TraceRGB(renderCamera, objTree, ray, index...)
}

func (pathTracingKernel) sampleSpectral(
	h *Handler,
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	wavelength optics.WavelengthSample,
	index ...int,
) rendercamera.SpectralSample {
	renderCamera.GenerateRay(ray, index...)
	ray.SetSpectralSample(wavelength)
	h.TraceRay(objTree, ray, 0)
	return rendercamera.SpectralSample{
		WavelengthNM: wavelength.LambdaNM,
		Value: optics.SpectralSampleRadiance(
			optics.SpectralRayToScalar(ray),
			ray.WavelengthPDF,
		),
	}
}

func (h *Handler) tracePixel(
	kernel pixelKernel,
	session *RenderSession,
	pixel int,
	index ...int,
) {
	color := optics.Color3{}
	ray := h.RayPool.Get().(*optics.Ray)
	ray.Geometry = h.SceneGeometry
	defer h.RayPool.Put(ray)

	switch h.SpectrumMode {
	case optics.SpectrumModeSampledWavelengths, optics.SpectrumModeHeroWavelength:
		for _, sample := range h.traceSpectral(
			kernel,
			session.Context.Camera,
			session.Context.ObjectTree,
			session.Context.Samples,
			index...,
		) {
			session.Accumulator.AddSpectral(pixel, sample.WavelengthNM, sample.Value)
		}
		return

	case optics.SpectrumModeRGB:
		for s := int64(0); s < session.Context.Samples; s++ {
			color = color.Add(kernel.sampleRGB(
				h,
				session.Context.Camera,
				session.Context.ObjectTree,
				ray,
				index...,
			))
		}

		color = color.MulScalar(1.0 / float64(session.Context.Samples))
		session.Accumulator.SetRGB(pixel, color)

	default:
	}

	return
}

func (h *Handler) TraceRGB(
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	index ...int,
) optics.Color3 {
	renderCamera.GenerateRay(ray, index...)
	ray.DisableSpectralSampling()

	h.TraceRay(objTree, ray, 0)

	r, g, b := rendercamera.LinearSRGBToFilmColorSpace(
		ray.Color[0],
		ray.Color[1],
		ray.Color[2],
		h.FilmColorSpace,
	)

	return optics.Color3{r, g, b}
}

func (h *Handler) TraceSpectral(
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	samples int64,
	index ...int,
) []rendercamera.SpectralSample {
	return h.traceSpectral(pathTracingKernel{}, renderCamera, objTree, samples, index...)
}

func (h *Handler) traceSpectral(
	kernel pixelKernel,
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	samples int64,
	index ...int,
) []rendercamera.SpectralSample {
	ray := h.RayPool.Get().(*optics.Ray)
	ray.Geometry = h.SceneGeometry
	defer h.RayPool.Put(ray)

	wavelengthSampler := h.wavelengthSampler()
	spectralSamples := make([]rendercamera.SpectralSample, 0, h.estimatedSpectralSampleCount(samples))

	switch h.SpectrumMode {
	case optics.SpectrumModeSampledWavelengths:
		wavelengthSamples := h.wavelengthSampleCount()

		for s := int64(0); s < samples; s++ {
			wavelengthBatch := make([]rendercamera.SpectralSample, 0, wavelengthSamples)

			for w := 0; w < wavelengthSamples; w++ {
				u := (float64(w) + rand.Float64()) / float64(wavelengthSamples)

				wavelengthBatch = append(wavelengthBatch, kernel.sampleSpectral(
					h, renderCamera, objTree, ray, wavelengthSampler.Sample(u), index...,
				))
			}

			spectralSamples = append(spectralSamples, wavelengthBatch...)
		}

	case optics.SpectrumModeHeroWavelength:
		for s := int64(0); s < samples; s++ {
			spectralSamples = append(spectralSamples, kernel.sampleSpectral(
				h, renderCamera, objTree, ray, wavelengthSampler.Sample(rand.Float64()), index...,
			))
		}

	default:
	}

	normalizeSpectralSamples(spectralSamples)
	return spectralSamples
}

func (h *Handler) TraceSpectralSample(
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	wavelengthSampler optics.WavelengthSampler,
	u float64,
	index ...int,
) rendercamera.SpectralSample {
	return pathTracingKernel{}.sampleSpectral(
		h, renderCamera, objTree, ray, wavelengthSampler.Sample(u), index...,
	)
}

func (h *Handler) wavelengthSampleCount() int {
	if h.WavelengthSamples > 0 {
		return h.WavelengthSamples
	}

	return defaultWavelengthSamples
}

func (h *Handler) estimatedSpectralSampleCount(samples int64) int {
	if samples <= 0 {
		return 0
	} else if h.SpectrumMode == optics.SpectrumModeSampledWavelengths {
		return int(samples) * h.wavelengthSampleCount()
	}
	return int(samples)
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
