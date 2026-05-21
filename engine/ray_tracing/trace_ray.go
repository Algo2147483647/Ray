package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math/rand/v2"

	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/utils/maths"
	"gonum.org/v1/gonum/mat"
)

func (h *Handler) TraceRay(objTree *object.ObjectTree, ray *renderray.Ray, level int64) *mat.VecDense {
	if level > h.MaxRayLevel {
		return terminateRay(ray)
	}

	hit, ok := objTree.GetSurfaceHit(ray.Origin, ray.Direction)
	if !ok {
		return terminateRay(ray)
	}

	media := objTree.Media
	if media == nil {
		media = medium.NewRegistry()
	}
	mediumCtx := bxdf.ShadingContext{
		SpectrumMode:  h.SpectrumMode,
		WavelengthNM:  ray.WaveLength,
		WavelengthPDF: ray.WavelengthPDF,
	}
	if h.SpectrumMode != bxdf.SpectrumRGB && ray.WaveLength > 0 {
		mediumCtx.WavelengthsNM = []float64{ray.WaveLength}
	}
	applyMediumAbsorption(media, ray, hit.Distance, mediumCtx)

	ray.Origin.CopyVec(hit.Point)
	normal := hit.ShadingNormal
	obj := hit.Object

	if obj.Material == nil {
		return terminateRay(ray)
	}

	ctx := bxdf.ShadingContext{
		TransportMode: bxdf.TransportRadiance,
		SpectrumMode:  h.SpectrumMode,
		CurrentIOR:    ray.RefractionIndex,
		WavelengthNM:  ray.WaveLength,
		WavelengthPDF: ray.WavelengthPDF,
	}

	if h.SpectrumMode != bxdf.SpectrumRGB && ray.WaveLength > 0 {
		ctx.WavelengthsNM = []float64{ray.WaveLength}
	}
	prepareMediumContext(&ctx, media, ray, obj.MediumBoundary, hit.FrontFace)

	woLocal, frameOK := worldToLocalNegated(ray.Direction, normal)
	if !frameOK {
		return terminateRay(ray)
	} else if obj.Material.HasEmission() {
		emitted := obj.Material.Emission.Emit(ctx, woLocal)
		applySpectrum(ray, emitted)
		return ray.Color
	} else if !obj.Material.HasSurface() {
		return terminateRay(ray)
	}

	sample := obj.Material.Surface.Sample(ctx, woLocal, maths.Sample2D{
		U: rand.Float64(),
		V: rand.Float64(),
	})
	if sample.PDF <= 0 || !sample.F.IsFinite() || !sample.F.IsNonNegative() {
		return terminateRay(ray)
	}

	weight := maths.AbsCosTheta(sample.Wi) / sample.PDF
	applySpectrum(ray, sample.F.MulScalar(weight))
	if sample.WavelengthNM > 0 {
		ray.SpectralPath = true
	}
	if sample.Flags&bxdf.DeltaTransmission != 0 {
		applyMediumTransmission(media, ray, ctx, obj.MediumBoundary, sample)
	}

	if !localToWorldInto(ray.Direction, sample.Wi, normal) {
		return terminateRay(ray)
	}
	math_lib.Normalize(ray.Direction)

	return h.TraceRay(objTree, ray, level+1)
}
