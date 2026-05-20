package bsdf

import (
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils/maths"
)

type Single struct {
	BxDF bxdf.BxDF
}

func NewSingle(bxdf bxdf.BxDF) Single {
	return Single{BxDF: bxdf}
}

func (s Single) Eval(ctx bxdf.ShadingContext, wi, wo maths.Direction) optics.Spectrum {
	if s.BxDF == nil {
		return optics.Spectrum{}
	}
	return s.BxDF.Eval(ctx, wi, wo)
}

func (s Single) Sample(ctx bxdf.ShadingContext, wo maths.Direction, u maths.Sample2D) bxdf.BxDFSample {
	if s.BxDF == nil {
		return bxdf.BxDFSample{}
	}
	return s.BxDF.Sample(ctx, wo, u)
}

func (s Single) PDF(ctx bxdf.ShadingContext, wi, wo maths.Direction) float64 {
	if s.BxDF == nil {
		return 0
	}
	return s.BxDF.PDF(ctx, wi, wo)
}

func (s Single) AlbedoBound(ctx bxdf.ShadingContext) optics.Spectrum {
	if s.BxDF == nil {
		return optics.Spectrum{}
	}
	return s.BxDF.AlbedoBound(ctx)
}

func (s Single) RoughnessInfo(ctx bxdf.ShadingContext) bxdf.RoughnessInfo {
	if s.BxDF == nil {
		return bxdf.RoughnessInfo{}
	}
	return s.BxDF.RoughnessInfo(ctx)
}

func (s Single) DeltaFlags() bxdf.DeltaFlags {
	if s.BxDF == nil {
		return bxdf.DeltaNone
	}
	return s.BxDF.DeltaFlags()
}
