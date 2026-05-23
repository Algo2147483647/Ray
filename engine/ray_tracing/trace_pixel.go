package ray_tracing

import (
	"math/rand/v2"

	rendercamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

const defaultWavelengthSamples = 4

func (h *Handler) TracePixel(
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	samples int64,
	index ...int,
) optics.Color3 {
	if samples <= 0 {
		return optics.Color3{}
	}

	color := optics.Color3{}
	ray := h.RayPool.Get().(*optics.Ray)
	defer h.RayPool.Put(ray)

	switch h.SpectrumMode {
	case optics.SpectrumModeRGB:
		for s := int64(0); s < samples; s++ {
			color = color.Add(h.TraceRGB(renderCamera, objTree, ray, index...))
		}

		return color.MulScalar(1.0 / float64(samples))

	case optics.SpectrumModeHeroWavelength:
		wavelengthSampler := h.wavelengthSampler()

		for s := int64(0); s < samples; s++ {
			color = color.Add(h.TraceSpectral(
				renderCamera,
				objTree,
				ray,
				wavelengthSampler,
				rand.Float64(),
				index...,
			))
		}

		return color.MulScalar(1.0 / float64(samples))

	case optics.SpectrumModeSampledWavelengths:
		wavelengthSampler := h.wavelengthSampler()
		wavelengthSamples := h.wavelengthSampleCount()

		for s := int64(0); s < samples; s++ {
			color = color.Add(h.traceSampledWavelengthsForPixelSample(
				renderCamera,
				objTree,
				ray,
				wavelengthSampler,
				wavelengthSamples,
				index...,
			))
		}

		totalTraces := samples * int64(wavelengthSamples)
		return color.MulScalar(1.0 / float64(totalTraces))

	default:
		return optics.Color3{}
	}
}

func (h *Handler) traceSampledWavelengthsForPixelSample(
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	wavelengthSampler optics.WavelengthSampler,
	wavelengthSamples int,
	index ...int,
) optics.Color3 {
	color := optics.Color3{}

	for w := 0; w < wavelengthSamples; w++ {
		u := stratifiedWavelengthSample(w, wavelengthSamples)

		color = color.Add(h.TraceSpectral(
			renderCamera,
			objTree,
			ray,
			wavelengthSampler,
			u,
			index...,
		))
	}

	return color
}

func (h *Handler) TraceRGB(
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	index ...int,
) optics.Color3 {
	renderCamera.GenerateRay(ray, index...)
	ray.DisableSpectralSampling()

	contribution := h.TraceRay(objTree, ray, 0)

	r, g, b := rendercamera.LinearSRGBToFilmColorSpace(
		contribution[0],
		contribution[1],
		contribution[2],
		h.FilmColorSpace,
	)

	return optics.Color3{r, g, b}
}

func (h *Handler) TraceSpectral(
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	wavelengthSampler optics.WavelengthSampler,
	u float64,
	index ...int,
) optics.Color3 {
	renderCamera.GenerateRay(ray, index...)

	sample := wavelengthSampler.Sample(u)
	ray.SetSpectralSample(sample)

	traced := h.TraceRay(objTree, ray, 0)
	xyz := optics.SpectralRayToXYZ(traced, ray)

	return xyzToWorkingColor(xyz, h.FilmColorSpace)
}

func (h *Handler) TracePixelSpectralSamples(
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	samples int64,
	index ...int,
) []rendercamera.SpectralSample {
	ray := h.RayPool.Get().(*optics.Ray)
	defer h.RayPool.Put(ray)

	wavelengthSampler := h.wavelengthSampler()
	spectralSamples := make([]rendercamera.SpectralSample, 0, h.estimatedSpectralSampleCount(samples))

	switch h.SpectrumMode {
	case optics.SpectrumModeSampledWavelengths:
		wavelengthSamples := h.wavelengthSampleCount()

		for s := int64(0); s < samples; s++ {
			spectralSamples = append(
				spectralSamples,
				h.collectSampledWavelengthsForPixelSample(
					renderCamera,
					objTree,
					ray,
					wavelengthSampler,
					wavelengthSamples,
					index...,
				)...,
			)
		}

	default:
		for s := int64(0); s < samples; s++ {
			spectralSamples = append(spectralSamples, h.traceSpectralSample(
				renderCamera,
				objTree,
				ray,
				wavelengthSampler,
				rand.Float64(),
				index...,
			))
		}
	}

	normalizeSpectralSamples(spectralSamples)
	return spectralSamples
}

func (h *Handler) collectSampledWavelengthsForPixelSample(
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	wavelengthSampler optics.WavelengthSampler,
	wavelengthSamples int,
	index ...int,
) []rendercamera.SpectralSample {
	samples := make([]rendercamera.SpectralSample, 0, wavelengthSamples)

	for w := 0; w < wavelengthSamples; w++ {
		u := stratifiedWavelengthSample(w, wavelengthSamples)

		samples = append(samples, h.traceSpectralSample(
			renderCamera,
			objTree,
			ray,
			wavelengthSampler,
			u,
			index...,
		))
	}

	return samples
}

func (h *Handler) traceSpectralSample(
	renderCamera rendercamera.Camera,
	objTree *object.ObjectTree,
	ray *optics.Ray,
	wavelengthSampler optics.WavelengthSampler,
	u float64,
	index ...int,
) rendercamera.SpectralSample {
	renderCamera.GenerateRay(ray, index...)

	sample := wavelengthSampler.Sample(u)
	ray.SetSpectralSample(sample)

	traced := h.TraceRay(objTree, ray, 0)

	value := optics.SpectralSampleRadiance(
		optics.SpectralRayToScalar(traced, ray),
		ray.WavelengthPDF,
	)

	return rendercamera.SpectralSample{
		WavelengthNM: sample.LambdaNM,
		Value:        value,
	}
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
	}

	if h.SpectrumMode == optics.SpectrumModeSampledWavelengths {
		return int(samples) * h.wavelengthSampleCount()
	}

	return int(samples)
}

func stratifiedWavelengthSample(index, count int) float64 {
	return (float64(index) + rand.Float64()) / float64(count)
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

func xyzToWorkingColor(xyz optics.XYZ, space rendercamera.FilmColorSpace) optics.Color3 {
	a, b, c := rendercamera.XYZToFilmColorSpace(xyz[0], xyz[1], xyz[2], space)
	return optics.Color3{a, b, c}
}
