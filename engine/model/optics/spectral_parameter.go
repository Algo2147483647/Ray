package optics

import "math"

type ColorSpace string

const (
	ColorSpaceLinearSRGB ColorSpace = "linear_srgb"
	ColorSpaceSRGB       ColorSpace = "srgb"
	ColorSpaceACEScg     ColorSpace = "acescg"
)

type SpectrumBounds struct {
	Min Spectrum
	Max Spectrum
}

type WavelengthContext interface {
	SpectralWavelengthNM() float64
	SpectralWavelengthsNM() []float64
}

type SpectralParameter interface {
	Eval(ctx WavelengthContext) Spectrum
	Bounds() SpectrumBounds
}

func SrgbChannelToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func Clamp(v, minValue, maxValue float64) float64 {
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}
