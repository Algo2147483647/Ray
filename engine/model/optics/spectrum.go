package optics

import "math"

type SpectrumKind int

const (
	SpectrumKindRGB SpectrumKind = iota
	SpectrumKindSampled
)

// Spectrum is a renderer-space value, not an authored color.
// RGB values are scene-linear sRGB. Sampled values are aligned with
// ShadingContext.WavelengthsNM and intentionally do not mirror RGB channels.
type Spectrum struct {
	Kind    SpectrumKind
	RGB     [3]float64
	Samples []float64
}

func NewSpectrum(r, g, b float64) Spectrum {
	return NewRGBSpectrum(r, g, b)
}

func NewRGBSpectrum(r, g, b float64) Spectrum {
	return Spectrum{
		Kind: SpectrumKindRGB,
		RGB:  [3]float64{r, g, b},
	}
}

func ConstantSpectrum(v float64) Spectrum {
	return NewSpectrum(v, v, v)
}

func NewSampledSpectrum(samples []float64) Spectrum {
	return Spectrum{
		Kind:    SpectrumKindSampled,
		Samples: append([]float64(nil), samples...),
	}
}

func (s Spectrum) HasSamples() bool {
	return s.Kind == SpectrumKindSampled && len(s.Samples) > 0
}

func (s Spectrum) SampleCount() int {
	return len(s.Samples)
}

func (s Spectrum) RGBChannel(i int) float64 {
	if i < 0 || i >= len(s.RGB) {
		return 0
	}
	return s.RGB[i]
}

func (s Spectrum) Average() float64 {
	if s.HasSamples() {
		sum := 0.0
		for _, sample := range s.Samples {
			sum += sample
		}
		return sum / float64(len(s.Samples))
	}
	return s.AverageRGB()
}

func (s Spectrum) Add(other Spectrum) Spectrum {
	if s.HasSamples() || other.HasSamples() {
		return NewSampledSpectrum(combineSamples(s, other, func(a, b float64) float64 { return a + b }))
	}
	return Spectrum{
		Kind: SpectrumKindRGB,
		RGB: [3]float64{
			s.RGB[0] + other.RGB[0],
			s.RGB[1] + other.RGB[1],
			s.RGB[2] + other.RGB[2],
		},
	}
}

func (s Spectrum) MulScalar(v float64) Spectrum {
	if s.HasSamples() {
		return NewSampledSpectrum(mapSamples(s.Samples, func(a float64) float64 { return a * v }))
	}
	return Spectrum{
		Kind: SpectrumKindRGB,
		RGB: [3]float64{
			s.RGB[0] * v,
			s.RGB[1] * v,
			s.RGB[2] * v,
		},
	}
}

func (s Spectrum) Mul(other Spectrum) Spectrum {
	if s.HasSamples() || other.HasSamples() {
		return NewSampledSpectrum(combineSamples(s, other, func(a, b float64) float64 { return a * b }))
	}
	return Spectrum{
		Kind: SpectrumKindRGB,
		RGB: [3]float64{
			s.RGB[0] * other.RGB[0],
			s.RGB[1] * other.RGB[1],
			s.RGB[2] * other.RGB[2],
		},
	}
}

func (s Spectrum) DivScalar(v float64) Spectrum {
	if v == 0 {
		return Spectrum{}
	}
	if s.HasSamples() {
		return NewSampledSpectrum(mapSamples(s.Samples, func(a float64) float64 { return a / v }))
	}
	return Spectrum{
		Kind: SpectrumKindRGB,
		RGB: [3]float64{
			s.RGB[0] / v,
			s.RGB[1] / v,
			s.RGB[2] / v,
		},
	}
}

func (s Spectrum) MaxComponent() float64 {
	if s.HasSamples() {
		maxValue := s.Samples[0]
		for _, sample := range s.Samples[1:] {
			maxValue = math.Max(maxValue, sample)
		}
		return maxValue
	}
	return math.Max(s.RGB[0], math.Max(s.RGB[1], s.RGB[2]))
}

func (s Spectrum) IsFinite() bool {
	if s.HasSamples() {
		for _, sample := range s.Samples {
			if !isFinite(sample) {
				return false
			}
		}
		return true
	}
	for _, value := range s.RGB {
		if !isFinite(value) {
			return false
		}
	}
	return true
}

func (s Spectrum) IsNonNegative() bool {
	if s.HasSamples() {
		for _, sample := range s.Samples {
			if sample < 0 {
				return false
			}
		}
		return true
	}
	for _, value := range s.RGB {
		if value < 0 {
			return false
		}
	}
	return true
}

func (s Spectrum) AlmostEqual(other Spectrum, eps float64) bool {
	if s.HasSamples() || other.HasSamples() {
		if s.SampleCount() != other.SampleCount() {
			return false
		}
		for i := 0; i < s.SampleCount(); i++ {
			if math.Abs(sampleAt(s, i)-sampleAt(other, i)) > eps {
				return false
			}
		}
		return true
	}

	return math.Abs(s.RGB[0]-other.RGB[0]) <= eps &&
		math.Abs(s.RGB[1]-other.RGB[1]) <= eps &&
		math.Abs(s.RGB[2]-other.RGB[2]) <= eps
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func combineSamples(a, b Spectrum, fn func(float64, float64) float64) []float64 {
	if !a.HasSamples() && !b.HasSamples() {
		return nil
	}
	count := a.SampleCount()
	if b.SampleCount() > count {
		count = b.SampleCount()
	}
	result := make([]float64, count)
	for i := 0; i < count; i++ {
		result[i] = fn(sampleAt(a, i), sampleAt(b, i))
	}
	return result
}

func mapSamples(samples []float64, fn func(float64) float64) []float64 {
	if len(samples) == 0 {
		return nil
	}
	result := make([]float64, len(samples))
	for i, sample := range samples {
		result[i] = fn(sample)
	}
	return result
}

func sampleAt(s Spectrum, i int) float64 {
	if i < len(s.Samples) {
		return s.Samples[i]
	}
	return s.AverageRGB()
}

func (s Spectrum) AverageRGB() float64 {
	return (s.RGB[0] + s.RGB[1] + s.RGB[2]) / 3
}
