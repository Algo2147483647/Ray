package maths

import "math"

type Direction []float64

func NewDirection(x, y, z float64) Direction {
	return Direction{x, y, z}
}

func NewDirectionFromComponents(values []float64) Direction {
	return Direction(append([]float64(nil), values...))
}

func (d Direction) Len() int {
	return len(d)
}

func (d Direction) Component(i int) float64 {
	if i < 0 || i >= len(d) {
		return 0
	}
	return d[i]
}

func (d Direction) Add(other Direction) Direction {
	dim := max(d.Len(), other.Len())
	components := make([]float64, dim)
	for i := 0; i < dim; i++ {
		components[i] = d.Component(i) + other.Component(i)
	}
	return Direction(components)
}

func (d Direction) MulScalar(v float64) Direction {
	components := make([]float64, len(d))
	for i, component := range d {
		components[i] = component * v
	}
	return Direction(components)
}

func (d Direction) Dot(other Direction) float64 {
	dim := max(d.Len(), other.Len())
	var dot float64
	for i := 0; i < dim; i++ {
		dot += d.Component(i) * other.Component(i)
	}
	return dot
}

func (d Direction) Length() float64 {
	return math.Sqrt(d.Dot(d))
}

func (d Direction) Normalize() Direction {
	length := d.Length()
	if length == 0 {
		return Direction{}
	}
	return d.MulScalar(1 / length)
}

func (d Direction) IsFinite() bool {
	for i := 0; i < d.Len(); i++ {
		if !isFinite(d.Component(i)) {
			return false
		}
	}
	return true
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func CosTheta(d Direction) float64 {
	if len(d) == 0 {
		return 0
	}
	return d[len(d)-1]
}

func AbsCosTheta(d Direction) float64 {
	return math.Abs(CosTheta(d))
}

func SameHemisphere(a, b Direction) bool {
	return CosTheta(a)*CosTheta(b) > 0
}

func IsUpperHemisphere(d Direction) bool {
	return CosTheta(d) > 0
}
