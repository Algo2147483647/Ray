package optics

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
)

type SpectrumKind int

const (
	SpectrumKindRGB SpectrumKind = iota
	SpectrumKindScalar
	SpectrumKindSampled
)

// Spectrum is a renderer-space value, not an authored color.
// RGB values are scene-linear sRGB. Sampled values are aligned with
// ShadingContext.WavelengthsNM and intentionally do not mirror RGB channels.
type Spectrum struct {
	Kind    SpectrumKind
	RGB     RGB
	Power   float64
	Samples []float64
}

func NewSpectrum(r, g, b float64) Spectrum {
	return NewRGBSpectrum(r, g, b)
}

func NewRGBSpectrum(r, g, b float64) Spectrum {
	return Spectrum{
		Kind: SpectrumKindRGB,
		RGB:  NewRGB(r, g, b),
	}
}

func ConstantSpectrum(v float64) Spectrum {
	return NewSpectrum(v, v, v)
}

func NewSampledSpectrum(samples []float64) Spectrum {
	if len(samples) == 0 {
		return Spectrum{}
	}
	if len(samples) == 1 {
		return Spectrum{Kind: SpectrumKindScalar, Power: samples[0]}
	}
	return Spectrum{
		Kind:    SpectrumKindSampled,
		Samples: append([]float64(nil), samples...),
	}
}

func (s Spectrum) Clone() Spectrum {
	if s.Kind == SpectrumKindScalar {
		return Spectrum{Kind: SpectrumKindScalar, Power: s.Power}
	}
	if s.HasSamples() {
		return NewSampledSpectrum(s.Samples)
	}
	return NewRGBSpectrum(s.RGB[0], s.RGB[1], s.RGB[2])
}

func (s Spectrum) HasSamples() bool {
	return s.Kind == SpectrumKindScalar || (s.Kind == SpectrumKindSampled && len(s.Samples) > 0)
}

func (s Spectrum) SampleCount() int {
	if s.Kind == SpectrumKindScalar {
		return 1
	}
	return len(s.Samples)
}

func (s Spectrum) RGBChannel(i int) float64 {
	if i < 0 || i >= len(s.RGB) {
		return 0
	}
	return s.RGB[i]
}

func (s Spectrum) Sample(i int) float64 {
	if s.Kind == SpectrumKindScalar {
		if i == 0 {
			return s.Power
		}
		return 0
	}
	if i < 0 || i >= len(s.Samples) {
		return 0
	}
	return s.Samples[i]
}

// PowerAt resolves a renderer value to one scalar spectral power sample.
// Authored RGB is uplifted only at this boundary; multi-sample values cannot
// be collapsed without an explicit wavelength-to-index association.
func (s Spectrum) PowerAt(wavelengthNM float64) (float64, bool) {
	if s.HasSamples() {
		if s.SampleCount() != 1 {
			return 0, false
		}
		return s.Sample(0), true
	}
	if wavelengthNM <= 0 {
		return 0, false
	}
	return s.RGBPowerAtWavelength(wavelengthNM), true
}

func (s Spectrum) Average() float64 {
	if s.Kind == SpectrumKindScalar {
		return s.Power
	}
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
	if s.HasSamples() && other.HasSamples() {
		return combineSampled(s, other, func(a, b float64) float64 { return a + b })
	}
	if s.HasSamples() || other.HasSamples() {
		if s.IsZero() {
			return other.Clone()
		}
		if other.IsZero() {
			return s.Clone()
		}
		return Spectrum{}
	}
	return Spectrum{
		Kind: SpectrumKindRGB,
		RGB:  s.RGB.Add(other.RGB),
	}
}

func (s Spectrum) MulScalar(v float64) Spectrum {
	if s.HasSamples() {
		return mapSampled(s, func(a float64) float64 { return a * v })
	}
	return Spectrum{
		Kind: SpectrumKindRGB,
		RGB:  s.RGB.MulScalar(v),
	}
}

func (s Spectrum) Mul(other Spectrum) Spectrum {
	if s.HasSamples() && other.HasSamples() {
		return combineSampled(s, other, func(a, b float64) float64 { return a * b })
	}
	if s.HasSamples() || other.HasSamples() {
		if s.IsZero() || other.IsZero() {
			return zeroLikeSampled(s, other)
		}
		return Spectrum{}
	}
	return Spectrum{
		Kind: SpectrumKindRGB,
		RGB:  s.RGB.Mul(other.RGB),
	}
}

func (s Spectrum) IsZero() bool {
	if s.HasSamples() {
		for i := 0; i < s.SampleCount(); i++ {
			if s.Sample(i) != 0 {
				return false
			}
		}
		return true
	}
	return s.RGB.IsZero()
}

