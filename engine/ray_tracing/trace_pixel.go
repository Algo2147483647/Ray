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
	film *rendercamera.Film,
	samples int64,
	pixel int,
	index ...int,
) {
	color := optics.Color3{}
	ray := h.RayPool.Get().(*optics.Ray)
	defer h.RayPool.Put(ray)

	switch h.SpectrumMode {
	case optics.SpectrumModeSampledWavelengths, optics.SpectrumModeHeroWavelength:
		for _, sample := range h.TraceSpectral(renderCamera, objTree, samples, index...) {
			film.RecordSpectralSample(pixel, sample.WavelengthNM, sample.Value)
		}
		return

	case optics.SpectrumModeRGB:
		for s := int64(0); s < samples; s++ {
			color = color.Add(h.TraceRGB(renderCamera, objTree, ray, index...))
		}

		color = color.MulScalar(1.0 / float64(samples))

		for ch := 0; ch < 3; ch++ {
			film.Data[ch].Data[pixel] = color[ch]
		}

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
	ray := h.RayPool.Get().(*optics.Ray)
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

				wavelengthBatch = append(wavelengthBatch, h.TraceSpectralSample(
					renderCamera,
					objTree,
					ray,
					wavelengthSampler,
					u,
					index...,
				))
			}

			spectralSamples = append(spectralSamples, wavelengthBatch...)
		}

	case optics.SpectrumModeHeroWavelength:
		for s := int64(0); s < samples; s++ {
			spectralSamples = append(spectralSamples, h.TraceSpectralSample(
				renderCamera,
				objTree,
				ray,
				wavelengthSampler,
				rand.Float64(),
				index...,
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
	renderCamera.GenerateRay(ray, index...)

	sample := wavelengthSampler.Sample(u)
	ray.SetSpectralSample(sample)

	h.TraceRay(objTree, ray, 0)

	value := optics.SpectralSampleRadiance(
		optics.SpectralRayToScalar(ray),
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
