package microfacet

import (
	"math"

	"github.com/Algo2147483647/ray/engine/material/core"
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

func (g GGX) D(wh core.Direction) float64 {
	if wh.Z <= 0 {
		return 0
	}

	alpha := ClampAlpha(g.Alpha)
	a2 := alpha * alpha
	cos2 := wh.Z * wh.Z
	denom := cos2*(a2-1) + 1
	return a2 / (math.Pi * denom * denom)
}

func (g GGX) Lambda(w core.Direction) float64 {
	absCos := core.AbsCosTheta(w)
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

func (g GGX) G1(w core.Direction) float64 {
	return 1 / (1 + g.Lambda(w))
}

func (g GGX) G(wi, wo core.Direction) float64 {
	return 1 / (1 + g.Lambda(wi) + g.Lambda(wo))
}

func (g GGX) SampleVisibleNormal(wo core.Direction, u core.Sample2D) core.Direction {
	if wo.Z <= 0 {
		return core.Direction{Z: 1}
	}

	alpha := ClampAlpha(g.Alpha)
	vh := core.NewDirection(alpha*wo.X, alpha*wo.Y, wo.Z).Normalize()

	var t1 core.Direction
	lensq := vh.X*vh.X + vh.Y*vh.Y
	if lensq > 0 {
		invLen := 1 / math.Sqrt(lensq)
		t1 = core.NewDirection(-vh.Y*invLen, vh.X*invLen, 0)
	} else {
		t1 = core.NewDirection(1, 0, 0)
	}
	t2 := cross(vh, t1)

	r := math.Sqrt(clamp01(u.U))
	phi := 2 * math.Pi * clamp01(u.V)
	t1v := r * math.Cos(phi)
	t2v := r * math.Sin(phi)
	s := 0.5 * (1 + vh.Z)
	t2v = (1-s)*math.Sqrt(math.Max(0, 1-t1v*t1v)) + s*t2v

	nh := t1.MulScalar(t1v).
		Add(t2.MulScalar(t2v)).
		Add(vh.MulScalar(math.Sqrt(math.Max(0, 1-t1v*t1v-t2v*t2v))))

	wh := core.NewDirection(alpha*nh.X, alpha*nh.Y, math.Max(0, nh.Z)).Normalize()
	if wh.Z <= 0 {
		return core.Direction{Z: 1}
	}
	return wh
}

func (g GGX) PDFVisibleNormal(wo, wh core.Direction) float64 {
	if wo.Z <= 0 || wh.Z <= 0 || wo.Dot(wh) <= 0 {
		return 0
	}
	return g.D(wh) * g.G1(wo) * math.Abs(wo.Dot(wh)) / core.AbsCosTheta(wo)
}

func cross(a, b core.Direction) core.Direction {
	return core.NewDirection(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
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
