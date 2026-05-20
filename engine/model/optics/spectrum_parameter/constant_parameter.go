package spectrum_parameter

import "github.com/Algo2147483647/ray/engine/model/optics"

type ConstantParameter struct {
	Value float64
}

func NewConstantParameter(value float64) ConstantParameter {
	return ConstantParameter{Value: value}
}

func (p ConstantParameter) Eval(ctx optics.WavelengthContext) optics.Spectrum {
	if ctx != nil && len(ctx.SpectralWavelengthsNM()) > 0 {
		values := make([]float64, len(ctx.SpectralWavelengthsNM()))
		for i := range values {
			values[i] = p.Value
		}
		return optics.NewSampledSpectrum(values)
	}
	return optics.ConstantSpectrum(p.Value)
}

func (p ConstantParameter) Bounds() optics.SpectrumBounds {
	value := optics.ConstantSpectrum(p.Value)
	return optics.SpectrumBounds{Min: value, Max: value}
}
