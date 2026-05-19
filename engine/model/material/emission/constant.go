package emission

import "github.com/Algo2147483647/ray/engine/model/material/core"

type Constant struct {
	Radiance core.SpectralParameter
}

func NewConstant(color core.Spectrum) Constant {
	return NewConstantParameter(core.NewRGBParameter(color))
}

func NewConstantParameter(radiance core.SpectralParameter) Constant {
	return Constant{Radiance: radiance}
}

func (e Constant) Emit(ctx core.ShadingContext, _ core.Direction) core.Spectrum {
	return e.Radiance.Eval(ctx)
}

func (e Constant) IsDelta() bool {
	return false
}
