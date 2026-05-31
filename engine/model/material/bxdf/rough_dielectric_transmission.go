package bxdf

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/material/microfacet"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
	"math"
)

type RoughDielectricTransmission struct {
	Transmittance optics.SpectralParameter
	EtaOutside    float64
	InsideIOR     medium.Model
	Alpha         float64
}

func NewRoughDielectricTransmission(
	transmittance optics.Spectrum,
	etaOutside float64,
	etaInside float64,
	alpha float64,
) RoughDielectricTransmission {
	return NewRoughDielectricTransmissionParameter(
		spectrum_parameter.NewRGBParameter(transmittance),
		etaOutside,
		medium.NewConstant(etaInside),
		alpha,
	)
}

func NewRoughDielectricTransmissionParameter(
	transmittance optics.SpectralParameter,
	etaOutside float64,
	insideIOR medium.Model,
	alpha float64,
) RoughDielectricTransmission {
	if transmittance == nil {
		transmittance = spectrum_parameter.NewConstantParameter(1)
	}
	if insideIOR == nil {
		insideIOR = medium.NewConstant(1.5)
	}

	return RoughDielectricTransmission{
		Transmittance: transmittance,
		EtaOutside:    etaOutside,
		InsideIOR:     insideIOR,
		Alpha:         microfacet.ClampAlpha(alpha),
	}
}

func (r RoughDielectricTransmission) Eval(ctx ShadingContext, wi, wo maths.Direction) optics.Spectrum {
	if maths.CosTheta(wi) == 0 || maths.CosTheta(wo) == 0 {
		return optics.Spectrum{}
	}

	// Transmission must cross hemispheres.
	if maths.SameHemisphere(wi, wo) {
		return optics.Spectrum{}
	}

	insideIOR := r.insideIOR()
	wavelengthNM, _ := r.resolveWavelength(ctx)
	etaInside := insideIOR.Evaluate(wavelengthNM)

	if !medium.IsValidEta(r.EtaOutside) || !medium.IsValidEta(etaInside) {
		return optics.Spectrum{}
	}

	etaI, etaT := r.resolveOrientedEta(ctx, etaInside, wo)

	// Internally orient wo to the upper hemisphere so the GGX distribution can
	// be evaluated with an upper-hemisphere visible normal.
	owi, owo := orientTransmissionPair(wi, wo)

	cosI := maths.AbsCosTheta(owi)
	cosO := maths.AbsCosTheta(owo)
	if cosI == 0 || cosO == 0 {
		return optics.Spectrum{}
	}

	// Walter et al. rough refraction half-vector:
	//
	//   wh = normalize(wo + eta * wi)
	//
	// where eta = etaT / etaI for the PDF/evaluation half-vector.
	eta := etaT / etaI

	sum := owo.Add(owi.MulScalar(eta))
	if sum.Length() == 0 {
		return optics.Spectrum{}
	}

	wh := sum.Normalize()
	if maths.CosTheta(wh) < 0 {
		wh = wh.MulScalar(-1)
	}

	dotO := owo.Dot(wh)
	dotI := owi.Dot(wh)

	// For transmission, wo and wi should lie on opposite sides of the microfacet.
	if dotO == 0 || dotI == 0 || dotO*dotI >= 0 {
		return optics.Spectrum{}
	}

	distribution := microfacet.NewGGX(r.Alpha)

	fresnel := FresnelDielectric(math.Abs(dotO), etaI, etaT)
	if fresnel >= 1 {
		return optics.Spectrum{}
	}

	sqrtDenom := dotO + eta*dotI
	if sqrtDenom == 0 {
		return optics.Spectrum{}
	}

	// Radiance transport needs the eta^2 correction. This matches the usual
	// adjoint BSDF convention used by PBRT-style path tracers.
	factor := 1.0
	if ctx.TransportMode == TransportRadiance {
		factor = etaI / etaT
	}

	scale := math.Abs(
		distribution.D(wh) *
			distribution.G(owo, owi) *
			eta * eta *
			math.Abs(dotI) *
			math.Abs(dotO) *
			factor * factor /
			(cosI * cosO * sqrtDenom * sqrtDenom),
	)

	return r.Transmittance.Eval(ctx).MulScalar((1 - fresnel) * scale)
}

func (r RoughDielectricTransmission) Sample(ctx ShadingContext, wo maths.Direction, u maths.Sample2D) BxDFSample {
	if maths.CosTheta(wo) == 0 {
		return BxDFSample{}
	}

	insideIOR := r.insideIOR()
	wavelengthNM, spectralSample := r.resolveWavelength(ctx)
	spectralSample = spectralSample && insideIOR.IsDispersive()
	etaInside := insideIOR.Evaluate(wavelengthNM)

	if !medium.IsValidEta(r.EtaOutside) || !medium.IsValidEta(etaInside) {
		return BxDFSample{}
	}

	sampleEtaI, sampleEtaT := r.resolveEta(ctx, etaInside, wo)
	etaI, etaT := orientEtaPair(sampleEtaI, sampleEtaT, wo)

	owo, flipped := orientOutgoingToUpper(wo)

	distribution := microfacet.NewGGX(r.Alpha)
	wh := distribution.SampleVisibleNormal(owo, u)

	if maths.CosTheta(wh) <= 0 || wh.Length() == 0 {
		return BxDFSample{}
	}

	eta := etaI / etaT

	owi, ok := refractAbout(owo, wh, eta)
	if !ok {
		return BxDFSample{}
	}

	// Transmission must go to the opposite hemisphere.
	if maths.SameHemisphere(owi, owo) {
		return BxDFSample{}
	}

	wi := owi
	if flipped {
		wi = wi.MulScalar(-1)
	}

	pdf := r.PDF(ctx, wi, wo)
	if pdf == 0 {
		return BxDFSample{}
	}

	f := r.Eval(ctx, wi, wo)

	sample := BxDFSample{
		Wi:             wi,
		F:              f,
		PDF:            pdf,
		Flags:          TransmissionEvent,
		Eta:            sampleEtaT,
		TransmitMedium: ctx.TransmitMedium,
	}

	if spectralSample {
		sample.WavelengthNM = wavelengthNM
	}

	return sample
}

