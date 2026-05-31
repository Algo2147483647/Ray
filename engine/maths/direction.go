package maths

import "math"

type Direction struct {
	X          float64
	Y          float64
	Z          float64
	Components []float64
}

func NewDirection(x, y, z float64) Direction {
	return Direction{X: x, Y: y, Z: z}
}

func NewDirectionFromComponents(values []float64) Direction {
	components := append([]float64(nil), values...)
	d := Direction{Components: components}
	if len(components) > 0 {
		d.X = components[0]
	}
	if len(components) > 1 {
		d.Y = components[1]
	}
	if len(components) > 2 {
		d.Z = components[2]
	}
	return d
}

func (d Direction) Len() int {
	if len(d.Components) > 0 {
		return len(d.Components)
	}
	return 3
}

func (d Direction) Component(i int) float64 {
	if len(d.Components) > 0 {
		if i < 0 || i >= len(d.Components) {
			return 0
		}
		return d.Components[i]
	}
	switch i {
	case 0:
		return d.X
	case 1:
		return d.Y
	case 2:
		return d.Z
	default:
		return 0
	}
}

func (d Direction) Add(other Direction) Direction {
	if len(d.Components) > 0 || len(other.Components) > 0 {
		dim := max(d.Len(), other.Len())
		components := make([]float64, dim)
		for i := 0; i < dim; i++ {
			components[i] = d.Component(i) + other.Component(i)
		}
		return NewDirectionFromComponents(components)
	}
	return NewDirection(d.X+other.X, d.Y+other.Y, d.Z+other.Z)
}

func (d Direction) MulScalar(v float64) Direction {
	if len(d.Components) > 0 {
		components := make([]float64, len(d.Components))
		for i, component := range d.Components {
			components[i] = component * v
		}
		return NewDirectionFromComponents(components)
	}
	return NewDirection(d.X*v, d.Y*v, d.Z*v)
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
	if len(d.Components) > 0 {
		return d.Components[len(d.Components)-1]
	}
	return d.Z
}

func AbsCosTheta(d Direction) float64 {
	return math.Abs(d.Z)
}

func SameHemisphere(a, b Direction) bool {
	return CosTheta(a)*CosTheta(b) > 0
}

func IsUpperHemisphere(d Direction) bool {
	return CosTheta(d) > 0
}
