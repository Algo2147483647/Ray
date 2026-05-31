package bxdf

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
)

type Lambert struct {
	Albedo optics.SpectralParameter
}

func NewLambert(albedo optics.Spectrum) Lambert {
	return NewLambertParameter(spectrum_parameter.NewRGBParameter(albedo))
}

func NewLambertParameter(albedo optics.SpectralParameter) Lambert {
	return Lambert{Albedo: albedo}
}

func (l Lambert) Eval(ctx ShadingContext, wi, wo maths.Direction) optics.Spectrum {
	if !maths.IsUpperHemisphere(wi) || !maths.IsUpperHemisphere(wo) {
		return optics.Spectrum{}
	}
	return l.Albedo.Eval(ctx).MulScalar(1 / maths.CosineHemisphereIntegral(wi.Len()))
}

func (l Lambert) Sample(ctx ShadingContext, wo maths.Direction, u maths.Sample2D) BxDFSample {
	if !maths.IsUpperHemisphere(wo) {
		return BxDFSample{}
	}

	wi := maths.CosineSampleHemisphereND(u, wo.Len())
	return BxDFSample{
		Wi:    wi,
		F:     l.Eval(ctx, wi, wo),
		PDF:   l.PDF(ctx, wi, wo),
		Flags: DeltaNone,
	}
}

func (l Lambert) PDF(_ ShadingContext, wi, wo maths.Direction) float64 {
	if !maths.IsUpperHemisphere(wi) || !maths.IsUpperHemisphere(wo) {
		return 0
	}
	return maths.CosineHemispherePDF(wi)
}

func (l Lambert) AlbedoBound(ShadingContext) optics.Spectrum {
	return l.Albedo.Bounds().Max
}

func (l Lambert) RoughnessInfo(ShadingContext) RoughnessInfo {
	return RoughnessInfo{
		IsDelta: false,
		AlphaX:  1,
		AlphaY:  1,
	}
}

func (l Lambert) DeltaFlags() DeltaFlags {
	return DeltaNone
}
