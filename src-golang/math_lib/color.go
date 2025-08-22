package math_lib

import (
	"gonum.org/v1/gonum/mat"
	"math"
)

func SpectrumHSV(t float64) (r, g, b float64) {
	// 将波长映射到 HSV 的色调 (H) 分量, 380nm(紫) -> 270°, 780nm(红) -> 0°
	hue := 360.0 * (1.0 - t)
	saturation := 1.0 // 固定饱和度和亮度为最大值
	value := 1.0

	// HSV 转 RGB
	c := value * saturation
	x := c * (1 - math.Abs(math.Mod(hue/60, 2)-1))
	m := value - c

	switch {
	case hue < 60:
		r, g, b = c, x, 0
	case hue < 120:
		r, g, b = x, c, 0
	case hue < 180:
		r, g, b = 0, c, x
	case hue < 240:
		r, g, b = 0, x, c
	case hue < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	r = r + m
	g = g + m
	b = b + m
	return r, g, b
}

func SpectrumFiveDivided(t float64) (r, g, b float64) {
	switch {
	case t < 1.0/5:
		r = 5.0 / 4 * t
		g = 5.0 / 4 * t * 2
		b = 1.0
	case t < 1.0/5*2:
		r = 5.0 / 4 * t
		g = 5.0 / 4 * t * 2
		b = 1.0 - 5.0/4*(t-1.0/5)
	case t < 1.0/5*3:
		r = 5.0 / 4 * t
		g = 1
		b = 1.0 - 5.0/4*(t-1.0/5)
	case t < 1.0/5*4:
		r = 5.0 / 4 * t
		g = 1 - 5.0/4*2*(t-3.0/5)
		b = 1.0 - 5.0/4*(t-1.0/5)
	default:
		r = 1
		g = 1 - 5.0/4*2*(t-3.0/5)
		b = 1.0 - 5.0/4*(t-1.0/5)
	}

	return
}

func SpectrumTrisection(t float64) (r, g, b float64) {
	switch {
	case t < 1.0/2:
		r = t
		g = t * 2
		b = 1.0 - t
	default:
		r = t
		g = 1 - (t-0.5)*2
		b = 1.0 - t
	}

	return
}

func CalculateLuminance(c *mat.VecDense) float64 {
	r := c.AtVec(0)
	g := c.AtVec(1)
	b := c.AtVec(2)
	return 0.2126*r + 0.7152*g + 0.0722*b
}
