package spectrum_parameter

import "github.com/Algo2147483647/ray/engine/model/material/core"

type ConstantParameter struct {
	Value float64
}

func NewConstantParameter(value float64) ConstantParameter {
	return ConstantParameter{Value: value}
}

func (p ConstantParameter) Eval(ctx core.ShadingContext) core.Spectrum {
	if len(ctx.WavelengthsNM) > 0 {
		values := make([]float64, len(ctx.WavelengthsNM))
		for i := range values {
			values[i] = p.Value
		}
		return core.NewSampledSpectrum(values)
	}
	return core.ConstantSpectrum(p.Value)
}

func (p ConstantParameter) Bounds() core.SpectrumBounds {
	value := core.ConstantSpectrum(p.Value)
	return core.SpectrumBounds{Min: value, Max: value}
}
