package ray_tracing

import (
	"math"

	renderray "github.com/Algo2147483647/ray/engine/model/optics"

	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

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
	ctx.CurrentIOR = ctx.EtaIncident
	ray.RefractionIndex = ctx.EtaIncident
}

func applyMediumTransmission(media *medium.Registry, ray *renderray.Ray, ctx bxdf.ShadingContext, boundary medium.Boundary, sample bxdf.BxDFSample) {
	if boundary.Active() && sample.TransmitMedium != medium.MediumNone {
		if !boundary.Thin {
			if ctx.Entering {
				ray.MediumStack.EnterBoundary(boundary)
			} else {
				ray.MediumStack.ExitBoundary(boundary)
			}
		}
		ray.RefractionIndex = media.IOR(ray.MediumStack.Current(), ctx)
		return
	}

	if sample.Eta > 0 {
		ray.RefractionIndex = sample.Eta
	}
}

func applyMediumAbsorption(media *medium.Registry, ray *renderray.Ray, distance float64, ctx bxdf.ShadingContext) {
	if media == nil || ray == nil || distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return
	}
	sigmaA := media.SigmaA(ray.MediumStack.Current(), ctx)
	if sigmaA.HasSamples() {
		transmittance := math.Exp(-sigmaA.Sample(0) * distance)
		ray.SpectralPower *= transmittance
		ray.SpectralPath = true
		return
	}
	transmittance := optics.NewRGBSpectrum(
		math.Exp(-sigmaA.RGBChannel(0)*distance),
		math.Exp(-sigmaA.RGBChannel(1)*distance),
		math.Exp(-sigmaA.RGBChannel(2)*distance),
	)
	applySpectrum(ray, transmittance)
}
