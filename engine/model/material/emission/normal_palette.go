package emission

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

// NormalPalette is a geometry-independent procedural field. It selects a
// palette entry from the dominant signed component of the geometric normal.
type NormalPalette struct {
	Palette []optics.Spectrum
}

var DefaultNormalPalette = []optics.Spectrum{
	optics.NewSpectrum(1.00, 0.20, 0.20),
	optics.NewSpectrum(0.20, 1.00, 0.20),
	optics.NewSpectrum(0.20, 0.40, 1.00),
	optics.NewSpectrum(1.00, 0.85, 0.20),
	optics.NewSpectrum(1.00, 0.30, 0.90),
	optics.NewSpectrum(0.20, 0.95, 0.95),
	optics.NewSpectrum(1.00, 0.55, 0.10),
	optics.NewSpectrum(0.92, 0.92, 0.92),
}

func NewNormalPalette() NormalPalette {
	return NormalPalette{Palette: append([]optics.Spectrum(nil), DefaultNormalPalette...)}
}

func (n NormalPalette) EvaluateRadiance(ctx bxdf.ShadingContext) optics.Spectrum {
	palette := n.Palette
	if len(palette) == 0 {
		palette = DefaultNormalPalette
	}
	axis, sign := dominantNormalAxis(ctx.GeometricNormal)
	if axis < 0 {
		return palette[0]
	}
	index := axis * 2
	if sign > 0 {
		index++
	}
	return palette[index%len(palette)]
}

func (n NormalPalette) Eval(ctx bxdf.ShadingContext, wo maths.Direction) optics.Spectrum {
	return n.defaultEmitter().Eval(ctx, wo)
}
func (n NormalPalette) SampleDirection(ctx bxdf.ShadingContext, u maths.Sample2D) DirectionSample {
	return n.defaultEmitter().SampleDirection(ctx, u)
}
func (n NormalPalette) PDFDirection(ctx bxdf.ShadingContext, wo maths.Direction) float64 {
	return n.defaultEmitter().PDFDirection(ctx, wo)
}
func (n NormalPalette) ExitanceEstimate(ctx bxdf.ShadingContext) optics.Spectrum {
	return n.defaultEmitter().ExitanceEstimate(ctx)
}
func (NormalPalette) DirectionFlags() DirectionFlags { return DirectionContinuous }
func (n NormalPalette) Emit(ctx bxdf.ShadingContext, wo maths.Direction) optics.Spectrum {
	return n.Eval(ctx, wo)
}
func (NormalPalette) IsDelta() bool { return false }
func (n NormalPalette) defaultEmitter() SurfaceEmitter {
	return NewSurfaceEmitter(n, NewUniform(TwoSided), PeakRadiance)
}

func dominantNormalAxis(normal maths.Direction) (int, float64) {
	bestAxis, bestMagnitude := -1, 0.0
	for axis := 0; axis < normal.Len(); axis++ {
		magnitude := math.Abs(normal.Component(axis))
		if magnitude > bestMagnitude {
			bestAxis, bestMagnitude = axis, magnitude
		}
	}
	if bestAxis < 0 || bestMagnitude == 0 {
		return -1, 0
	}
	return bestAxis, normal.Component(bestAxis)
}
