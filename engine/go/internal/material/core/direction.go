package core

import "math"

type Direction struct {
	X float64
	Y float64
	Z float64
}

func NewDirection(x, y, z float64) Direction {
	return Direction{X: x, Y: y, Z: z}
}

func (d Direction) Add(other Direction) Direction {
	return Direction{
		X: d.X + other.X,
		Y: d.Y + other.Y,
		Z: d.Z + other.Z,
	}
}

func (d Direction) MulScalar(v float64) Direction {
	return Direction{
		X: d.X * v,
		Y: d.Y * v,
		Z: d.Z * v,
	}
}

func (d Direction) Dot(other Direction) float64 {
	return d.X*other.X + d.Y*other.Y + d.Z*other.Z
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
	return isFinite(d.X) && isFinite(d.Y) && isFinite(d.Z)
}

func CosTheta(d Direction) float64 {
	return d.Z
}

func AbsCosTheta(d Direction) float64 {
	return math.Abs(d.Z)
}

func SameHemisphere(a, b Direction) bool {
	return a.Z*b.Z > 0
}

func IsUpperHemisphere(d Direction) bool {
	return d.Z > 0
}
