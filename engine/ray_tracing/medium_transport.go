package ray_tracing

import (
	"math"

	renderray "github.com/Algo2147483647/ray/engine/model/optics"

	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

// SegmentTransmittance is the attenuation accumulated while travelling through
// one homogeneous medium segment. Keeping it as a spectrum lets camera paths,
// light paths, and shadow connections share the same Beer-Lambert evaluation.
type SegmentTransmittance struct {
	Value    optics.Spectrum
	Identity bool
}

func evaluateSegmentTransmittance(
	media *medium.Registry,
	mediumID medium.MediumID,
	distance float64,
	ctx bxdf.ShadingContext,
) SegmentTransmittance {
	if media == nil || distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return SegmentTransmittance{Identity: true}
	}

	sigmaA := media.SigmaA(mediumID, ctx)
	if sigmaA.HasSamples() {
		allZero := true
		values := make([]float64, len(sigmaA.Samples))
		for i, coefficient := range sigmaA.Samples {
			allZero = allZero && coefficient == 0
			values[i] = math.Exp(-coefficient * distance)
		}
		if allZero {
			return SegmentTransmittance{Identity: true}
		}
		return SegmentTransmittance{Value: optics.NewSampledSpectrum(values)}
	}
	if sigmaA.RGBChannel(0) == 0 && sigmaA.RGBChannel(1) == 0 && sigmaA.RGBChannel(2) == 0 {
		return SegmentTransmittance{Identity: true}
	}
	if wavelengths := segmentWavelengths(ctx); len(wavelengths) > 0 {
		values := make([]float64, len(wavelengths))
		for i, wavelength := range wavelengths {
			coefficient := rgbCoefficientAtWavelength(sigmaA, wavelength)
			values[i] = math.Exp(-coefficient * distance)
		}
		return SegmentTransmittance{Value: optics.NewSampledSpectrum(values)}
	}

	value := optics.NewRGBSpectrum(
		math.Exp(-sigmaA.RGBChannel(0)*distance),
		math.Exp(-sigmaA.RGBChannel(1)*distance),
		math.Exp(-sigmaA.RGBChannel(2)*distance),
	)
	return SegmentTransmittance{Value: value}
}

func segmentWavelengths(ctx bxdf.ShadingContext) []float64 {
	if wavelengths := ctx.SpectralWavelengthsNM(); len(wavelengths) > 0 {
		return wavelengths
	}
	if wavelength := ctx.SpectralWavelengthNM(); wavelength > 0 {
		return []float64{wavelength}
	}
	return nil
}

// rgbCoefficientAtWavelength treats RGB sigma_a as three non-negative basis
// coefficients. Normalizing the wavelength weights preserves neutral media:
// equal RGB coefficients remain constant at every wavelength.
func rgbCoefficientAtWavelength(sigmaA medium.CoefficientSpectrum, wavelength float64) float64 {
	weights := optics.RGBWeight(wavelength)
	weightSum := weights[0] + weights[1] + weights[2]
	if weightSum <= 0 || math.IsNaN(weightSum) || math.IsInf(weightSum, 0) {
		return (sigmaA.RGBChannel(0) + sigmaA.RGBChannel(1) + sigmaA.RGBChannel(2)) / 3
	}
	return (sigmaA.RGBChannel(0)*weights[0] +
		sigmaA.RGBChannel(1)*weights[1] +
		sigmaA.RGBChannel(2)*weights[2]) / weightSum
}

func (t SegmentTransmittance) ApplyToSpectrum(value optics.Spectrum) optics.Spectrum {
	if t.Identity {
		return value
	}
	return value.Mul(t.Value)
}

func (t SegmentTransmittance) ApplyToRay(ray *renderray.Ray) {
	if ray == nil || t.Identity {
		return
	}
	applySpectrum(ray, t.Value)
}

func prepareMediumContext(ctx *bxdf.ShadingContext, media *medium.Registry, ray *renderray.Ray, boundary medium.Boundary, frontFace bool) {
	transition := ray.MediumStack.ResolveTransition(boundary, frontFace)
	incident := transition.Incident
	transmit := transition.Transmit
	entering := transition.Entering

	ctx.IncidentMedium = incident
	ctx.TransmitMedium = transmit
	ctx.Entering = entering

	if !boundary.Active() {
		return
	}
	ctx.EtaIncident = media.IOR(incident, *ctx)
	ctx.EtaTransmit = media.IOR(transmit, *ctx)
}

func applyMediumTransmission(ray *renderray.Ray, ctx bxdf.ShadingContext, boundary medium.Boundary, sample bxdf.BxDFSample) {
	if boundary.Active() && sample.TransmitMedium != medium.MediumNone {
		if !boundary.Thin {
			if ctx.Entering {
				ray.MediumStack.EnterBoundary(boundary)
			} else {
				ray.MediumStack.ExitBoundary(boundary)
			}
		}
	}
}

func applyMediumAbsorption(media *medium.Registry, ray *renderray.Ray, distance float64, ctx bxdf.ShadingContext) {
	if ray == nil {
		return
	}
	evaluateSegmentTransmittance(media, ray.MediumStack.Current(), distance, ctx).ApplyToRay(ray)
}
