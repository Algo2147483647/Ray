package bxdf

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/material/microfacet"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
)

// RoughDielectricReflection models the reflected lobe of an opaque rough
// dielectric coating. It is useful for varnish, ceramic glaze, and clearcoat
// layered over a diffuse substrate.
type RoughDielectricReflection struct {
	Reflectance optics.SpectralParameter
	EtaOutside  float64
	InsideIOR   medium.Model
	Alpha       float64
}

func NewRoughDielectricReflection(
	reflectance optics.Spectrum,
	etaOutside float64,
	etaInside float64,
	alpha float64,
) RoughDielectricReflection {
	return NewRoughDielectricReflectionParameter(
		spectrum_parameter.NewRGBParameter(reflectance),
		etaOutside,
		medium.NewConstant(etaInside),
		alpha,
	)
}

func NewRoughDielectricReflectionParameter(
	reflectance optics.SpectralParameter,
	etaOutside float64,
	insideIOR medium.Model,
	alpha float64,
) RoughDielectricReflection {
	if reflectance == nil {
		reflectance = spectrum_parameter.NewConstantParameter(1)
	}
	if insideIOR == nil {
		insideIOR = medium.NewConstant(1.5)
	}
	return RoughDielectricReflection{
		Reflectance: reflectance,
		EtaOutside:  etaOutside,
		InsideIOR:   insideIOR,
		Alpha:       microfacet.ClampAlpha(alpha),
	}
}

func (r RoughDielectricReflection) Eval(ctx ShadingContext, wi, wo maths.Direction) optics.Spectrum {
	if !maths.IsUpperHemisphere(wi) || !maths.IsUpperHemisphere(wo) {
		return optics.Spectrum{}
	}

	wh := wi.Add(wo).Normalize()
	if maths.CosTheta(wh) <= 0 || wh.Length() == 0 {
		return optics.Spectrum{}
	}
	cosI := maths.AbsCosTheta(wi)
	cosO := maths.AbsCosTheta(wo)
	if cosI == 0 || cosO == 0 {
		return optics.Spectrum{}
	}

	etaInside := r.InsideIOR.Evaluate(reflectionWavelength(ctx))
	if !medium.IsValidEta(r.EtaOutside) || !medium.IsValidEta(etaInside) {
		return optics.Spectrum{}
	}
	fresnel := microfacet.FresnelDielectric(math.Abs(wi.Dot(wh)), r.EtaOutside, etaInside)
	distribution := microfacet.NewGGX(r.Alpha)
	scale := fresnel * distribution.D(wh) * distribution.G(wi, wo) / (4 * cosI * cosO)
	return r.Reflectance.Eval(ctx).MulScalar(scale)
}

func (r RoughDielectricReflection) Sample(ctx ShadingContext, wo maths.Direction, u maths.Sample2D) BxDFSample {
	if !maths.IsUpperHemisphere(wo) {
		return BxDFSample{}
	}
	distribution := microfacet.NewGGX(r.Alpha)
	wh := distribution.SampleVisibleNormal(wo, u)
	wi := reflectAbout(wo, wh).Normalize()
	if !maths.IsUpperHemisphere(wi) {
		return BxDFSample{}
	}
	return BxDFSample{
		Wi:    wi,
		F:     r.Eval(ctx, wi, wo),
		PDF:   r.PDF(ctx, wi, wo),
		Flags: DeltaNone,
	}
}

func (r RoughDielectricReflection) PDF(_ ShadingContext, wi, wo maths.Direction) float64 {
	if !maths.IsUpperHemisphere(wi) || !maths.IsUpperHemisphere(wo) {
		return 0
	}
	wh := wi.Add(wo).Normalize()
	if maths.CosTheta(wh) <= 0 || wh.Length() == 0 {
		return 0
	}
	dot := math.Abs(wo.Dot(wh))
	if dot == 0 {
		return 0
	}
	return microfacet.NewGGX(r.Alpha).PDFVisibleNormal(wo, wh) / (4 * dot)
}

func (r RoughDielectricReflection) AlbedoBound(ShadingContext) optics.Spectrum {
	bound := r.Reflectance.Bounds().Max
	return optics.NewSpectrum(
		math.Min(1, bound.RGBChannel(0)),
		math.Min(1, bound.RGBChannel(1)),
		math.Min(1, bound.RGBChannel(2)),
	)
}

func (r RoughDielectricReflection) RoughnessInfo(ShadingContext) RoughnessInfo {
	return RoughnessInfo{IsDelta: false, AlphaX: r.Alpha, AlphaY: r.Alpha}
}

func (r RoughDielectricReflection) DeltaFlags() DeltaFlags { return DeltaNone }

func reflectionWavelength(ctx ShadingContext) float64 {
	if ctx.WavelengthNM > 0 && !math.IsNaN(ctx.WavelengthNM) && !math.IsInf(ctx.WavelengthNM, 0) {
		return ctx.WavelengthNM
	}
	return medium.DefaultWavelengthNM
}
