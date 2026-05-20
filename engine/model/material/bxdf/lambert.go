package bxdf

import (
	"github.com/Algo2147483647/ray/engine/model/material/core/spectrum_parameter"
	"math"

	"github.com/Algo2147483647/ray/engine/model/material/core"
)

type Lambert struct {
	Albedo core.SpectralParameter
}

func NewLambert(albedo core.Spectrum) Lambert {
	return NewLambertParameter(spectrum_parameter.NewRGBParameter(albedo))
}

func NewLambertParameter(albedo core.SpectralParameter) Lambert {
	return Lambert{Albedo: albedo}
}

func (l Lambert) Eval(ctx core.ShadingContext, wi, wo core.Direction) core.Spectrum {
	if !core.IsUpperHemisphere(wi) || !core.IsUpperHemisphere(wo) {
		return core.Spectrum{}
	}
	return l.Albedo.Eval(ctx).MulScalar(1 / math.Pi)
}

func (l Lambert) Sample(ctx core.ShadingContext, wo core.Direction, u core.Sample2D) core.BxDFSample {
	if !core.IsUpperHemisphere(wo) {
		return core.BxDFSample{}
	}

	wi := core.CosineSampleHemisphere(u)
	return core.BxDFSample{
		Wi:    wi,
		F:     l.Eval(ctx, wi, wo),
		PDF:   l.PDF(ctx, wi, wo),
		Flags: core.DeltaNone,
	}
}

func (l Lambert) PDF(_ core.ShadingContext, wi, wo core.Direction) float64 {
	if !core.IsUpperHemisphere(wi) || !core.IsUpperHemisphere(wo) {
		return 0
	}
	return core.CosineHemispherePDF(wi)
}

func (l Lambert) AlbedoBound(core.ShadingContext) core.Spectrum {
	return l.Albedo.Bounds().Max
}

func (l Lambert) RoughnessInfo(core.ShadingContext) core.RoughnessInfo {
	return core.RoughnessInfo{
		IsDelta: false,
		AlphaX:  1,
		AlphaY:  1,
	}
}

func (l Lambert) DeltaFlags() core.DeltaFlags {
	return core.DeltaNone
}
