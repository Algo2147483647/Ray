package bsdf

import "github.com/Algo2147483647/ray/engine/go/internal/material/core"

type Single struct {
	BxDF core.BxDF
}

func NewSingle(bxdf core.BxDF) Single {
	return Single{BxDF: bxdf}
}

func (s Single) Eval(ctx core.ShadingContext, wi, wo core.Direction) core.Spectrum {
	if s.BxDF == nil {
		return core.Spectrum{}
	}
	return s.BxDF.Eval(ctx, wi, wo)
}

func (s Single) Sample(ctx core.ShadingContext, wo core.Direction, u core.Sample2D) core.BxDFSample {
	if s.BxDF == nil {
		return core.BxDFSample{}
	}
	return s.BxDF.Sample(ctx, wo, u)
}

func (s Single) PDF(ctx core.ShadingContext, wi, wo core.Direction) float64 {
	if s.BxDF == nil {
		return 0
	}
	return s.BxDF.PDF(ctx, wi, wo)
}

func (s Single) AlbedoBound(ctx core.ShadingContext) core.Spectrum {
	if s.BxDF == nil {
		return core.Spectrum{}
	}
	return s.BxDF.AlbedoBound(ctx)
}

func (s Single) RoughnessInfo(ctx core.ShadingContext) core.RoughnessInfo {
	if s.BxDF == nil {
		return core.RoughnessInfo{}
	}
	return s.BxDF.RoughnessInfo(ctx)
}

func (s Single) DeltaFlags() core.DeltaFlags {
	if s.BxDF == nil {
		return core.DeltaNone
	}
	return s.BxDF.DeltaFlags()
}