func (s Spectrum) UpliftRGBToSampled(wavelengthsNM []float64) Spectrum {
	if s.HasSamples() || len(wavelengthsNM) == 0 {
		return s.Clone()
	}
	samples := make([]float64, len(wavelengthsNM))
	for i, wavelengthNM := range wavelengthsNM {
		samples[i] = s.RGBPowerAtWavelength(wavelengthNM)
	}
	return NewSampledSpectrum(samples)
}

func (s Spectrum) UpliftRGBReflectanceToSampled(wavelengthsNM []float64) Spectrum {
	if s.HasSamples() || len(wavelengthsNM) == 0 {
		return s.Clone()
	}
	maxReflectance := s.MaxComponent()
	if maxReflectance <= 0 {
		return NewSampledSpectrum(make([]float64, len(wavelengthsNM)))
	}
	samples := make([]float64, len(wavelengthsNM))
	for i, wavelengthNM := range wavelengthsNM {
		samples[i] = math.Min(maxReflectance, s.RGBPowerAtWavelength(wavelengthNM))
	}
	return NewSampledSpectrum(samples)
}

func (s Spectrum) RGBPowerAtWavelength(wavelengthNM float64) float64 {
	if s.HasSamples() {
		return s.Sample(0)
	}
	weight := RGBWeight(wavelengthNM)
	return math.Max(0,
		s.RGB[0]*weight[0]+
			s.RGB[1]*weight[1]+
			s.RGB[2]*weight[2],
	)
}

func (s Spectrum) DivScalar(v float64) Spectrum {
	if v == 0 {
		return Spectrum{}
	}
	if s.HasSamples() {
		return mapSampled(s, func(a float64) float64 { return a / v })
	}
	return Spectrum{
		Kind: SpectrumKindRGB,
		RGB:  s.RGB.DivScalar(v),
	}
}

func (s Spectrum) MaxComponent() float64 {
	if s.HasSamples() {
		maxValue := s.Sample(0)
		for i := 1; i < s.SampleCount(); i++ {
			maxValue = math.Max(maxValue, s.Sample(i))
		}
		return maxValue
	}
	return s.RGB.Max()
}

func (s Spectrum) IsFinite() bool {
	if s.HasSamples() {
		for i := 0; i < s.SampleCount(); i++ {
			if !maths.IsFinite(s.Sample(i)) {
				return false
			}
		}
		return true
	}
	return s.RGB.IsFinite()
}

func (s Spectrum) IsNonNegative() bool {
	if s.HasSamples() {
		for i := 0; i < s.SampleCount(); i++ {
			if s.Sample(i) < 0 {
				return false
			}
		}
		return true
	}
	return s.RGB.IsNonNegative()
}

func (s Spectrum) AlmostEqual(other Spectrum, eps float64) bool {
	if s.HasSamples() || other.HasSamples() {
		if s.SampleCount() != other.SampleCount() {
			return false
		}
		for i := 0; i < s.SampleCount(); i++ {
			if math.Abs(s.Sample(i)-other.Sample(i)) > eps {
				return false
			}
		}
		return true
	}

	return math.Abs(s.RGB[0]-other.RGB[0]) <= eps &&
		math.Abs(s.RGB[1]-other.RGB[1]) <= eps &&
		math.Abs(s.RGB[2]-other.RGB[2]) <= eps
}

func combineSampled(a, b Spectrum, fn func(float64, float64) float64) Spectrum {
	count := a.SampleCount()
	if b.SampleCount() > count {
		count = b.SampleCount()
	}
	if count == 1 {
		return Spectrum{Kind: SpectrumKindScalar, Power: fn(a.Sample(0), b.Sample(0))}
	}
	result := make([]float64, count)
	for i := 0; i < count; i++ {
		result[i] = fn(a.Sample(i), b.Sample(i))
	}
	return NewSampledSpectrum(result)
}

func mapSampled(spectrum Spectrum, fn func(float64) float64) Spectrum {
	if spectrum.Kind == SpectrumKindScalar {
		return Spectrum{Kind: SpectrumKindScalar, Power: fn(spectrum.Power)}
	}
	result := make([]float64, spectrum.SampleCount())
	for i := range result {
		result[i] = fn(spectrum.Sample(i))
	}
	return NewSampledSpectrum(result)
}

func zeroLikeSampled(a, b Spectrum) Spectrum {
	count := a.SampleCount()
	if b.SampleCount() > count {
		count = b.SampleCount()
	}
	if count == 0 {
		return Spectrum{}
	}
	return NewSampledSpectrum(make([]float64, count))
}

func (s Spectrum) AverageRGB() float64 {
	return s.RGB.Average()
}
