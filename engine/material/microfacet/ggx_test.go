package microfacet

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/material/core"
)

func TestGGXDAtNormal(t *testing.T) {
	ggx := NewGGX(0.5)
	got := ggx.D(core.NewDirection(0, 0, 1))
	want := 1 / (math.Pi * 0.5 * 0.5)
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("unexpected D at normal: got %f want %f", got, want)
	}
}

func TestGGXVisibleNormalSampleIsFiniteAndUpperHemisphere(t *testing.T) {
	ggx := NewGGX(0.35)
	wo := core.NewDirection(0.2, -0.1, math.Sqrt(0.95)).Normalize()

	for i := 0; i < 64; i++ {
		u := core.Sample2D{
			U: (float64(i) + 0.5) / 64,
			V: math.Mod(float64(i)*0.6180339887498949, 1),
		}
		wh := ggx.SampleVisibleNormal(wo, u)
		if !wh.IsFinite() || wh.Z <= 0 {
			t.Fatalf("invalid visible normal sample %d: %+v", i, wh)
		}
		if pdf := ggx.PDFVisibleNormal(wo, wh); pdf <= 0 || math.IsNaN(pdf) || math.IsInf(pdf, 0) {
			t.Fatalf("invalid visible normal pdf %d: %f wh=%+v", i, pdf, wh)
		}
	}
}

func TestFresnelConductorRange(t *testing.T) {
	f := FresnelConductor(0.5, core.NewSpectrum(0.2, 0.9, 1.5), core.NewSpectrum(3.9, 2.5, 1.9))
	if !f.IsFinite() || !f.IsNonNegative() || f.R > 1 || f.G > 1 || f.B > 1 {
		t.Fatalf("conductor fresnel should be finite and in [0,1], got %+v", f)
	}
}
