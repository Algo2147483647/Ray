package ray_tracing

import (
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	rendercamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
	"math/rand/v2"
)

func (h *Handler) TracePixel(renderCamera rendercamera.Camera, objTree *object.ObjectTree, samples int64, index ...int) *mat.VecDense {
	color, _ := h.tracePixel(renderCamera, objTree, samples, nil, index...)
	return color
}

func (h *Handler) TracePixelWithSpectralDiagnostics(renderCamera rendercamera.Camera, objTree *object.ObjectTree, samples int64, index ...int) (*mat.VecDense, []rendercamera.SpectralSample) {
	diagnostics := make([]rendercamera.SpectralSample, 0)
	color, diagnostics := h.tracePixel(renderCamera, objTree, samples, diagnostics, index...)
	return color, diagnostics
}

func (h *Handler) tracePixel(renderCamera rendercamera.Camera, objTree *object.ObjectTree, samples int64, diagnostics []rendercamera.SpectralSample, index ...int) (*mat.VecDense, []rendercamera.SpectralSample) {
	color := mat.NewVecDense(3, nil)
	ray := h.RayPool.Get().(*optics.Ray) // new ray
	defer h.RayPool.Put(ray)

	wavelengthSampler := h.wavelengthSampler()
	totalTraces := int64(0)

	for s := int64(0); s < samples; s++ {
		switch h.SpectrumMode {
		case optics.SpectrumModeRGB:
			renderCamera.GenerateRay(ray, index...)
			ray.DisableSpectralSampling()
			color.AddVec(color, h.TraceRay(objTree, ray, 0))
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
				contribution := optics.SpectralRayToXYZ(h.TraceRay(objTree, ray, 0), ray)
				color.AddVec(color, contribution)
				diagnostics = appendSpectralDiagnostic(diagnostics, sample, contribution)
				totalTraces++
			}

		default:
			renderCamera.GenerateRay(ray, index...)
			sample := wavelengthSampler.Sample(rand.Float64())
			ray.SetSpectralSample(sample)
			contribution := optics.SpectralRayToXYZ(h.TraceRay(objTree, ray, 0), ray)
			color.AddVec(color, contribution)
			diagnostics = appendSpectralDiagnostic(diagnostics, sample, contribution)
			totalTraces++
		}
	}

	if totalTraces == 0 {
		return color, diagnostics
	}
	return math_lib.ScaleVec(color, 1.0/float64(totalTraces), color), diagnostics
}

func appendSpectralDiagnostic(diagnostics []rendercamera.SpectralSample, sample optics.WavelengthSample, contribution *mat.VecDense) []rendercamera.SpectralSample {
	if diagnostics == nil || contribution == nil || contribution.Len() < 2 {
		return diagnostics
	}
	return append(diagnostics, rendercamera.SpectralSample{
		WavelengthNM: sample.LambdaNM,
		Value:        contribution.AtVec(1),
	})
}
