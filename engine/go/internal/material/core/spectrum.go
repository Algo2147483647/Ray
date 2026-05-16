package core

import "math"

type Spectrum struct {
	R float64
	G float64
	B float64
}

func NewSpectrum(r, g, b float64) Spectrum {
	return Spectrum{R: r, G: g, B: b}
}

func ConstantSpectrum(v float64) Spectrum {
	return Spectrum{R: v, G: v, B: v}
}

func (s Spectrum) Add(other Spectrum) Spectrum {
	return Spectrum{
		R: s.R + other.R,
		G: s.G + other.G,
		B: s.B + other.B,
	}
}

func (s Spectrum) MulScalar(v float64) Spectrum {
	return Spectrum{
		R: s.R * v,
		G: s.G * v,
		B: s.B * v,
	}
}

func (s Spectrum) Mul(other Spectrum) Spectrum {
	return Spectrum{
		R: s.R * other.R,
		G: s.G * other.G,
		B: s.B * other.B,
	}
}

func (s Spectrum) DivScalar(v float64) Spectrum {
	if v == 0 {
		return Spectrum{}
	}
	return Spectrum{
		R: s.R / v,
		G: s.G / v,
		B: s.B / v,
	}
}

func (s Spectrum) MaxComponent() float64 {
	return math.Max(s.R, math.Max(s.G, s.B))
}

func (s Spectrum) IsFinite() bool {
	return isFinite(s.R) && isFinite(s.G) && isFinite(s.B)
}

func (s Spectrum) IsNonNegative() bool {
	return s.R >= 0 && s.G >= 0 && s.B >= 0
}

func (s Spectrum) AlmostEqual(other Spectrum, eps float64) bool {
	return math.Abs(s.R-other.R) <= eps &&
		math.Abs(s.G-other.G) <= eps &&
		math.Abs(s.B-other.B) <= eps
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
