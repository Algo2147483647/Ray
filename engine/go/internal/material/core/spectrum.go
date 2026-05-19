package core

import "math"

type Spectrum struct {
	R       float64
	G       float64
	B       float64
	Samples []float64
}

func NewSpectrum(r, g, b float64) Spectrum {
	return Spectrum{R: r, G: g, B: b}
}

func ConstantSpectrum(v float64) Spectrum {
	return Spectrum{R: v, G: v, B: v}
}

func NewSampledSpectrum(samples []float64) Spectrum {
	copied := append([]float64(nil), samples...)
	r, g, b := 0.0, 0.0, 0.0
	if len(copied) > 0 {
		r = copied[0]
		g = copied[0]
		b = copied[0]
	}
	if len(copied) > 1 {
		g = copied[1]
	}
	if len(copied) > 2 {
		b = copied[2]
	}
	return Spectrum{R: r, G: g, B: b, Samples: copied}
}

func (s Spectrum) HasSamples() bool {
	return len(s.Samples) > 0
}

func (s Spectrum) SampleCount() int {
	return len(s.Samples)
}

func (s Spectrum) Add(other Spectrum) Spectrum {
	return Spectrum{
		R:       s.R + other.R,
		G:       s.G + other.G,
		B:       s.B + other.B,
		Samples: combineSamples(s, other, func(a, b float64) float64 { return a + b }),
	}
}

func (s Spectrum) MulScalar(v float64) Spectrum {
	return Spectrum{
		R:       s.R * v,
		G:       s.G * v,
		B:       s.B * v,
		Samples: mapSamples(s.Samples, func(a float64) float64 { return a * v }),
	}
}

func (s Spectrum) Mul(other Spectrum) Spectrum {
	return Spectrum{
		R:       s.R * other.R,
		G:       s.G * other.G,
		B:       s.B * other.B,
		Samples: combineSamples(s, other, func(a, b float64) float64 { return a * b }),
	}
}

func (s Spectrum) DivScalar(v float64) Spectrum {
	if v == 0 {
		return Spectrum{}
	}
	return Spectrum{
		R:       s.R / v,
		G:       s.G / v,
		B:       s.B / v,
		Samples: mapSamples(s.Samples, func(a float64) float64 { return a / v }),
	}
}

func (s Spectrum) MaxComponent() float64 {
	maxValue := math.Max(s.R, math.Max(s.G, s.B))
	for _, sample := range s.Samples {
		maxValue = math.Max(maxValue, sample)
	}
	return maxValue
}

func (s Spectrum) IsFinite() bool {
	if !isFinite(s.R) || !isFinite(s.G) || !isFinite(s.B) {
		return false
	}
	for _, sample := range s.Samples {
		if !isFinite(sample) {
			return false
		}
	}
	return true
}

func (s Spectrum) IsNonNegative() bool {
	if s.R < 0 || s.G < 0 || s.B < 0 {
		return false
	}
	for _, sample := range s.Samples {
		if sample < 0 {
			return false
		}
	}
	return true
}

func (s Spectrum) AlmostEqual(other Spectrum, eps float64) bool {
	if math.Abs(s.R-other.R) > eps ||
		math.Abs(s.G-other.G) > eps ||
		math.Abs(s.B-other.B) > eps {
		return false
	}
	if len(s.Samples) != len(other.Samples) {
		return false
	}
	for i := range s.Samples {
		if math.Abs(s.Samples[i]-other.Samples[i]) > eps {
			return false
		}
	}
	return true
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
	return s.averageRGB()
}

func (s Spectrum) averageRGB() float64 {
	return (s.R + s.G + s.B) / 3
}
