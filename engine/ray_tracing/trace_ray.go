package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils"
	"math"
	"math/rand/v2"
)

type SurfaceInteraction struct {
	Hit     *object.SurfaceHit
	Object  *object.Object
	Frame   maths.Frame
	WoLocal maths.Direction
	Context bxdf.ShadingContext
}

func (h *Handler) TraceRay(objTree *object.ObjectTree, ray *optics.Ray, level int64) {
	if h.terminateBeforeBounce(ray, level) {
		return
	}

	g := ray.G()

	// Find the closest surface intersection along the current ray.
	hit, ok := surfaceHitInGeometry(objTree, ray, g)
	if !ok {
		// Spherical: the current half-great-circle reached the antipode without
		// hitting anything; wrap and continue tracing if we still have arc budget.
		if newO, newD, wrapped := g.WrapBeyond(ray.Origin, ray.Direction, math.Pi); wrapped {
			advance := math.Pi
			if h.MaxArc > 0 && ray.ArcTraveled+advance > h.MaxArc {
				terminateRay(ray)
				return
			}
			ray.Origin.CopyVec(newO)
			ray.Direction.CopyVec(newD)
			ray.ArcTraveled += advance
			h.TraceRay(objTree, ray, level+1)
			return
		}
		terminateRay(ray)
		return
	}

	// Translate the embedded-domain ray parameter into geodesic arc length
	// before doing anything physical with it (absorption, direction update,
	// arc-budget bookkeeping).
	arcLen := hit.ArcLength
	if arcLen <= 0 {
		arcLen = g.ArcLengthFromEmbedT(ray.Origin, ray.Direction, hit.Distance)
	}

	// Apply medium absorption accumulated along the segment before the hit point.
	media := getMediumRegistry(objTree)
	mediumCtx := h.newShadingContext(ray)
	applyMediumAbsorption(media, ray, arcLen, mediumCtx)

	// Track total geodesic distance traveled (used by the S^3 wrap loop and
	// as a fail-safe even on flat geometries).
	ray.ArcTraveled += arcLen

	// Geodesic-budget kill (used primarily by S^3 to bound the wrap loop).
	if h.MaxArc > 0 && ray.ArcTraveled >= h.MaxArc {
		terminateRay(ray)
		return
	}

	if g.Name() == "spherical" {
		if newO, newD, wrapped := g.WrapBeyond(ray.Origin, ray.Direction, arcLen); wrapped {
			hit.Point = newO
			ray.Direction.CopyVec(newD)
		}
	}

	// Prepare all surface-local interaction data for this hit.
	si, ok := h.prepareSurfaceInteraction(media, ray, hit)
	if !ok {
		terminateRay(ray)
		return
	}

	// Handle emissive surfaces directly; terminate if there is no BSDF to sample.
	if h.traceEmission(ray, si.Object, si.Context, si.WoLocal) {
		return
	} else if !si.Object.Material.HasSurface() {
		terminateRay(ray)
		return
	}

	// Sample the surface BSDF to choose the next path direction.
	sample, ok := sampleSurface(si.Object, si.Context, si.WoLocal)
	if !ok {
		terminateRay(ray)
		return
	}

	// Apply the BSDF weight, spectral update, and medium transmission if needed.
	applySurfaceSample(media, ray, si.Context, si.Object, sample)

	// Transform the sampled local direction back to world space.
	si.Frame.LocalToWorldInto(ray.Direction, sample.Wi)

	// Project the sampled outgoing direction back into T_p of the current
	// geometry, then renormalize using the geometry's inner product so the
	// next embedded-ray intersection is parameterized correctly.
	ray.G().ProjectTangent(ray.Origin, ray.Direction, ray.Direction)
	maths.Normalize(ray.Direction)

	// Continue tracing the next bounce.
	h.TraceRay(objTree, ray, level+1)
}

func surfaceHitInGeometry(objTree *object.ObjectTree, ray *optics.Ray, g geometry.Geometry) (*object.SurfaceHit, bool) {
	if g.Name() == "spherical" {
		return objTree.GetSphericalSurfaceHit(ray.Origin, ray.Direction, utils.EPS, math.Pi)
	}
	embeddedOrigin, embeddedDirection, tMax := g.EmbeddedRay(ray.Origin, ray.Direction)
	if tMax <= 0 {
		return nil, false
	}
	return objTree.GetSurfaceHitRange(embeddedOrigin, embeddedDirection, utils.EPS, tMax)
}

func (h *Handler) terminateBeforeBounce(ray *optics.Ray, level int64) bool {
	if level > h.MaxRayLevel {
		terminateRay(ray)
		return true
	}

	if h.killByRussianRoulette(ray, level) {
		terminateRay(ray)
		return true
	}

	return false
}

func (h *Handler) prepareSurfaceInteraction(
	media *medium.Registry,
	ray *optics.Ray,
	hit *object.SurfaceHit,
) (SurfaceInteraction, bool) {
	// Move the ray origin to the hit point for the next bounce.
	ray.Origin.CopyVec(hit.Point)

	obj := hit.Object
	if obj == nil || obj.Material == nil {
		return SurfaceInteraction{}, false
	}

	ctx := h.newShadingContext(ray)
	ctx.TransportMode = bxdf.TransportRadiance
	ctx.CurrentIOR = ray.RefractionIndex

	if hit.GeometricNormal != nil {
		ctx.GeometricNormal = maths.NewDirectionFromComponents(hit.GeometricNormal.RawVector().Data)
	}
	if hit.Point != nil {
		ctx.HitPoint = maths.NewDirectionFromComponents(hit.Point.RawVector().Data)
	}
	if obj.Shape != nil {
		pmin, pmax := obj.Shape.BuildBoundingBox()
		if pmin != nil && pmax != nil {
			ctx.HitObjectAABBMin = maths.NewDirectionFromComponents(pmin.RawVector().Data)
			ctx.HitObjectAABBMax = maths.NewDirectionFromComponents(pmax.RawVector().Data)
		}
	}

	prepareMediumContext(&ctx, media, ray, obj.MediumBoundary, hit.FrontFace)

	frame, ok := maths.NewFrameFromNormalInGeometry(ray.G(), hit.Point, hit.ShadingNormal)
	if !ok {
		return SurfaceInteraction{}, false
	}

	woLocal := frame.WorldToLocalNegated(ray.Direction)

	return SurfaceInteraction{
		Hit:     hit,
		Object:  obj,
		Frame:   frame,
		WoLocal: woLocal,
		Context: ctx,
	}, true
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

	if sample.Flags&bxdf.TransmissionEvent != 0 {
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
