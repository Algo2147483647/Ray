package ray_tracing

import (
	rendercamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"math/rand/v2"
)

func (h *Handler) TracePixel(renderCamera rendercamera.Camera, objTree *object.ObjectTree, samples int64, index ...int) optics.Color3 {
	color, _ := h.tracePixel(renderCamera, objTree, samples, nil, index...)
	return color
}

func (h *Handler) TracePixelWithSpectralDiagnostics(renderCamera rendercamera.Camera, objTree *object.ObjectTree, samples int64, index ...int) (optics.Color3, []rendercamera.SpectralSample) {
	diagnostics := make([]rendercamera.SpectralSample, 0)
	color, diagnostics := h.tracePixel(renderCamera, objTree, samples, diagnostics, index...)
	return color, diagnostics
}

func (h *Handler) tracePixel(renderCamera rendercamera.Camera, objTree *object.ObjectTree, samples int64, diagnostics []rendercamera.SpectralSample, index ...int) (optics.Color3, []rendercamera.SpectralSample) {
	color := optics.Color3{}
	ray := h.RayPool.Get().(*optics.Ray) // new ray
	defer h.RayPool.Put(ray)

	wavelengthSampler := h.wavelengthSampler()
	totalTraces := int64(0)

	for s := int64(0); s < samples; s++ {
		switch h.SpectrumMode {
		case optics.SpectrumModeRGB:
			renderCamera.GenerateRay(ray, index...)
			ray.DisableSpectralSampling()
			contribution := h.TraceRay(objTree, ray, 0)
			r, g, b := rendercamera.LinearSRGBToFilmColorSpace(
				contribution[0],
				contribution[1],
				contribution[2],
				h.WorkingSpace,
			)
			color = color.Add(optics.Color3{r, g, b})
			totalTraces++

		case optics.SpectrumModeSampledWavelengths:
			wavelengthSamples := h.WavelengthSamples
			if wavelengthSamples <= 0 {
				wavelengthSamples = 4
			}
			for w := 0; w < wavelengthSamples; w++ {
				renderCamera.GenerateRay(ray, index...)
				sample := wavelengthSampler.Sample((float64(w) + rand.Float64()) / float64(wavelengthSamples))
				ray.SetSpectralSample(sample)
				traced := h.TraceRay(objTree, ray, 0)
				contribution := optics.SpectralRayToXYZ(traced, ray)
				color = color.Add(xyzToWorkingColor(contribution, h.WorkingSpace))
				diagnostics = appendSpectralDiagnostic(diagnostics, sample, contribution)
				totalTraces++
			}

		default:
			renderCamera.GenerateRay(ray, index...)
			sample := wavelengthSampler.Sample(rand.Float64())
			ray.SetSpectralSample(sample)
			traced := h.TraceRay(objTree, ray, 0)
			contribution := optics.SpectralRayToXYZ(traced, ray)
			color = color.Add(xyzToWorkingColor(contribution, h.WorkingSpace))
			diagnostics = appendSpectralDiagnostic(diagnostics, sample, contribution)
			totalTraces++
		}
	}

	if totalTraces == 0 {
		return color, diagnostics
	}
	return color.MulScalar(1.0 / float64(totalTraces)), diagnostics
}

func (h *Handler) TracePixelSpectralSamples(renderCamera rendercamera.Camera, objTree *object.ObjectTree, samples int64, index ...int) []rendercamera.SpectralSample {
	if h.SpectrumMode == optics.SpectrumModeRGB {
		return nil
	}

	ray := h.RayPool.Get().(*optics.Ray)
	defer h.RayPool.Put(ray)

	wavelengthSampler := h.wavelengthSampler()
	spectralSamples := make([]rendercamera.SpectralSample, 0)
	totalTraces := int64(0)

	for s := int64(0); s < samples; s++ {
		switch h.SpectrumMode {
		case optics.SpectrumModeSampledWavelengths:
			wavelengthSamples := h.WavelengthSamples
			if wavelengthSamples <= 0 {
				wavelengthSamples = 4
			}
			for w := 0; w < wavelengthSamples; w++ {
				renderCamera.GenerateRay(ray, index...)
				sample := wavelengthSampler.Sample((float64(w) + rand.Float64()) / float64(wavelengthSamples))
				ray.SetSpectralSample(sample)
				traced := h.TraceRay(objTree, ray, 0)
				value := optics.SpectralSampleRadiance(optics.SpectralRayToScalar(traced, ray), ray.WavelengthPDF)
				spectralSamples = append(spectralSamples, rendercamera.SpectralSample{
					WavelengthNM: sample.LambdaNM,
					Value:        value,
				})
				totalTraces++
			}
		default:
			renderCamera.GenerateRay(ray, index...)
			sample := wavelengthSampler.Sample(rand.Float64())
			ray.SetSpectralSample(sample)
			traced := h.TraceRay(objTree, ray, 0)
			value := optics.SpectralSampleRadiance(optics.SpectralRayToScalar(traced, ray), ray.WavelengthPDF)
			spectralSamples = append(spectralSamples, rendercamera.SpectralSample{
				WavelengthNM: sample.LambdaNM,
				Value:        value,
			})
			totalTraces++
		}
	}

	if totalTraces == 0 {
		return spectralSamples
	}
	scale := 1 / float64(totalTraces)
	for i := range spectralSamples {
		spectralSamples[i].Value *= scale
	}
	return spectralSamples
}

func xyzToWorkingColor(xyz optics.XYZ, space rendercamera.FilmColorSpace) optics.Color3 {
	a, b, c := rendercamera.XYZToFilmColorSpace(xyz[0], xyz[1], xyz[2], space)
	return optics.Color3{a, b, c}
}

func appendSpectralDiagnostic(diagnostics []rendercamera.SpectralSample, sample optics.WavelengthSample, contribution optics.XYZ) []rendercamera.SpectralSample {
	if diagnostics == nil {
		return diagnostics
	}
	return append(diagnostics, rendercamera.SpectralSample{
		WavelengthNM: sample.LambdaNM,
		Value:        contribution[1],
	})
}
