package spectrum_parameter

import "github.com/Algo2147483647/ray/engine/model/material/core"

type RGBParameter struct {
	Value core.Spectrum
	Space core.ColorSpace
}

func NewRGBParameter(value core.Spectrum) RGBParameter {
	return RGBParameter{
		Value: value,
		Space: core.ColorSpaceLinearSRGB,
	}
}

func NewSRGBParameter(value core.Spectrum) RGBParameter {
	return RGBParameter{
		Value: core.NewSpectrum(
			core.SrgbChannelToLinear(value.RGBChannel(0)),
			core.SrgbChannelToLinear(value.RGBChannel(1)),
			core.SrgbChannelToLinear(value.RGBChannel(2)),
		),
		Space: core.ColorSpaceSRGB,
	}
}

func NewACEScgParameter(value core.Spectrum) RGBParameter {
	// The renderer does not yet have a full color-management transform stack.
	// Store ACEScg values as authored linear values until output transforms land.
	return RGBParameter{
		Value: value,
		Space: core.ColorSpaceACEScg,
	}
}

func (p RGBParameter) Eval(ctx core.ShadingContext) optics.Spectrum {
	if len(ctx.WavelengthsNM) > 0 {
		values := make([]float64, len(ctx.WavelengthsNM))
		value := p.Value.AverageRGB()
		for i := range values {
			values[i] = value
		}
		return core.NewSampledSpectrum(values)
	}
	return p.Value
}

func (p RGBParameter) Bounds() core.SpectrumBounds {
	return core.SpectrumBounds{Min: p.Value, Max: p.Value}
}
