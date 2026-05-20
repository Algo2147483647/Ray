package spectrum_parameter

import (
	"github.com/Algo2147483647/ray/engine/model/material/core"
	"math"
)

type BlackbodyParameter struct {
	Temperature float64
	Scale       float64
}

func NewBlackbodyParameter(temperature, scale float64) BlackbodyParameter {
	return BlackbodyParameter{
		Temperature: temperature,
		Scale:       scale,
	}
}

func (p BlackbodyParameter) Eval(ctx core.ShadingContext) optics.Spectrum {
	if p.Temperature <= 0 || p.Scale <= 0 {
		return core.Spectrum{}
	}
	if len(ctx.WavelengthsNM) > 0 {
		values := make([]float64, len(ctx.WavelengthsNM))
		for i, wavelengthNM := range ctx.WavelengthsNM {
			values[i] = p.Scale * relativeBlackbody(wavelengthNM, p.Temperature)
		}
		return core.NewSampledSpectrum(values)
	}
	if ctx.WavelengthNM > 0 {
		return core.ConstantSpectrum(p.Scale * relativeBlackbody(ctx.WavelengthNM, p.Temperature))
	}
	return ApproximateBlackbodyRGB(p.Temperature).MulScalar(p.Scale)
}

func (p BlackbodyParameter) Bounds() core.SpectrumBounds {
	return core.SpectrumBounds{
		Min: core.Spectrum{},
		Max: core.ConstantSpectrum(p.Scale),
	}
}

func relativeBlackbody(wavelengthNM, temperature float64) float64 {
	value := blackbodyPower(wavelengthNM, temperature)
	reference := blackbodyPower(560, temperature)
	if reference == 0 {
		return 0
	}
	return value / reference
}

func blackbodyPower(wavelengthNM, temperature float64) float64 {
	if wavelengthNM <= 0 || temperature <= 0 {
		return 0
	}
	const c2NMK = 1.438776877e7
	exponent := c2NMK / (wavelengthNM * temperature)
	if exponent > 700 {
		return 0
	}
	denominator := math.Pow(wavelengthNM, 5) * math.Expm1(exponent)
	if denominator == 0 {
		return 0
	}
	return 1 / denominator
}

func ApproximateBlackbodyRGB(temperature float64) optics.Spectrum {
	temp := core.Clamp(temperature/100, 10, 400)
	var r, g, b float64

	if temp <= 66 {
		r = 255
		g = 99.4708025861*math.Log(temp) - 161.1195681661
		if temp <= 19 {
			b = 0
		} else {
			b = 138.5177312231*math.Log(temp-10) - 305.0447927307
		}
	} else {
		r = 329.698727446 * math.Pow(temp-60, -0.1332047592)
		g = 288.1221695283 * math.Pow(temp-60, -0.0755148492)
		b = 255
	}

	return core.NewSpectrum(
		core.SrgbChannelToLinear(core.Clamp(r/255, 0, 1)),
		core.SrgbChannelToLinear(core.Clamp(g/255, 0, 1)),
		core.SrgbChannelToLinear(core.Clamp(b/255, 0, 1)),
	)
}
