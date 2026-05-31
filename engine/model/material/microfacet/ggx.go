package microfacet

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"math"
)

const minAlpha = 1e-4

type GGX struct {
	Alpha float64
}

func NewGGX(alpha float64) GGX {
	return GGX{Alpha: ClampAlpha(alpha)}
}

func ClampAlpha(alpha float64) float64 {
	if alpha < minAlpha {
		return minAlpha
	}
	if alpha > 1 {
		return 1
	}
	return alpha
}

func (g GGX) D(wh maths.Direction) float64 {
	if maths.CosTheta(wh) <= 0 {
		return 0
	}

	alpha := ClampAlpha(g.Alpha)
	a2 := alpha * alpha
	cos2 := maths.CosTheta(wh) * maths.CosTheta(wh)
	denom := cos2*(a2-1) + 1
	return a2 / (math.Pi * denom * denom)
}

func (g GGX) Lambda(w maths.Direction) float64 {
	absCos := maths.AbsCosTheta(w)
	if absCos == 0 {
		return math.Inf(1)
	}

	alpha := ClampAlpha(g.Alpha)
	sin2 := math.Max(0, 1-absCos*absCos)
	tan2 := sin2 / (absCos * absCos)
	if math.IsInf(tan2, 0) {
		return 0
	}

	return (math.Sqrt(1+alpha*alpha*tan2) - 1) * 0.5
}

func (g GGX) G1(w maths.Direction) float64 {
	return 1 / (1 + g.Lambda(w))
}

func (g GGX) G(wi, wo maths.Direction) float64 {
	return 1 / (1 + g.Lambda(wi) + g.Lambda(wo))
}

func (g GGX) SampleVisibleNormal(wo maths.Direction, u maths.Sample2D) maths.Direction {
	if maths.CosTheta(wo) <= 0 {
		return maths.NewDirection(0, 0, 1)
	}

	alpha := ClampAlpha(g.Alpha)
	vh := maths.NewDirection(alpha*wo.Component(0), alpha*wo.Component(1), maths.CosTheta(wo)).Normalize()

	var t1 maths.Direction
	lensq := vh.Component(0)*vh.Component(0) + vh.Component(1)*vh.Component(1)
	if lensq > 0 {
		invLen := 1 / math.Sqrt(lensq)
		t1 = maths.NewDirection(-vh.Component(1)*invLen, vh.Component(0)*invLen, 0)
	} else {
		t1 = maths.NewDirection(1, 0, 0)
	}
	t2 := cross(vh, t1)

	r := math.Sqrt(clamp01(u.U))
	phi := 2 * math.Pi * clamp01(u.V)
	t1v := r * math.Cos(phi)
	t2v := r * math.Sin(phi)
	s := 0.5 * (1 + maths.CosTheta(vh))
	t2v = (1-s)*math.Sqrt(math.Max(0, 1-t1v*t1v)) + s*t2v

	nh := t1.MulScalar(t1v).
		Add(t2.MulScalar(t2v)).
		Add(vh.MulScalar(math.Sqrt(math.Max(0, 1-t1v*t1v-t2v*t2v))))

	wh := maths.NewDirection(alpha*nh.Component(0), alpha*nh.Component(1), math.Max(0, maths.CosTheta(nh))).Normalize()
	if maths.CosTheta(wh) <= 0 {
		return maths.NewDirection(0, 0, 1)
	}
	return wh
}

func (g GGX) PDFVisibleNormal(wo, wh maths.Direction) float64 {
	if maths.CosTheta(wo) <= 0 || maths.CosTheta(wh) <= 0 || wo.Dot(wh) <= 0 {
		return 0
	}
	return g.D(wh) * g.G1(wo) * math.Abs(wo.Dot(wh)) / maths.AbsCosTheta(wo)
}

func cross(a, b maths.Direction) maths.Direction {
	return maths.NewDirection(
		a.Component(1)*b.Component(2)-a.Component(2)*b.Component(1),
		a.Component(2)*b.Component(0)-a.Component(0)*b.Component(2),
		a.Component(0)*b.Component(1)-a.Component(1)*b.Component(0),
	)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
