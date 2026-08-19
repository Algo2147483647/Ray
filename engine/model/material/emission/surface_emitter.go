package emission

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

type StrengthQuantity uint8

const (
	PeakRadiance StrengthQuantity = iota
	TotalExitance
)

// SurfaceEmitter composes an authored spatial/spectral field with a normalized
// and sampleable angular distribution.
type SurfaceEmitter struct {
	Field        RadianceField
	Distribution AngularDistribution
	Quantity     StrengthQuantity
}

func NewSurfaceEmitter(field RadianceField, distribution AngularDistribution, quantity StrengthQuantity) SurfaceEmitter {
	if distribution == nil {
		distribution = NewUniform(TwoSided)
	}
	return SurfaceEmitter{Field: field, Distribution: distribution, Quantity: quantity}
}

func (e SurfaceEmitter) Eval(ctx bxdf.ShadingContext, woLocal maths.Direction) optics.Spectrum {
	if e.Field == nil {
		return optics.Spectrum{}
	}
	distribution := e.angular()
	scale := distribution.Eval(woLocal)
	if scale <= 0 {
		return optics.Spectrum{}
	}
	value := e.Field.EvaluateRadiance(ctx)
	if e.Quantity == TotalExitance {
		z := distribution.ProjectedIntegral(woLocal.Len())
		if z <= 0 {
			return optics.Spectrum{}
		}
		value = value.DivScalar(z)
	}
	return value.MulScalar(scale)
}

func (e SurfaceEmitter) SampleDirection(ctx bxdf.ShadingContext, u maths.Sample2D) DirectionSample {
	dimension := ctx.GeometricNormal.Len()
	if dimension == 0 {
		dimension = 3
	}
	sample := e.angular().Sample(u, dimension)
	if sample.PDF <= 0 {
		return DirectionSample{}
	}
	return DirectionSample{
		Wo: sample.Wo, Le: e.Eval(ctx, sample.Wo), PDF: sample.PDF, Flags: sample.Flags,
	}
}

func (e SurfaceEmitter) PDFDirection(_ bxdf.ShadingContext, woLocal maths.Direction) float64 {
	return e.angular().PDF(woLocal)
}

func (e SurfaceEmitter) ExitanceEstimate(ctx bxdf.ShadingContext) optics.Spectrum {
	if e.Field == nil {
		return optics.Spectrum{}
	}
	value := e.Field.EvaluateRadiance(ctx)
	if e.Quantity == TotalExitance {
		return value
	}
	dimension := ctx.GeometricNormal.Len()
	if dimension == 0 {
		dimension = 3
	}
	return value.MulScalar(e.angular().ProjectedIntegral(dimension))
}

func (e SurfaceEmitter) DirectionFlags() DirectionFlags { return e.angular().Flags() }

func (e SurfaceEmitter) Emit(ctx bxdf.ShadingContext, wo maths.Direction) optics.Spectrum {
	return e.Eval(ctx, wo)
}

func (e SurfaceEmitter) IsDelta() bool {
	return e.DirectionFlags()&DirectionDelta != 0
}

func (e SurfaceEmitter) angular() AngularDistribution {
	if e.Distribution == nil {
		return NewUniform(TwoSided)
	}
	return e.Distribution
}
