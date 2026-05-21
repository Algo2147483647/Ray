package bxdf

import (
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
	"github.com/Algo2147483647/ray/engine/utils/maths"
	"math"

	"github.com/Algo2147483647/ray/engine/model/material/microfacet"
)

type RoughConductor struct {
	Eta    optics.SpectralParameter
	K      optics.SpectralParameter
	Alpha  float64
	Weight optics.SpectralParameter
}

func NewRoughConductor(eta, k optics.Spectrum, alpha float64) RoughConductor {
	return NewRoughConductorParameter(spectrum_parameter.NewRGBParameter(eta), spectrum_parameter.NewRGBParameter(k), alpha)
}

func NewRoughConductorParameter(eta, k optics.SpectralParameter, alpha float64) RoughConductor {
	return RoughConductor{
		Eta:    eta,
		K:      k,
		Alpha:  microfacet.ClampAlpha(alpha),
		Weight: spectrum_parameter.NewConstantParameter(1),
	}
}

func (r RoughConductor) Eval(ctx ShadingContext, wi, wo maths.Direction) optics.Spectrum {
	if !maths.IsUpperHemisphere(wi) || !maths.IsUpperHemisphere(wo) {
		return optics.Spectrum{}
	}

	wh := wi.Add(wo).Normalize()
	if wh.Z <= 0 || wh.Length() == 0 {
		return optics.Spectrum{}
	}

	distribution := microfacet.NewGGX(r.Alpha)
	cosI := maths.AbsCosTheta(wi)
	cosO := maths.AbsCosTheta(wo)
	if cosI == 0 || cosO == 0 {
		return optics.Spectrum{}
	}

	f := microfacet.FresnelConductor(math.Abs(wi.Dot(wh)), r.Eta.Eval(ctx), r.K.Eval(ctx))
	scale := distribution.D(wh) * distribution.G(wi, wo) / (4 * cosI * cosO)
	weight := compatibleWeightSpectrum(r.Weight.Eval(ctx), f, ctx)
	return f.Mul(weight).MulScalar(scale)
}

func (r RoughConductor) Sample(ctx ShadingContext, wo maths.Direction, u maths.Sample2D) BxDFSample {
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

func (r RoughConductor) PDF(_ ShadingContext, wi, wo maths.Direction) float64 {
	if !maths.IsUpperHemisphere(wi) || !maths.IsUpperHemisphere(wo) {
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

func (r RoughConductor) AlbedoBound(ShadingContext) optics.Spectrum {
	weight := r.Weight.Bounds().Max
	return optics.NewSpectrum(
		math.Min(1, weight.RGBChannel(0)),
		math.Min(1, weight.RGBChannel(1)),
		math.Min(1, weight.RGBChannel(2)),
	)
}

func (r RoughConductor) RoughnessInfo(ShadingContext) RoughnessInfo {
	return RoughnessInfo{
		IsDelta: false,
		AlphaX:  r.Alpha,
		AlphaY:  r.Alpha,
	}
}

func (r RoughConductor) DeltaFlags() DeltaFlags {
	return DeltaNone
}

func reflectAbout(wo, wh maths.Direction) maths.Direction {
	return wh.MulScalar(2 * wo.Dot(wh)).Add(wo.MulScalar(-1))
}

func compatibleWeightSpectrum(weight, target optics.Spectrum, ctx ShadingContext) optics.Spectrum {
	if weight.HasSamples() == target.HasSamples() {
		return weight
	}
	if target.HasSamples() && !weight.HasSamples() {
		return weight.UpliftRGBReflectanceToSampled(ctx.WavelengthsNM)
	}
	if !target.HasSamples() && sampledSpectrumIsConstant(weight) {
		return optics.ConstantSpectrum(weight.Sample(0))
	}
	return optics.Spectrum{}
}

func sampledSpectrumIsConstant(s optics.Spectrum) bool {
	if !s.HasSamples() {
		return false
	}
	first := s.Sample(0)
	for i := 1; i < s.SampleCount(); i++ {
		if math.Abs(s.Sample(i)-first) > 1e-12 {
			return false
		}
	}
	return true
}
