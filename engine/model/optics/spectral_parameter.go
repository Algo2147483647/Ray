package optics

import "math"

type RGBColorSpace string

const (
	RGBColorSpaceLinearSRGB RGBColorSpace = "linear_srgb"
	RGBColorSpaceSRGB       RGBColorSpace = "srgb"
	RGBColorSpaceACEScg     RGBColorSpace = "acescg"
)

type SpectrumBounds struct {
	Min Spectrum
	Max Spectrum
}

type WavelengthContext interface {
	SpectralWavelengthNM() float64
}

// WavelengthBatchContext is an optional extension for explicit batched/offline
// parameter evaluation. Runtime transport contexts intentionally do not
// implement it because one path owns exactly one wavelength.
type WavelengthBatchContext interface {
	SpectralWavelengthsNM() []float64
}

func ContextWavelengthsNM(ctx WavelengthContext) []float64 {
	batch, ok := ctx.(WavelengthBatchContext)
	if !ok {
		return nil
	}
	return batch.SpectralWavelengthsNM()
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
