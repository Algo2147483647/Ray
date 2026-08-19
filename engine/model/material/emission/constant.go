package emission

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
)

type Constant struct {
	Radiance optics.SpectralParameter
}

func NewConstant(color optics.Spectrum) Constant {
	return NewConstantParameter(spectrum_parameter.NewRGBParameter(color))
}

func NewConstantParameter(radiance optics.SpectralParameter) Constant {
	return Constant{Radiance: radiance}
}

func (e Constant) EvaluateRadiance(ctx bxdf.ShadingContext) optics.Spectrum {
	return e.Radiance.Eval(ctx)
}

func (e Constant) Eval(ctx bxdf.ShadingContext, wo maths.Direction) optics.Spectrum {
	return e.defaultEmitter().Eval(ctx, wo)
}
func (e Constant) SampleDirection(ctx bxdf.ShadingContext, u maths.Sample2D) DirectionSample {
	return e.defaultEmitter().SampleDirection(ctx, u)
}
func (e Constant) PDFDirection(ctx bxdf.ShadingContext, wo maths.Direction) float64 {
	return e.defaultEmitter().PDFDirection(ctx, wo)
}
func (e Constant) ExitanceEstimate(ctx bxdf.ShadingContext) optics.Spectrum {
	return e.defaultEmitter().ExitanceEstimate(ctx)
}
func (Constant) DirectionFlags() DirectionFlags { return DirectionContinuous }

// Emit and IsDelta preserve the source-level API used before directional
// emitter sampling was introduced.
func (e Constant) Emit(ctx bxdf.ShadingContext, wo maths.Direction) optics.Spectrum {
	return e.Eval(ctx, wo)
}
func (Constant) IsDelta() bool { return false }

func (e Constant) defaultEmitter() SurfaceEmitter {
	return NewSurfaceEmitter(e, NewUniform(TwoSided), PeakRadiance)
}
