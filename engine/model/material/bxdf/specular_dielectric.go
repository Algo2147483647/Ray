package bxdf

import (
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
	"github.com/Algo2147483647/ray/engine/utils/maths"
	"math"
)

type SpecularDielectric struct {
	Reflectance   optics.SpectralParameter
	Transmittance optics.SpectralParameter
	EtaOutside    float64
	InsideIOR     medium.Model
}

func NewSpecularDielectric(reflectance, transmittance optics.Spectrum, etaOutside float64, insideIOR medium.Model) SpecularDielectric {
	return NewSpecularDielectricParameter(spectrum_parameter.NewRGBParameter(reflectance), spectrum_parameter.NewRGBParameter(transmittance), etaOutside, insideIOR)
}

func NewSpecularDielectricParameter(reflectance, transmittance optics.SpectralParameter, etaOutside float64, insideIOR medium.Model) SpecularDielectric {
	if insideIOR == nil {
		insideIOR = medium.NewConstant(1.5)
	}
	return SpecularDielectric{
		Reflectance:   reflectance,
		Transmittance: transmittance,
		EtaOutside:    etaOutside,
		InsideIOR:     insideIOR,
	}
}

func NewSpecularDielectricConstant(reflectance, transmittance optics.Spectrum, etaOutside, etaInside float64) SpecularDielectric {
	return NewSpecularDielectric(reflectance, transmittance, etaOutside, medium.NewConstant(etaInside))
}

func (s SpecularDielectric) Eval(ShadingContext, maths.Direction, maths.Direction) optics.Spectrum {
	return optics.Spectrum{}
}

func (s SpecularDielectric) Sample(ctx ShadingContext, wo maths.Direction, u maths.Sample2D) BxDFSample {
	if wo.Z == 0 {
		return BxDFSample{}
	}

	insideIOR := s.insideIOR()
	wavelengthNM, spectralSample := s.resolveWavelength(ctx)
	spectralSample = spectralSample && insideIOR.IsDispersive()
	etaInside := insideIOR.Evaluate(wavelengthNM)
	if !medium.IsValidEta(s.EtaOutside) || !medium.IsValidEta(etaInside) {
		return BxDFSample{}
	}

	etaI, etaT := s.resolveEta(ctx, etaInside)

	fresnel := FresnelDielectric(math.Abs(wo.Z), etaI, etaT)
	if u.U < fresnel {
		wi := reflectLocal(wo)
		cos := maths.AbsCosTheta(wi)
		sample := BxDFSample{
			Wi:    wi,
			F:     s.Reflectance.Eval(ctx).MulScalar(fresnel).DivScalar(cos),
			PDF:   fresnel,
			Flags: DeltaReflection,
			Eta:   etaI,
		}
		if spectralSample {
			sample.WavelengthNM = wavelengthNM
		}
		return sample
	}

	eta := etaI / etaT
	wi, ok := refractLocal(wo, eta)
	if !ok {
		wi := reflectLocal(wo)
		cos := maths.AbsCosTheta(wi)
		sample := BxDFSample{
			Wi:    wi,
			F:     s.Reflectance.Eval(ctx).DivScalar(cos),
			PDF:   1,
			Flags: DeltaReflection,
			Eta:   etaI,
		}
		if spectralSample {
			sample.WavelengthNM = wavelengthNM
		}
		return sample
	}

	cos := maths.AbsCosTheta(wi)
	sample := BxDFSample{
		Wi:             wi,
		F:              s.Transmittance.Eval(ctx).MulScalar(1 - fresnel).DivScalar(cos),
		PDF:            1 - fresnel,
		Flags:          DeltaTransmission,
		Eta:            etaT,
		TransmitMedium: ctx.TransmitMedium,
	}
	if spectralSample {
		sample.WavelengthNM = wavelengthNM
	}
	return sample
}

func (s SpecularDielectric) PDF(ShadingContext, maths.Direction, maths.Direction) float64 {
	return 0
}

func (s SpecularDielectric) AlbedoBound(ShadingContext) optics.Spectrum {
	reflectance := s.Reflectance.Bounds().Max
	transmittance := s.Transmittance.Bounds().Max
	return optics.NewSpectrum(
		math.Max(reflectance.RGBChannel(0), transmittance.RGBChannel(0)),
		math.Max(reflectance.RGBChannel(1), transmittance.RGBChannel(1)),
		math.Max(reflectance.RGBChannel(2), transmittance.RGBChannel(2)),
	)
}

func (s SpecularDielectric) RoughnessInfo(ShadingContext) RoughnessInfo {
	return RoughnessInfo{IsDelta: true}
}

func (s SpecularDielectric) DeltaFlags() DeltaFlags {
	return DeltaReflection | DeltaTransmission
}

func (s SpecularDielectric) resolveWavelength(ctx ShadingContext) (float64, bool) {
	if ctx.WavelengthNM > 0 {
		return ctx.WavelengthNM, true
	}
	return medium.DefaultWavelengthNM, false
}

func (s SpecularDielectric) resolveEta(ctx ShadingContext, etaInside float64) (float64, float64) {
	if medium.IsValidEta(ctx.EtaIncident) && medium.IsValidEta(ctx.EtaTransmit) {
		return ctx.EtaIncident, ctx.EtaTransmit
	}

	etaI := s.EtaOutside
	etaT := etaInside
	if ctx.CurrentIOR > 0 {
		etaI = ctx.CurrentIOR
		if almostEqualIOR(ctx.CurrentIOR, etaInside) {
			etaT = s.EtaOutside
		} else {
			etaT = etaInside
		}
	}
	return etaI, etaT
}

func (s SpecularDielectric) insideIOR() medium.Model {
	if s.InsideIOR == nil {
		return medium.NewConstant(1.5)
	}
	return s.InsideIOR
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

func reflectLocal(wo maths.Direction) maths.Direction {
	return maths.NewDirection(-wo.X, -wo.Y, wo.Z)
}

func refractLocal(wo maths.Direction, eta float64) (maths.Direction, bool) {
	cosThetaO := wo.Z
	sin2ThetaO := math.Max(0, 1-cosThetaO*cosThetaO)
	sin2ThetaI := eta * eta * sin2ThetaO
	if sin2ThetaI >= 1 {
		return maths.Direction{}, false
	}

	cosThetaI := math.Sqrt(math.Max(0, 1-sin2ThetaI))
	if cosThetaO > 0 {
		cosThetaI = -cosThetaI
	}

	return maths.NewDirection(-eta*wo.X, -eta*wo.Y, cosThetaI).Normalize(), true
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
