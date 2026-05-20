package material

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils/maths"
	"math"
)

type Options struct {
	DirectionSamples int
	Tolerance        float64
}

func DefaultOptions() Options {
	return Options{
		DirectionSamples: 256,
		Tolerance:        1e-4,
	}
}

func CheckNonNegative(scattering bxdf.Scattering, ctx bxdf.ShadingContext, opts Options) error {
	opts = normalizeOptions(opts)
	wo := maths.NewDirection(0, 0, 1)

	for i := 0; i < opts.DirectionSamples; i++ {
		wi := maths.UniformHemisphereDirection(i, opts.DirectionSamples)
		f := scattering.Eval(ctx, wi, wo)
		pdf := scattering.PDF(ctx, wi, wo)
		if !f.IsFinite() || !f.IsNonNegative() {
			return fmt.Errorf("eval must be finite and non-negative for sample %d: %+v", i, f)
		}
		if !isFinite(pdf) || pdf < 0 {
			return fmt.Errorf("pdf must be finite and non-negative for sample %d: %f", i, pdf)
		}
	}

	return nil
}

func CheckReciprocity(scattering bxdf.Scattering, ctx bxdf.ShadingContext, opts Options) error {
	if scattering.DeltaFlags()&bxdf.NonReciprocal != 0 {
		return nil
	}

	opts = normalizeOptions(opts)
	for i := 0; i < opts.DirectionSamples; i++ {
		wi := maths.UniformHemisphereDirection(i, opts.DirectionSamples)
		wo := maths.UniformHemisphereDirection(opts.DirectionSamples-1-i, opts.DirectionSamples)
		fForward := scattering.Eval(ctx, wi, wo)
		fReverse := scattering.Eval(ctx, wo, wi)
		if !fForward.AlmostEqual(fReverse, opts.Tolerance) {
			return fmt.Errorf("reciprocity failed for sample %d: f(wi,wo)=%+v f(wo,wi)=%+v", i, fForward, fReverse)
		}
	}

	return nil
}

func CheckEnergyConservation(scattering bxdf.Scattering, ctx bxdf.ShadingContext, opts Options) error {
	opts = normalizeOptions(opts)
	wo := maths.NewDirection(0, 0, 1)
	sum := optics.Spectrum{}

	for i := 0; i < opts.DirectionSamples; i++ {
		wi := maths.UniformHemisphereDirection(i, opts.DirectionSamples)
		f := scattering.Eval(ctx, wi, wo)
		weight := maths.AbsCosTheta(wi) * 2 * math.Pi / float64(opts.DirectionSamples)
		sum = sum.Add(f.MulScalar(weight))
	}

	if sum.MaxComponent() > 1+opts.Tolerance {
		return fmt.Errorf("energy conservation failed: reflected=%+v tolerance=%f", sum, opts.Tolerance)
	}

	return nil
}

func CheckSamplePDFConsistency(scattering bxdf.Scattering, ctx bxdf.ShadingContext, opts Options) error {
	opts = normalizeOptions(opts)
	wo := maths.NewDirection(0, 0, 1)

	for i := 0; i < opts.DirectionSamples; i++ {
		u := maths.Sample2D{
			U: (float64(i) + 0.5) / float64(opts.DirectionSamples),
			V: math.Mod(float64(i)*0.6180339887498949, 1),
		}
		sample := scattering.Sample(ctx, wo, u)
		if !sample.Wi.IsFinite() || !sample.F.IsFinite() || !isFinite(sample.PDF) {
			return fmt.Errorf("sample must be finite for sample %d: %+v", i, sample)
		}
		if sample.PDF < 0 {
			return fmt.Errorf("sample pdf must be non-negative for sample %d: %f", i, sample.PDF)
		}

		expectedPDF := scattering.PDF(ctx, sample.Wi, wo)
		if math.Abs(sample.PDF-expectedPDF) > opts.Tolerance {
			return fmt.Errorf("sample/pdf mismatch for sample %d: sample=%f pdf()=%f", i, sample.PDF, expectedPDF)
		}
		expectedF := scattering.Eval(ctx, sample.Wi, wo)
		if !sample.F.AlmostEqual(expectedF, opts.Tolerance) {
			return fmt.Errorf("sample/eval mismatch for sample %d: sample=%+v eval()=%+v", i, sample.F, expectedF)
		}
	}

	return nil
}

func CheckBasicPhysicalValidity(scattering bxdf.Scattering, ctx bxdf.ShadingContext, opts Options) error {
	checks := []struct {
		name string
		fn   func(bxdf.Scattering, bxdf.ShadingContext, Options) error
	}{
		{name: "non-negativity", fn: CheckNonNegative},
		{name: "reciprocity", fn: CheckReciprocity},
		{name: "energy conservation", fn: CheckEnergyConservation},
		{name: "sample/pdf consistency", fn: CheckSamplePDFConsistency},
	}

	for _, check := range checks {
		if err := check.fn(scattering, ctx, opts); err != nil {
			return fmt.Errorf("%s: %w", check.name, err)
		}
	}

	return nil
}

func normalizeOptions(opts Options) Options {
	if opts.DirectionSamples <= 0 {
		opts.DirectionSamples = DefaultOptions().DirectionSamples
	}
	if opts.Tolerance <= 0 {
		opts.Tolerance = DefaultOptions().Tolerance
	}
	return opts
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
