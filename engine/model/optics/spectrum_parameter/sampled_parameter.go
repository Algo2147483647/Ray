package spectrum_parameter

import (
	"github.com/Algo2147483647/ray/engine/model/material/core"
	"math"
)

type SampledParameter struct {
	WavelengthsNM []float64
	Values        []float64
}

func NewSampledParameter(wavelengthsNM, values []float64) SampledParameter {
	wavelengths := append([]float64(nil), wavelengthsNM...)
	sampledValues := append([]float64(nil), values...)
	return SampledParameter{
		WavelengthsNM: wavelengths,
		Values:        sampledValues,
	}
}

func (p SampledParameter) Eval(ctx core.ShadingContext) optics.Spectrum {
	if len(p.WavelengthsNM) == 0 || len(p.Values) == 0 {
		return core.Spectrum{}
	}
	if len(ctx.WavelengthsNM) > 0 {
		values := make([]float64, len(ctx.WavelengthsNM))
		for i, wavelengthNM := range ctx.WavelengthsNM {
			values[i] = p.valueAt(wavelengthNM)
		}
		return core.NewSampledSpectrum(values)
	}
	if ctx.WavelengthNM > 0 {
		return core.ConstantSpectrum(p.valueAt(ctx.WavelengthNM))
	}

	sum := 0.0
	for _, value := range p.Values {
		sum += value
	}
	return core.ConstantSpectrum(sum / float64(len(p.Values)))
}

func (p SampledParameter) Bounds() core.SpectrumBounds {
	if len(p.Values) == 0 {
		return core.SpectrumBounds{}
	}
	minValue := p.Values[0]
	maxValue := p.Values[0]
	for _, value := range p.Values[1:] {
		minValue = math.Min(minValue, value)
		maxValue = math.Max(maxValue, value)
	}
	return core.SpectrumBounds{
		Min: core.ConstantSpectrum(minValue),
		Max: core.ConstantSpectrum(maxValue),
	}
}

func (p SampledParameter) valueAt(wavelengthNM float64) float64 {
	if wavelengthNM <= p.WavelengthsNM[0] {
		return p.Values[0]
	}
	last := len(p.WavelengthsNM) - 1
	if wavelengthNM >= p.WavelengthsNM[last] {
		return p.Values[last]
	}

	for i := 0; i < last; i++ {
		left := p.WavelengthsNM[i]
		right := p.WavelengthsNM[i+1]
		if wavelengthNM >= left && wavelengthNM <= right {
			t := (wavelengthNM - left) / (right - left)
			return p.Values[i]*(1-t) + p.Values[i+1]*t
		}
	}
	return p.Values[last]
}
