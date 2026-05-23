package ray_tracing

import (
	"math/rand/v2"

	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils/maths"
)

func (h *Handler) TraceRay(objTree *object.ObjectTree, ray *optics.Ray, level int64) {
	// Stop tracing when the recursion depth exceeds the configured limit.
	if level > h.MaxRayLevel {
		terminateRay(ray)
		return
	}

	// Find the closest surface intersection along the current ray.
	hit, ok := objTree.GetSurfaceHit(ray.Origin, ray.Direction)
	if !ok {
		terminateRay(ray)
		return
	}

	// Apply medium absorption accumulated along the segment before the hit point.
	media := getMediumRegistry(objTree)
	mediumCtx := h.newShadingContext(ray)
	applyMediumAbsorption(media, ray, hit.Distance, mediumCtx)

	// Move the ray origin to the hit point for the next bounce.
	ray.Origin.CopyVec(hit.Point)

	// Fetch the hit object and validate its material.
	obj := hit.Object
	if obj.Material == nil {
		terminateRay(ray)
		return
	}

	// Build the shading context for surface evaluation at this interaction.
	ctx := h.newShadingContext(ray)
	ctx.TransportMode = bxdf.TransportRadiance
	ctx.CurrentIOR = ray.RefractionIndex

	// Update medium-related state according to the material boundary.
	prepareMediumContext(&ctx, media, ray, obj.MediumBoundary, hit.FrontFace)

	// Construct a local shading frame from the hit normal.
	frame, ok := maths.NewFrameFromNormal(hit.ShadingNormal)
	if !ok {
		terminateRay(ray)
		return
	}

	// Convert the outgoing direction into local shading coordinates.
	woLocal := frame.WorldToLocalNegated(ray.Direction)

	// Handle emissive surfaces directly; terminate if there is no BSDF to sample.
	if h.traceEmission(ray, obj, ctx, woLocal) {
		return
	} else if !obj.Material.HasSurface() {
		terminateRay(ray)
		return
	}

	// Sample the surface BSDF to choose the next path direction.
	sample, ok := sampleSurface(obj, ctx, woLocal)
	if !ok {
		terminateRay(ray)
		return
	}

	// Apply the BSDF weight, spectral update, and medium transmission if needed.
	applySurfaceSample(media, ray, ctx, obj, sample)

	// Transform the sampled local direction back to world space.
	frame.LocalToWorldInto(ray.Direction, sample.Wi)
	math_lib.Normalize(ray.Direction)

	// Probabilistically terminate low-contribution paths after enough bounces.
	if h.killByRussianRoulette(ray, level+1) {
		terminateRay(ray)
		return
	}

	// Continue tracing the next bounce.
	h.TraceRay(objTree, ray, level+1)
}

func getMediumRegistry(objTree *object.ObjectTree) *medium.Registry {
	if objTree.Media != nil {
		return objTree.Media
	}
	return medium.NewRegistry()
}

func (h *Handler) newShadingContext(ray *optics.Ray) bxdf.ShadingContext {
	ctx := bxdf.ShadingContext{
		SpectrumMode:  h.SpectrumMode,
		WavelengthNM:  ray.WaveLength,
		WavelengthPDF: ray.WavelengthPDF,
	}

	if h.SpectrumMode != optics.SpectrumModeRGB && ray.WaveLength > 0 {
		ctx.WavelengthsNM = []float64{ray.WaveLength}
	}

	return ctx
}

func (h *Handler) traceEmission(
	ray *optics.Ray,
	obj *object.Object,
	ctx bxdf.ShadingContext,
	woLocal maths.Direction,
) bool {
	if !obj.Material.HasEmission() {
		return false
	}

	emitted := obj.Material.Emission.Emit(ctx, woLocal)
	applySpectrum(ray, emitted)
	return true
}

func sampleSurface(
	obj *object.Object,
	ctx bxdf.ShadingContext,
	woLocal maths.Direction,
) (bxdf.BxDFSample, bool) {
	sample := obj.Material.Surface.Sample(ctx, woLocal, maths.Sample2D{
		U: rand.Float64(),
		V: rand.Float64(),
	})

	if sample.PDF <= 0 {
		return sample, false
	} else if !sample.F.IsFinite() {
		return sample, false
	} else if !sample.F.IsNonNegative() {
		return sample, false
	}

	return sample, true
}

func applySurfaceSample(
	media *medium.Registry,
	ray *optics.Ray,
	ctx bxdf.ShadingContext,
	obj *object.Object,
	sample bxdf.BxDFSample,
) {
	weight := maths.AbsCosTheta(sample.Wi) / sample.PDF
	applySpectrum(ray, sample.F.MulScalar(weight))

	if sample.WavelengthNM > 0 {
		ray.SpectralPath = true
	}

	if sample.Flags&bxdf.DeltaTransmission != 0 {
		applyMediumTransmission(media, ray, ctx, obj.MediumBoundary, sample)
	}
}

func (h *Handler) killByRussianRoulette(ray *optics.Ray, nextLevel int64) bool {
	if !h.shouldApplyRussianRoulette(nextLevel) {
		return false
	}

	survival := russianRouletteSurvivalProbability(ray)
	if survival <= 0 || rand.Float64() >= survival {
		return true
	}

	scaleRayThroughput(ray, 1/survival)
	return false
}

func (h *Handler) shouldApplyRussianRoulette(nextLevel int64) bool {
	depth := h.RussianRouletteDepth
	if depth <= 0 {
		depth = 3
	}
	return nextLevel >= depth
}
