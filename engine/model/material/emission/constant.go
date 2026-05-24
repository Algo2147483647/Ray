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

func (e Constant) Emit(ctx bxdf.ShadingContext, _ maths.Direction) optics.Spectrum {
	return e.Radiance.Eval(ctx)
}

func (e Constant) IsDelta() bool {
	return false
}
