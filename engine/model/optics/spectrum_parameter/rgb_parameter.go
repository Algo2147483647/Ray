package spectrum_parameter

import "github.com/Algo2147483647/ray/engine/model/optics"

type RGBParameter struct {
	Value optics.Spectrum
	Space optics.RGBColorSpace
}

func NewRGBParameter(value optics.Spectrum) RGBParameter {
	return RGBParameter{
		Value: value,
		Space: optics.RGBColorSpaceLinearSRGB,
	}
}

func NewSRGBParameter(value optics.Spectrum) RGBParameter {
	return RGBParameter{
		Value: optics.NewSpectrum(
			optics.SrgbChannelToLinear(value.RGBChannel(0)),
			optics.SrgbChannelToLinear(value.RGBChannel(1)),
			optics.SrgbChannelToLinear(value.RGBChannel(2)),
		),
		Space: optics.RGBColorSpaceSRGB,
	}
}

func NewACEScgParameter(value optics.Spectrum) RGBParameter {
	// The renderer does not yet have a full color-management transform stack.
	// Store ACEScg values as authored linear values until output transforms land.
	return RGBParameter{
		Value: value,
		Space: optics.RGBColorSpaceACEScg,
	}
}

func (p RGBParameter) Eval(ctx optics.WavelengthContext) optics.Spectrum {
	if ctx != nil {
		if wavelengths := ctx.SpectralWavelengthsNM(); len(wavelengths) > 0 {
			return p.Value.UpliftRGBReflectanceToSampled(wavelengths)
		}
		if wavelength := ctx.SpectralWavelengthNM(); wavelength > 0 {
			return p.Value.UpliftRGBReflectanceToSampled([]float64{wavelength})
		}
	}
	return p.Value
}

func (p RGBParameter) Bounds() optics.SpectrumBounds {
	return optics.SpectrumBounds{Min: p.Value, Max: p.Value}
}
