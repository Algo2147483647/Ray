package bxdf

import (
	"math"

	"github.com/Algo2147483647/ray/engine/go/internal/material/core"
)

type SpecularReflection struct {
	Reflectance core.Spectrum
}

func NewSpecularReflection(reflectance core.Spectrum) SpecularReflection {
	return SpecularReflection{Reflectance: reflectance}
}

func (s SpecularReflection) Eval(core.ShadingContext, core.Direction, core.Direction) core.Spectrum {
	return core.Spectrum{}
}

func (s SpecularReflection) Sample(_ core.ShadingContext, wo core.Direction, _ core.Sample2D) core.BxDFSample {
	if !core.IsUpperHemisphere(wo) {
		return core.BxDFSample{}
	}

	wi := reflectLocal(wo)
	cos := core.AbsCosTheta(wi)
	return core.BxDFSample{
		Wi:    wi,
		F:     s.Reflectance.DivScalar(cos),
		PDF:   1,
		Flags: core.DeltaReflection,
	}
}

func (s SpecularReflection) PDF(core.ShadingContext, core.Direction, core.Direction) float64 {
	return 0
}

func (s SpecularReflection) AlbedoBound(core.ShadingContext) core.Spectrum {
	return s.Reflectance
}

func (s SpecularReflection) RoughnessInfo(core.ShadingContext) core.RoughnessInfo {
	return core.RoughnessInfo{IsDelta: true}
}

func (s SpecularReflection) DeltaFlags() core.DeltaFlags {
	return core.DeltaReflection
}

type SpecularDielectric struct {
	Reflectance   core.Spectrum
	Transmittance core.Spectrum
	EtaOutside    float64
	EtaInside     float64
}

func NewSpecularDielectric(reflectance, transmittance core.Spectrum, etaOutside, etaInside float64) SpecularDielectric {
	return SpecularDielectric{
		Reflectance:   reflectance,
		Transmittance: transmittance,
		EtaOutside:    etaOutside,
		EtaInside:     etaInside,
	}
}

func (s SpecularDielectric) Eval(core.ShadingContext, core.Direction, core.Direction) core.Spectrum {
	return core.Spectrum{}
}

func (s SpecularDielectric) Sample(ctx core.ShadingContext, wo core.Direction, u core.Sample2D) core.BxDFSample {
	if wo.Z == 0 {
		return core.BxDFSample{}
	}

	etaI := s.EtaOutside
	etaT := s.EtaInside
	if ctx.CurrentIOR > 0 {
		etaI = ctx.CurrentIOR
		if almostEqualIOR(ctx.CurrentIOR, s.EtaInside) {
			etaT = s.EtaOutside
		} else {
			etaT = s.EtaInside
		}
	}

	fresnel := FresnelDielectric(math.Abs(wo.Z), etaI, etaT)
	if u.U < fresnel {
		wi := reflectLocal(wo)
		cos := core.AbsCosTheta(wi)
		return core.BxDFSample{
			Wi:    wi,
			F:     s.Reflectance.MulScalar(fresnel).DivScalar(cos),
			PDF:   fresnel,
			Flags: core.DeltaReflection,
			Eta:   etaI,
		}
	}

	eta := etaI / etaT
	wi, ok := refractLocal(wo, eta)
	if !ok {
		wi := reflectLocal(wo)
		cos := core.AbsCosTheta(wi)
		return core.BxDFSample{
			Wi:    wi,
			F:     s.Reflectance.DivScalar(cos),
			PDF:   1,
			Flags: core.DeltaReflection,
			Eta:   etaI,
		}
	}

	cos := core.AbsCosTheta(wi)
	return core.BxDFSample{
		Wi:    wi,
		F:     s.Transmittance.MulScalar(1 - fresnel).DivScalar(cos),
		PDF:   1 - fresnel,
		Flags: core.DeltaTransmission,
		Eta:   etaT,
	}
}

func (s SpecularDielectric) PDF(core.ShadingContext, core.Direction, core.Direction) float64 {
	return 0
}

func (s SpecularDielectric) AlbedoBound(core.ShadingContext) core.Spectrum {
	return core.NewSpectrum(
		math.Max(s.Reflectance.R, s.Transmittance.R),
		math.Max(s.Reflectance.G, s.Transmittance.G),
		math.Max(s.Reflectance.B, s.Transmittance.B),
	)
}

func (s SpecularDielectric) RoughnessInfo(core.ShadingContext) core.RoughnessInfo {
	return core.RoughnessInfo{IsDelta: true}
}

func (s SpecularDielectric) DeltaFlags() core.DeltaFlags {
	return core.DeltaReflection | core.DeltaTransmission
}

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

func reflectLocal(wo core.Direction) core.Direction {
	return core.NewDirection(-wo.X, -wo.Y, wo.Z)
}

func refractLocal(wo core.Direction, eta float64) (core.Direction, bool) {
	cosThetaO := wo.Z
	sin2ThetaO := math.Max(0, 1-cosThetaO*cosThetaO)
	sin2ThetaI := eta * eta * sin2ThetaO
	if sin2ThetaI >= 1 {
		return core.Direction{}, false
	}

	cosThetaI := math.Sqrt(math.Max(0, 1-sin2ThetaI))
	if cosThetaO > 0 {
		cosThetaI = -cosThetaI
	}

	return core.NewDirection(-eta*wo.X, -eta*wo.Y, cosThetaI).Normalize(), true
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

func almostEqualIOR(a, b float64) bool {
	return math.Abs(a-b) <= 1e-6
}
