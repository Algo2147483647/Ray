package core

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

type SpectralParameter interface {
	Eval(ctx ShadingContext) Spectrum
	Bounds() SpectrumBounds
}

type RGBParameter struct {
	Value Spectrum
	Space ColorSpace
}

func NewRGBParameter(value Spectrum) RGBParameter {
	return RGBParameter{
		Value: value,
		Space: ColorSpaceLinearSRGB,
	}
}

func NewSRGBParameter(value Spectrum) RGBParameter {
	return RGBParameter{
		Value: NewSpectrum(
			srgbChannelToLinear(value.R),
			srgbChannelToLinear(value.G),
			srgbChannelToLinear(value.B),
		),
		Space: ColorSpaceSRGB,
	}
}

func NewACEScgParameter(value Spectrum) RGBParameter {
	// The renderer does not yet have a full color-management transform stack.
	// Store ACEScg values as authored linear values until output transforms land.
	return RGBParameter{
		Value: value,
		Space: ColorSpaceACEScg,
	}
}

func (p RGBParameter) Eval(ShadingContext) Spectrum {
	return p.Value
}

func (p RGBParameter) Bounds() SpectrumBounds {
	return SpectrumBounds{Min: p.Value, Max: p.Value}
}

type ConstantParameter struct {
	Value float64
}

func NewConstantParameter(value float64) ConstantParameter {
	return ConstantParameter{Value: value}
}

func (p ConstantParameter) Eval(ShadingContext) Spectrum {
	return ConstantSpectrum(p.Value)
}

func (p ConstantParameter) Bounds() SpectrumBounds {
	value := ConstantSpectrum(p.Value)
	return SpectrumBounds{Min: value, Max: value}
}

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

func (p SampledParameter) Eval(ctx ShadingContext) Spectrum {
	if len(p.WavelengthsNM) == 0 || len(p.Values) == 0 {
		return Spectrum{}
	}
	if ctx.WavelengthNM > 0 {
		return ConstantSpectrum(p.valueAt(ctx.WavelengthNM))
	}

	sum := 0.0
	for _, value := range p.Values {
		sum += value
	}
	return ConstantSpectrum(sum / float64(len(p.Values)))
}

func (p SampledParameter) Bounds() SpectrumBounds {
	if len(p.Values) == 0 {
		return SpectrumBounds{}
	}
	minValue := p.Values[0]
	maxValue := p.Values[0]
	for _, value := range p.Values[1:] {
		minValue = math.Min(minValue, value)
		maxValue = math.Max(maxValue, value)
	}
	return SpectrumBounds{
		Min: ConstantSpectrum(minValue),
		Max: ConstantSpectrum(maxValue),
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

func (p BlackbodyParameter) Eval(ctx ShadingContext) Spectrum {
	if p.Temperature <= 0 || p.Scale <= 0 {
		return Spectrum{}
	}
	if ctx.WavelengthNM > 0 {
		return ConstantSpectrum(p.Scale * relativeBlackbody(ctx.WavelengthNM, p.Temperature))
	}
	return approximateBlackbodyRGB(p.Temperature).MulScalar(p.Scale)
}

func (p BlackbodyParameter) Bounds() SpectrumBounds {
	return SpectrumBounds{
		Min: Spectrum{},
		Max: ConstantSpectrum(p.Scale),
	}
}

func srgbChannelToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
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

func approximateBlackbodyRGB(temperature float64) Spectrum {
	temp := clamp(temperature/100, 10, 400)
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

	return NewSpectrum(
		srgbChannelToLinear(clamp(r/255, 0, 1)),
		srgbChannelToLinear(clamp(g/255, 0, 1)),
		srgbChannelToLinear(clamp(b/255, 0, 1)),
	)
}

func clamp(v, minValue, maxValue float64) float64 {
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}
