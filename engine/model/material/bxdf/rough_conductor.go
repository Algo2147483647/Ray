package bxdf

import (
	"math"

	"github.com/Algo2147483647/ray/engine/model/material/core"
	"github.com/Algo2147483647/ray/engine/model/material/microfacet"
)

type RoughConductor struct {
	Eta    core.SpectralParameter
	K      core.SpectralParameter
	Alpha  float64
	Weight core.SpectralParameter
}

func NewRoughConductor(eta, k core.Spectrum, alpha float64) RoughConductor {
	return NewRoughConductorParameter(core.NewRGBParameter(eta), core.NewRGBParameter(k), alpha)
}

func NewRoughConductorParameter(eta, k core.SpectralParameter, alpha float64) RoughConductor {
	return RoughConductor{
		Eta:    eta,
		K:      k,
		Alpha:  microfacet.ClampAlpha(alpha),
		Weight: core.NewConstantParameter(1),
	}
}

func (r RoughConductor) Eval(ctx core.ShadingContext, wi, wo core.Direction) core.Spectrum {
	if !core.IsUpperHemisphere(wi) || !core.IsUpperHemisphere(wo) {
		return core.Spectrum{}
	}

	wh := wi.Add(wo).Normalize()
	if wh.Z <= 0 || wh.Length() == 0 {
		return core.Spectrum{}
	}

	distribution := microfacet.NewGGX(r.Alpha)
	cosI := core.AbsCosTheta(wi)
	cosO := core.AbsCosTheta(wo)
	if cosI == 0 || cosO == 0 {
		return core.Spectrum{}
	}

	f := microfacet.FresnelConductor(math.Abs(wi.Dot(wh)), r.Eta.Eval(ctx), r.K.Eval(ctx))
	scale := distribution.D(wh) * distribution.G(wi, wo) / (4 * cosI * cosO)
	return f.Mul(r.Weight.Eval(ctx)).MulScalar(scale)
}

func (r RoughConductor) Sample(ctx core.ShadingContext, wo core.Direction, u core.Sample2D) core.BxDFSample {
	if !core.IsUpperHemisphere(wo) {
		return core.BxDFSample{}
	}

	distribution := microfacet.NewGGX(r.Alpha)
	wh := distribution.SampleVisibleNormal(wo, u)
	wi := reflectAbout(wo, wh).Normalize()
	if !core.IsUpperHemisphere(wi) {
		return core.BxDFSample{}
	}

	return core.BxDFSample{
		Wi:    wi,
		F:     r.Eval(ctx, wi, wo),
		PDF:   r.PDF(ctx, wi, wo),
		Flags: core.DeltaNone,
	}
}

func (r RoughConductor) PDF(_ core.ShadingContext, wi, wo core.Direction) float64 {
	if !core.IsUpperHemisphere(wi) || !core.IsUpperHemisphere(wo) {
		return 0
	}

	wh := wi.Add(wo).Normalize()
	if wh.Z <= 0 || wh.Length() == 0 {
		return 0
	}

	dot := math.Abs(wo.Dot(wh))
	if dot == 0 {
		return 0
	}

	distribution := microfacet.NewGGX(r.Alpha)
	return distribution.PDFVisibleNormal(wo, wh) / (4 * dot)
}

func (r RoughConductor) AlbedoBound(core.ShadingContext) core.Spectrum {
	weight := r.Weight.Bounds().Max
	return core.NewSpectrum(
		math.Min(1, weight.RGBChannel(0)),
		math.Min(1, weight.RGBChannel(1)),
		math.Min(1, weight.RGBChannel(2)),
	)
}

func (r RoughConductor) RoughnessInfo(core.ShadingContext) core.RoughnessInfo {
	return core.RoughnessInfo{
		IsDelta: false,
		AlphaX:  r.Alpha,
		AlphaY:  r.Alpha,
	}
}

func (r RoughConductor) DeltaFlags() core.DeltaFlags {
	return core.DeltaNone
}

func reflectAbout(wo, wh core.Direction) core.Direction {
	return wh.MulScalar(2 * wo.Dot(wh)).Add(wo.MulScalar(-1))
}
