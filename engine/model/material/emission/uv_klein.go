package emission

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

// UVKlein visualizes the Klein bottle UV chart. Hue follows u and brightness
// alternates along v, making the half-turn seam visible in projected renders.
type UVKlein struct {
	Saturation float64
	Lightness  float64
	VStripes   int
	Intensity  float64
}

func NewUVKlein(saturation, lightness float64, vStripes int, intensity float64) UVKlein {
	if saturation < 0 {
		saturation = 0
	}
	if saturation > 1 {
		saturation = 1
	}
	if lightness < 0 {
		lightness = 0
	}
	if lightness > 1 {
		lightness = 1
	}
	if vStripes <= 0 {
		vStripes = 1
	}
	if intensity < 0 {
		intensity = 0
	}
	return UVKlein{
		Saturation: saturation,
		Lightness:  lightness,
		VStripes:   vStripes,
		Intensity:  intensity,
	}
}

func (u UVKlein) Emit(ctx bxdf.ShadingContext, _ maths.Direction) optics.Spectrum {
	const twoPi = 2 * math.Pi

	hue := positiveMod(ctx.UV[0], twoPi) / twoPi
	v := positiveMod(ctx.UV[1], twoPi) / twoPi

	lightness := u.Lightness
	if int(math.Floor(v*float64(u.VStripes)*2))%2 == 1 {
		lightness *= 0.45
	}

	r, g, b := hslToRGB(hue, u.Saturation, lightness)
	return optics.NewSpectrum(r, g, b).MulScalar(u.Intensity)
}

func (UVKlein) IsDelta() bool { return false }

func positiveMod(v, period float64) float64 {
	v = math.Mod(v, period)
	if v < 0 {
		v += period
	}
	return v
}

func hslToRGB(h, s, l float64) (float64, float64, float64) {
	if s == 0 {
		return l, l, l
	}

	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q

	return hueToRGB(p, q, h+1.0/3.0),
		hueToRGB(p, q, h),
		hueToRGB(p, q, h-1.0/3.0)
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 0.5:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}
