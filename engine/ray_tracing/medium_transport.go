package ray_tracing

import (
	"math"

	renderray "github.com/Algo2147483647/ray/engine/model/optics"

	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

// SegmentTransmittance is scalar attenuation at the path's sampled wavelength.
type SegmentTransmittance struct {
	Power    float64
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
		if len(sigmaA.Samples) != 1 {
			return SegmentTransmittance{}
		}
		if sigmaA.Sample(0) == 0 {
			return SegmentTransmittance{Identity: true}
		}
		return SegmentTransmittance{Power: math.Exp(-sigmaA.Sample(0) * distance)}
	}
	if sigmaA.RGBChannel(0) == 0 && sigmaA.RGBChannel(1) == 0 && sigmaA.RGBChannel(2) == 0 {
		return SegmentTransmittance{Identity: true}
	}
	if wavelength := ctx.SpectralWavelengthNM(); wavelength > 0 {
		coefficient := rgbCoefficientAtWavelength(sigmaA, wavelength)
		return SegmentTransmittance{Power: math.Exp(-coefficient * distance)}
	}
	return SegmentTransmittance{}
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

func (t SegmentTransmittance) ApplyToPower(value float64) float64 {
	if t.Identity {
		return value
	}
	return value * t.Power
}

func (t SegmentTransmittance) ApplyToRay(ray *renderray.Ray) {
	if ray == nil || t.Identity {
		return
	}
	ray.Path.Throughput *= t.Power
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
