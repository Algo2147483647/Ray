package bsdf

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

type CylindricalGridCutout struct {
	Line            bxdf.Scattering
	Origin          maths.Direction
	Axis            maths.Direction
	Reference       maths.Direction
	Bitangent       maths.Direction
	GapWidth        float64
	GapHeight       float64
	LineWidth       float64
	ReferenceRadius float64
}

func NewCylindricalGridCutout(
	line bxdf.Scattering,
	origin, axis, reference maths.Direction,
	gapWidth, gapHeight, lineWidth, referenceRadius float64,
) CylindricalGridCutout {
	axis = normalizeOr(axis, maths.NewDirection(0, 0, 1))
	reference = orthonormalReference(axis, reference)
	if gapWidth < 0 {
		gapWidth = 0
	}
	if gapHeight < 0 {
		gapHeight = 0
	}
	if lineWidth < 0 {
		lineWidth = 0
	}
	if referenceRadius <= 0 {
		referenceRadius = 1
	}
	return CylindricalGridCutout{
		Line:            line,
		Origin:          cloneDirection(origin, 3),
		Axis:            axis,
		Reference:       reference,
		Bitangent:       cross3(axis, reference).Normalize(),
		GapWidth:        gapWidth,
		GapHeight:       gapHeight,
		LineWidth:       lineWidth,
		ReferenceRadius: referenceRadius,
	}
}

func (c CylindricalGridCutout) Eval(ctx bxdf.ShadingContext, wi, wo maths.Direction) optics.Spectrum {
	if !c.OnGridLine(ctx) || c.Line == nil {
		return optics.Spectrum{}
	}
	return c.Line.Eval(ctx, wi, wo)
}

func (c CylindricalGridCutout) Sample(ctx bxdf.ShadingContext, wo maths.Direction, u maths.Sample2D) bxdf.BxDFSample {
	if c.OnGridLine(ctx) && c.Line != nil {
		return c.Line.Sample(ctx, wo, u)
	}

	wi := wo.MulScalar(-1)
	cos := maths.AbsCosTheta(wi)
	if cos <= 0 {
		return bxdf.BxDFSample{}
	}
	return bxdf.BxDFSample{
		Wi:    wi,
		F:     optics.ConstantSpectrum(1).DivScalar(cos),
		PDF:   1,
		Flags: bxdf.DeltaTransmission,
	}
}

func (c CylindricalGridCutout) PDF(ctx bxdf.ShadingContext, wi, wo maths.Direction) float64 {
	if !c.OnGridLine(ctx) || c.Line == nil {
		return 0
	}
	return c.Line.PDF(ctx, wi, wo)
}

func (c CylindricalGridCutout) AlbedoBound(ctx bxdf.ShadingContext) optics.Spectrum {
	return optics.ConstantSpectrum(1)
}

func (c CylindricalGridCutout) RoughnessInfo(ctx bxdf.ShadingContext) bxdf.RoughnessInfo {
	if c.OnGridLine(ctx) && c.Line != nil {
		return c.Line.RoughnessInfo(ctx)
	}
	return bxdf.RoughnessInfo{IsDelta: true}
}

func (c CylindricalGridCutout) DeltaFlags() bxdf.DeltaFlags {
	flags := bxdf.DeltaTransmission
	if c.Line != nil {
		flags |= c.Line.DeltaFlags()
	}
	return flags
}

func (c CylindricalGridCutout) OnGridLine(ctx bxdf.ShadingContext) bool {
	p := ctx.HitPoint
	if p.Len() < 3 {
		return true
	}

	rel := p.Add(c.Origin.MulScalar(-1))
	height := rel.Dot(c.Axis)
	inPlane := rel.Add(c.Axis.MulScalar(-height))
	radius := inPlane.Length()
	halfLine := c.LineWidth * 0.5

	if c.GapWidth >= 0 && c.LineWidth > 0 && radius > 0 {
		x := inPlane.Dot(c.Reference)
		y := inPlane.Dot(c.Bitangent)
		theta := math.Atan2(y, x)
		circumferential := theta * c.ReferenceRadius
		if periodicDistance(circumferential, c.GapWidth+c.LineWidth) <= halfLine {
			return true
		}
	}

	if c.GapHeight >= 0 && c.LineWidth > 0 {
		if periodicDistance(height, c.GapHeight+c.LineWidth) <= halfLine {
			return true
		}
	}

	return false
}

func periodicDistance(value, period float64) float64 {
	if period <= 0 {
		return math.Inf(1)
	}
	wrapped := math.Mod(value, period)
	if wrapped < 0 {
		wrapped += period
	}
	return math.Min(wrapped, period-wrapped)
}

func orthonormalReference(axis, reference maths.Direction) maths.Direction {
	if reference.Len() >= 3 {
		projected := reference.Add(axis.MulScalar(-reference.Dot(axis)))
		if projected.Length() > 0 {
			return projected.Normalize()
		}
	}
	fallback := maths.NewDirection(1, 0, 0)
	if math.Abs(axis.Dot(fallback)) > 0.9 {
		fallback = maths.NewDirection(0, 1, 0)
	}
	return fallback.Add(axis.MulScalar(-fallback.Dot(axis))).Normalize()
}

func normalizeOr(value, fallback maths.Direction) maths.Direction {
	if value.Len() >= 3 && value.IsFinite() && value.Length() > 0 {
		return value.Normalize()
	}
	return fallback.Normalize()
}

func cloneDirection(value maths.Direction, dimension int) maths.Direction {
	out := make(maths.Direction, dimension)
	for i := 0; i < dimension; i++ {
		out[i] = value.Component(i)
	}
	return out
}

func cross3(a, b maths.Direction) maths.Direction {
	return maths.NewDirection(
		a.Component(1)*b.Component(2)-a.Component(2)*b.Component(1),
		a.Component(2)*b.Component(0)-a.Component(0)*b.Component(2),
		a.Component(0)*b.Component(1)-a.Component(1)*b.Component(0),
	)
}