func (r RoughDielectricTransmission) PDF(ctx ShadingContext, wi, wo maths.Direction) float64 {
	if maths.CosTheta(wi) == 0 || maths.CosTheta(wo) == 0 {
		return 0
	}

	if maths.SameHemisphere(wi, wo) {
		return 0
	}

	insideIOR := r.insideIOR()
	wavelengthNM, _ := r.resolveWavelength(ctx)
	etaInside := insideIOR.Evaluate(wavelengthNM)

	if !medium.IsValidEta(r.EtaOutside) || !medium.IsValidEta(etaInside) {
		return 0
	}

	etaI, etaT := r.resolveOrientedEta(ctx, etaInside, wo)

	owi, owo := orientTransmissionPair(wi, wo)

	eta := etaT / etaI

	sum := owo.Add(owi.MulScalar(eta))
	if sum.Length() == 0 {
		return 0
	}

	wh := sum.Normalize()
	if maths.CosTheta(wh) < 0 {
		wh = wh.MulScalar(-1)
	}

	dotO := owo.Dot(wh)
	dotI := owi.Dot(wh)

	if dotO == 0 || dotI == 0 || dotO*dotI >= 0 {
		return 0
	}

	sqrtDenom := dotO + eta*dotI
	if sqrtDenom == 0 {
		return 0
	}

	dwhDwi := math.Abs((eta * eta * dotI) / (sqrtDenom * sqrtDenom))

	distribution := microfacet.NewGGX(r.Alpha)
	return distribution.PDFVisibleNormal(owo, wh) * dwhDwi
}

func (r RoughDielectricTransmission) AlbedoBound(ShadingContext) optics.Spectrum {
	t := r.Transmittance.Bounds().Max
	return optics.NewSpectrum(
		math.Min(1, t.RGBChannel(0)),
		math.Min(1, t.RGBChannel(1)),
		math.Min(1, t.RGBChannel(2)),
	)
}

func (r RoughDielectricTransmission) RoughnessInfo(ShadingContext) RoughnessInfo {
	return RoughnessInfo{
		IsDelta: false,
		AlphaX:  r.Alpha,
		AlphaY:  r.Alpha,
	}
}

func (r RoughDielectricTransmission) DeltaFlags() DeltaFlags {
	return TransmissionEvent | NonReciprocal
}

func (r RoughDielectricTransmission) resolveWavelength(ctx ShadingContext) (float64, bool) {
	if ctx.WavelengthNM > 0 {
		return ctx.WavelengthNM, true
	}
	return medium.DefaultWavelengthNM, false
}

func (r RoughDielectricTransmission) resolveEta(ctx ShadingContext, etaInside float64, wo maths.Direction) (float64, float64) {
	if medium.IsValidEta(ctx.EtaIncident) && medium.IsValidEta(ctx.EtaTransmit) {
		return ctx.EtaIncident, ctx.EtaTransmit
	}

	// Prefer the geometric direction if explicit eta information is unavailable.
	if maths.CosTheta(wo) >= 0 {
		return r.EtaOutside, etaInside
	}

	return etaInside, r.EtaOutside
}

func (r RoughDielectricTransmission) resolveOrientedEta(ctx ShadingContext, etaInside float64, wo maths.Direction) (float64, float64) {
	etaI, etaT := r.resolveEta(ctx, etaInside, wo)
	return orientEtaPair(etaI, etaT, wo)
}

func orientEtaPair(etaI, etaT float64, wo maths.Direction) (float64, float64) {
	if maths.CosTheta(wo) < 0 {
		return etaT, etaI
	}
	return etaI, etaT
}

func (r RoughDielectricTransmission) insideIOR() medium.Model {
	if r.InsideIOR == nil {
		return medium.NewConstant(1.5)
	}
	return r.InsideIOR
}

func orientOutgoingToUpper(wo maths.Direction) (maths.Direction, bool) {
	if maths.CosTheta(wo) < 0 {
		return wo.MulScalar(-1), true
	}
	return wo, false
}

func orientTransmissionPair(wi, wo maths.Direction) (maths.Direction, maths.Direction) {
	if maths.CosTheta(wo) < 0 {
		return wi.MulScalar(-1), wo.MulScalar(-1)
	}
	return wi, wo
}

func refractAbout(wo, wh maths.Direction, eta float64) (maths.Direction, bool) {
	cosThetaO := wo.Dot(wh)

	// The sampled microfacet normal should be visible from wo.
	if cosThetaO <= 0 {
		return maths.Direction{}, false
	}

	sin2ThetaO := math.Max(0, 1-cosThetaO*cosThetaO)
	sin2ThetaI := eta * eta * sin2ThetaO

	if sin2ThetaI >= 1 {
		return maths.Direction{}, false
	}

	cosThetaI := math.Sqrt(1 - sin2ThetaI)

	wi := wo.MulScalar(-eta).Add(
		wh.MulScalar(eta*cosThetaO - cosThetaI),
	)

	if wi.Length() == 0 {
		return maths.Direction{}, false
	}

	return wi.Normalize(), true
}
