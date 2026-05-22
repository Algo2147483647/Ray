package optics

import "math"

type Color3 [3]float64

type RGB = Color3
type XYZ = Color3
type ACEScg = Color3

func NewRGB(r, g, b float64) RGB {
	return RGB{r, g, b}
}

func NewXYZ(x, y, z float64) XYZ {
	return XYZ{x, y, z}
}

func NewACEScg(r, g, b float64) ACEScg {
	return ACEScg{r, g, b}
}

func (c Color3) R() float64 { return c[0] }
func (c Color3) G() float64 { return c[1] }
func (c Color3) B() float64 { return c[2] }

func (c Color3) At(i int) float64 {
	if i < 0 || i >= len(c) {
		return 0
	}
	return c[i]
}

func (c Color3) Add(other Color3) Color3 {
	return Color3{c[0] + other[0], c[1] + other[1], c[2] + other[2]}
}

func (c Color3) Mul(other Color3) Color3 {
	return Color3{c[0] * other[0], c[1] * other[1], c[2] * other[2]}
}

func (c Color3) MulScalar(v float64) Color3 {
	return Color3{c[0] * v, c[1] * v, c[2] * v}
}

func (c Color3) DivScalar(v float64) Color3 {
	if v == 0 {
		return Color3{}
	}
	return Color3{c[0] / v, c[1] / v, c[2] / v}
}

func (c Color3) Max() float64 {
	return math.Max(c[0], math.Max(c[1], c[2]))
}

func (c Color3) Average() float64 {
	return (c[0] + c[1] + c[2]) / 3
}

func (c Color3) IsZero() bool {
	return c[0] == 0 && c[1] == 0 && c[2] == 0
}

func (c Color3) IsFinite() bool {
	for _, value := range c {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

func (c Color3) IsNonNegative() bool {
	return c[0] >= 0 && c[1] >= 0 && c[2] >= 0
}

func (c Color3) ClampMin(minValue float64) Color3 {
	return Color3{
		math.Max(minValue, c[0]),
		math.Max(minValue, c[1]),
		math.Max(minValue, c[2]),
	}
}
