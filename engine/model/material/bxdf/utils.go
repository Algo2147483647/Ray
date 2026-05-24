package bxdf

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"math"
)

func reflectLocal(wo maths.Direction) maths.Direction {
	return maths.NewDirection(-wo.X, -wo.Y, wo.Z)
}

func reflectAbout(wo, wh maths.Direction) maths.Direction {
	return wh.MulScalar(2 * wo.Dot(wh)).Add(wo.MulScalar(-1))
}

func refractLocal(wo maths.Direction, eta float64) (maths.Direction, bool) {
	cosThetaO := wo.Z
	sin2ThetaO := math.Max(0, 1-cosThetaO*cosThetaO)
	sin2ThetaI := eta * eta * sin2ThetaO
	if sin2ThetaI >= 1 {
		return maths.Direction{}, false
	}

	cosThetaI := math.Sqrt(math.Max(0, 1-sin2ThetaI))
	if cosThetaO > 0 {
		cosThetaI = -cosThetaI
	}

	return maths.NewDirection(-eta*wo.X, -eta*wo.Y, cosThetaI).Normalize(), true
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func almostEqualIOR(a, b float64) bool {
	return math.Abs(a-b) <= 1e-6
}

func FresnelDielectric(cosThetaI, etaI, etaT float64) float64 {
	cosThetaI = clamp(cosThetaI, -1, 1)
	entering := cosThetaI > 0
	if !entering {
		etaI, etaT = etaT, etaI
		cosThetaI = math.Abs(cosThetaI)
	}

	sinThetaI := math.Sqrt(math.Max(0, 1-cosThetaI*cosThetaI))
	sinThetaT := etaI / etaT * sinThetaI
	if sinThetaT >= 1 {
		return 1
	}

	cosThetaT := math.Sqrt(math.Max(0, 1-sinThetaT*sinThetaT))
	rParallel := ((etaT * cosThetaI) - (etaI * cosThetaT)) / ((etaT * cosThetaI) + (etaI * cosThetaT))
	rPerpendicular := ((etaI * cosThetaI) - (etaT * cosThetaT)) / ((etaI * cosThetaI) + (etaT * cosThetaT))
	return (rParallel*rParallel + rPerpendicular*rPerpendicular) * 0.5
}

func compatibleWeightSpectrum(weight, target optics.Spectrum, ctx ShadingContext) optics.Spectrum {
	if weight.HasSamples() == target.HasSamples() {
		return weight
	}
	if target.HasSamples() && !weight.HasSamples() {
		return weight.UpliftRGBReflectanceToSampled(ctx.WavelengthsNM)
	}
	if !target.HasSamples() && sampledSpectrumIsConstant(weight) {
		return optics.ConstantSpectrum(weight.Sample(0))
	}
	return optics.Spectrum{}
}

func sampledSpectrumIsConstant(s optics.Spectrum) bool {
	if !s.HasSamples() {
		return false
	}
	first := s.Sample(0)
	for i := 1; i < s.SampleCount(); i++ {
		if math.Abs(s.Sample(i)-first) > 1e-12 {
			return false
		}
	}
	return true
}
