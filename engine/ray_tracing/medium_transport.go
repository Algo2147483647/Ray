package ray_tracing

import (
	"math"

	renderray "github.com/Algo2147483647/ray/engine/model/optics"

	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
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
	if sigmaA == 0 {
		return SegmentTransmittance{Identity: true}
	}
	if sigmaA < 0 || math.IsNaN(sigmaA) || math.IsInf(sigmaA, 0) {
		return SegmentTransmittance{}
	}
	return SegmentTransmittance{Power: math.Exp(-sigmaA * distance)}
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
