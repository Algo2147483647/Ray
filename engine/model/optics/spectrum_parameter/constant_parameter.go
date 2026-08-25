package spectrum_parameter

import "github.com/Algo2147483647/ray/engine/model/optics"

type ConstantParameter struct {
	Value float64
}

func NewConstantParameter(value float64) ConstantParameter {
	return ConstantParameter{Value: value}
}

func (p ConstantParameter) Eval(ctx optics.WavelengthContext) optics.Spectrum {
	if wavelengths := optics.ContextWavelengthsNM(ctx); len(wavelengths) > 0 {
		if len(wavelengths) == 1 {
			return optics.NewSpectralPower(p.Value)
		}
		values := make([]float64, len(wavelengths))
		for i := range values {
			values[i] = p.Value
		}
		return optics.NewSampledSpectrum(values)
	}
	if ctx != nil && ctx.SpectralWavelengthNM() > 0 {
		return optics.NewSpectralPower(p.Value)
	}
	return optics.ConstantSpectrum(p.Value)
}

func (p ConstantParameter) Bounds() optics.SpectrumBounds {
	value := optics.ConstantSpectrum(p.Value)
	return optics.SpectrumBounds{Min: value, Max: value}
}
