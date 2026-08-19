package emission

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
)

func TestCosinePowerPDFNormalization(t *testing.T) {
	for _, test := range []struct {
		name      string
		exponent  float64
		sidedness Sidedness
	}{
		{name: "front uniform", exponent: 0, sidedness: FrontSide},
		{name: "front narrow", exponent: 24, sidedness: FrontSide},
		{name: "back", exponent: 3.5, sidedness: BackSide},
		{name: "two sided", exponent: 8, sidedness: TwoSided},
	} {
		t.Run(test.name, func(t *testing.T) {
			distribution, err := NewCosinePower(test.exponent, test.sidedness)
			if err != nil {
				t.Fatal(err)
			}
			const steps = 200000
			integral := 0.0
			for i := 0; i < steps; i++ {
				z := -1 + 2*(float64(i)+0.5)/steps
				wo := maths.NewDirection(math.Sqrt(math.Max(0, 1-z*z)), 0, z)
				integral += distribution.PDF(wo) * 4 * math.Pi / steps
			}
			if math.Abs(integral-1) > 2e-5 {
				t.Fatalf("integral(pdf) = %.12f, want 1", integral)
			}
		})
	}
}

func TestCosinePowerSampleMatchesPDFAndSide(t *testing.T) {
	distribution, err := NewCosinePower(12, FrontSide)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1024; i++ {
		u := maths.Sample2D{
			U: (float64(i) + 0.5) / 1024,
			V: math.Mod(float64(i)*0.6180339887498949, 1),
		}
		sample := distribution.Sample(u, 3)
		if sample.Wo.Len() != 3 || sample.Wo.Component(2) <= 0 {
			t.Fatalf("sample left front hemisphere: %+v", sample.Wo)
		}
		want := distribution.PDF(sample.Wo)
		if math.Abs(sample.PDF-want) > 1e-12 {
			t.Fatalf("sample PDF %.12g != evaluated PDF %.12g", sample.PDF, want)
		}
	}
}

func TestTotalExitanceNormalization(t *testing.T) {
	distribution, err := NewCosinePower(18, TwoSided)
	if err != nil {
		t.Fatal(err)
	}
	emitter := NewSurfaceEmitter(
		Constant{Radiance: spectrum_parameter.NewConstantParameter(7)},
		distribution,
		TotalExitance,
	)
	ctx := bxdf.ShadingContext{GeometricNormal: maths.NewDirection(0, 0, 1)}
	if got := emitter.ExitanceEstimate(ctx).MaxComponent(); math.Abs(got-7) > 1e-12 {
		t.Fatalf("exitance estimate = %g, want 7", got)
	}

	const steps = 200000
	integral := 0.0
	for i := 0; i < steps; i++ {
		z := -1 + 2*(float64(i)+0.5)/steps
		wo := maths.NewDirection(math.Sqrt(math.Max(0, 1-z*z)), 0, z)
		integral += emitter.Eval(ctx, wo).MaxComponent() * math.Abs(z) * 4 * math.Pi / steps
	}
	if math.Abs(integral-7) > 2e-5 {
		t.Fatalf("integrated exitance = %.12f, want 7", integral)
	}
}

func TestLegacyConstantRemainsTwoSidedUniformRadiance(t *testing.T) {
	emitter := NewConstant(optics.ConstantSpectrum(2))
	ctx := bxdf.ShadingContext{GeometricNormal: maths.NewDirection(0, 0, 1)}
	for _, wo := range []maths.Direction{
		maths.NewDirection(0, 0, 1), maths.NewDirection(0, 0, -1),
	} {
		if got := emitter.Eval(ctx, wo).MaxComponent(); got != 2 {
			t.Fatalf("legacy radiance = %g, want 2", got)
		}
	}
	if got, want := emitter.ExitanceEstimate(ctx).MaxComponent(), 4*math.Pi; math.Abs(got-want) > 1e-12 {
		t.Fatalf("two-sided exitance = %g, want %g", got, want)
	}
}
