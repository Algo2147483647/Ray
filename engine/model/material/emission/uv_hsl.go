package emission

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

// UVHSL is a geometry-independent UV-to-color procedural field. U controls
// hue and optional V stripes modulate lightness.
type UVHSL struct {
	Saturation float64
	Lightness  float64
	VStripes   int
	Intensity  float64
}

func NewUVHSL(saturation, lightness float64, vStripes int, intensity float64) UVHSL {
	return UVHSL{
		Saturation: clampUnit(saturation),
		Lightness:  clampUnit(lightness),
		VStripes:   max(1, vStripes),
		Intensity:  math.Max(0, intensity),
	}
}

func (u UVHSL) EvaluateRadiance(ctx bxdf.ShadingContext) optics.Spectrum {
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

func (u UVHSL) Eval(ctx bxdf.ShadingContext, wo maths.Direction) optics.Spectrum {
	return u.defaultEmitter().Eval(ctx, wo)
}
func (u UVHSL) SampleDirection(ctx bxdf.ShadingContext, sample maths.Sample2D) DirectionSample {
	return u.defaultEmitter().SampleDirection(ctx, sample)
}
func (u UVHSL) PDFDirection(ctx bxdf.ShadingContext, wo maths.Direction) float64 {
	return u.defaultEmitter().PDFDirection(ctx, wo)
}
func (u UVHSL) ExitanceEstimate(ctx bxdf.ShadingContext) optics.Spectrum {
	return u.defaultEmitter().ExitanceEstimate(ctx)
}
func (UVHSL) DirectionFlags() DirectionFlags { return DirectionContinuous }
func (u UVHSL) Emit(ctx bxdf.ShadingContext, wo maths.Direction) optics.Spectrum {
	return u.Eval(ctx, wo)
}
func (UVHSL) IsDelta() bool { return false }
func (u UVHSL) defaultEmitter() SurfaceEmitter {
	return NewSurfaceEmitter(u, NewUniform(TwoSided), PeakRadiance)
}

func positiveMod(value, period float64) float64 {
	value = math.Mod(value, period)
	if value < 0 {
		value += period
	}
	return value
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
	return hueToRGB(p, q, h+1.0/3.0), hueToRGB(p, q, h), hueToRGB(p, q, h-1.0/3.0)
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
