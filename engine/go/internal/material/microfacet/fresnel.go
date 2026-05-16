package microfacet

import (
	"math"

	"github.com/Algo2147483647/ray/engine/go/internal/material/core"
)

func FresnelDielectric(cosThetaI, etaI, etaT float64) float64 {
	cosThetaI = clamp(cosThetaI, -1, 1)
	entering := cosThetaI > 0
	if !entering {
		etaI, etaT = etaT, etaI
		cosThetaI = math.Abs(cosThetaI)
	}

	sinThetaI := math.Sqrt(math.Max(0, 1-cosThetaI*cosThetaI))
	sinThetaT := etaI / etaT * sinThetaI
	if sinThetaT >= 1 {
		return 1
	}

	cosThetaT := math.Sqrt(math.Max(0, 1-sinThetaT*sinThetaT))
	rParallel := ((etaT * cosThetaI) - (etaI * cosThetaT)) / ((etaT * cosThetaI) + (etaI * cosThetaT))
	rPerpendicular := ((etaI * cosThetaI) - (etaT * cosThetaT)) / ((etaI * cosThetaI) + (etaT * cosThetaT))
	return (rParallel*rParallel + rPerpendicular*rPerpendicular) * 0.5
}

func FresnelConductor(cosThetaI float64, eta, k core.Spectrum) core.Spectrum {
	return core.NewSpectrum(
		fresnelConductorChannel(cosThetaI, eta.R, k.R),
		fresnelConductorChannel(cosThetaI, eta.G, k.G),
		fresnelConductorChannel(cosThetaI, eta.B, k.B),
	)
}

func fresnelConductorChannel(cosThetaI, eta, k float64) float64 {
	cosThetaI = math.Abs(clamp(cosThetaI, -1, 1))
	cos2 := cosThetaI * cosThetaI
	sin2 := 1 - cos2
	eta2 := eta * eta
	k2 := k * k

	t0 := eta2 - k2 - sin2
	a2plusb2 := math.Sqrt(t0*t0 + 4*eta2*k2)
	t1 := a2plusb2 + cos2
	a := math.Sqrt(0.5 * (a2plusb2 + t0))
	t2 := 2 * cosThetaI * a
	rs := (t1 - t2) / (t1 + t2)

	t3 := cos2*a2plusb2 + sin2*sin2
	t4 := t2 * sin2
	rp := rs * (t3 - t4) / (t3 + t4)

	return clamp((rp+rs)*0.5, 0, 1)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
