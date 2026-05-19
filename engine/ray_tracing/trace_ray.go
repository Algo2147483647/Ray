package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math"
	"math/rand/v2"

	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/model/material/core"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

func (h *Handler) TraceRay(objTree *object.ObjectTree, ray *renderray.Ray, level int64) *mat.VecDense {
	var (
		normal        = mat.NewVecDense(ray.Origin.Len(), nil)
		DebugRayTrace = map[string]interface{}{}
	)

	if utils.IsDebug {
		DebugRayTrace = map[string]interface{}{
			"start":      append([]float64(nil), ray.Origin.RawVector().Data...),
			"direction":  append([]float64(nil), ray.Direction.RawVector().Data...),
			"color":      append([]float64(nil), ray.Color.RawVector().Data...),
			"level":      level,
			"hit_object": "",
		}

		normal.AddVec(ray.Origin, math_lib.ScaleVec2(1, ray.Direction))
		DebugRayTrace["end"] = append([]float64(nil), normal.RawVector().Data...)
	}

	if level > h.MaxRayLevel {
		ray.Color.ScaleVec(0, ray.Color)
		return ray.Color
	}

	hit, ok := objTree.GetSurfaceHit(ray.Origin, ray.Direction)
	if !ok {
		return math_lib.ScaleVec(ray.Color, 0, ray.Color)
	}

	ray.Origin.CopyVec(hit.Point)
	normal = hit.ShadingNormal
	obj := hit.Object

	if utils.IsDebug {
		DebugRayTrace["hit_object"] = obj.Shape.Name()
		DebugRayTrace["end"] = append([]float64(nil), ray.Origin.RawVector().Data...)
		DebugRayTrace["distance"] = hit.Distance
		DebugRayTrace["front_face"] = hit.FrontFace
	}

	if obj.Material == nil {
		ray.Color.ScaleVec(0, ray.Color)
		return ray.Color
	}

	media := objTree.Media
	if media == nil {
		media = medium.NewRegistry()
	}

	ctx := core.ShadingContext{
		TransportMode: core.TransportRadiance,
		SpectrumMode:  h.SpectrumMode,
		CurrentIOR:    ray.RefractionIndex,
		WavelengthNM:  ray.WaveLength,
		WavelengthPDF: ray.WavelengthPDF,
	}

	if h.SpectrumMode == core.SpectrumRGBAndSpectral && ray.WaveLength > 0 {
		ctx.WavelengthsNM = []float64{ray.WaveLength}
	}
	prepareMediumContext(&ctx, media, ray, obj.MediumBoundary, hit.FrontFace)

	woWorld := negateVec(ray.Direction)
	woLocal, frameOK := worldToLocal(woWorld, normal)
	if !frameOK {
		ray.Color.ScaleVec(0, ray.Color)
		return ray.Color
	} else if obj.Material.HasEmission() {
		emitted := obj.Material.Emission.Emit(ctx, woLocal)
		applySpectrum(ray.Color, emitted)
		return ray.Color
	} else if !obj.Material.HasSurface() {
		ray.Color.ScaleVec(0, ray.Color)
		return ray.Color
	}

	sample := obj.Material.Surface.Sample(ctx, woLocal, core.Sample2D{
		U: rand.Float64(),
		V: rand.Float64(),
	})
	if sample.PDF <= 0 || !sample.F.IsFinite() || !sample.F.IsNonNegative() {
		ray.Color.ScaleVec(0, ray.Color)
		return ray.Color
	}

	weight := core.AbsCosTheta(sample.Wi) / sample.PDF
	applySpectrum(ray.Color, sample.F.MulScalar(weight))
	if sample.Flags&core.DeltaTransmission != 0 {
		applyMediumTransmission(media, ray, ctx, obj.MediumBoundary, sample)
	}

	ray.Direction = localToWorld(sample.Wi, normal)
	math_lib.Normalize(ray.Direction)

	return h.TraceRay(objTree, ray, level+1)
}

func prepareMediumContext(ctx *core.ShadingContext, media *medium.Registry, ray *renderray.Ray, boundary medium.Boundary, frontFace bool) {
	incident := ray.MediumStack.Current()
	transmit := incident
	entering := false

	ctx.IncidentMedium = incident
	ctx.TransmitMedium = transmit
	ctx.Entering = entering

	if !boundary.Active() {
		return
	}

	entering = frontFace
	if entering {
		transmit = boundary.Inside
	} else if boundary.Outside != core.MediumNone {
		transmit = boundary.Outside
	} else {
		transmit = core.MediumAir
	}

	ctx.IncidentMedium = incident
	ctx.TransmitMedium = transmit
	ctx.Entering = entering
	ctx.EtaIncident = media.IOR(incident, *ctx)
	ctx.EtaTransmit = media.IOR(transmit, *ctx)
	ctx.CurrentIOR = ctx.EtaIncident
	ray.RefractionIndex = ctx.EtaIncident
}

func applyMediumTransmission(media *medium.Registry, ray *renderray.Ray, ctx core.ShadingContext, boundary medium.Boundary, sample core.BxDFSample) {
	if boundary.Active() && sample.TransmitMedium != core.MediumNone {
		if ctx.Entering {
			ray.MediumStack.Push(sample.TransmitMedium)
		} else {
			ray.MediumStack.Remove(boundary.Inside)
		}
		ray.RefractionIndex = media.IOR(ray.MediumStack.Current(), ctx)
		return
	}

	if sample.Eta > 0 {
		ray.RefractionIndex = sample.Eta
	}
}

func applySpectrum(color *mat.VecDense, spectrum core.Spectrum) {
	color.SetVec(0, color.AtVec(0)*spectrum.R)
	color.SetVec(1, color.AtVec(1)*spectrum.G)
	color.SetVec(2, color.AtVec(2)*spectrum.B)
}

func negateVec(v *mat.VecDense) *mat.VecDense {
	res := mat.NewVecDense(v.Len(), nil)
	res.ScaleVec(-1, v)
	return res
}

func worldToLocal(v, normal *mat.VecDense) (core.Direction, bool) {
	tangent, bitangent, ok := tangentFrame(normal)
	if !ok {
		return core.Direction{}, false
	}
	return core.NewDirection(
		mat.Dot(v, tangent),
		mat.Dot(v, bitangent),
		mat.Dot(v, normal),
	), true
}

func localToWorld(v core.Direction, normal *mat.VecDense) *mat.VecDense {
	tangent, bitangent, ok := tangentFrame(normal)
	if !ok {
		return mat.NewVecDense(normal.Len(), nil)
	}

	res := mat.NewVecDense(3, nil)
	res.AddScaledVec(res, v.X, tangent)
	res.AddScaledVec(res, v.Y, bitangent)
	res.AddScaledVec(res, v.Z, normal)
	return res
}

func tangentFrame(normal *mat.VecDense) (*mat.VecDense, *mat.VecDense, bool) {
	if normal.Len() != 3 {
		return nil, nil, false
	}

	n := mat.VecDenseCopyOf(normal)
	math_lib.Normalize(n)

	var tangent *mat.VecDense
	if math.Abs(n.AtVec(2)) < 0.999999 {
		tangent = mat.NewVecDense(3, []float64{-n.AtVec(1), n.AtVec(0), 0})
	} else {
		tangent = mat.NewVecDense(3, []float64{0, 1, 0})
	}
	math_lib.Normalize(tangent)

	bitangent := math_lib.Cross2(n, tangent)
	math_lib.Normalize(bitangent)
	return tangent, bitangent, true
}
