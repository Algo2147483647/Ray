package spectrum_parameter

import "github.com/Algo2147483647/ray/engine/model/optics"

type RGBParameter struct {
	Value optics.Spectrum
	Space optics.ColorSpace
}

func NewRGBParameter(value optics.Spectrum) RGBParameter {
	return RGBParameter{
		Value: value,
		Space: optics.ColorSpaceLinearSRGB,
	}
}

func NewSRGBParameter(value optics.Spectrum) RGBParameter {
	return RGBParameter{
		Value: optics.NewSpectrum(
			optics.SrgbChannelToLinear(value.RGBChannel(0)),
			optics.SrgbChannelToLinear(value.RGBChannel(1)),
			optics.SrgbChannelToLinear(value.RGBChannel(2)),
		),
		Space: optics.ColorSpaceSRGB,
	}
}

func NewACEScgParameter(value optics.Spectrum) RGBParameter {
	// The renderer does not yet have a full color-management transform stack.
	// Store ACEScg values as authored linear values until output transforms land.
	return RGBParameter{
		Value: value,
		Space: optics.ColorSpaceACEScg,
	}
}

func (p RGBParameter) Eval(ctx optics.WavelengthContext) optics.Spectrum {
	if ctx != nil && len(ctx.SpectralWavelengthsNM()) > 0 {
		values := make([]float64, len(ctx.SpectralWavelengthsNM()))
		value := p.Value.AverageRGB()
		for i := range values {
			values[i] = value
		}
		return optics.NewSampledSpectrum(values)
	}
	return p.Value
}

func (p RGBParameter) Bounds() optics.SpectrumBounds {
	return optics.SpectrumBounds{Min: p.Value, Max: p.Value}
}
