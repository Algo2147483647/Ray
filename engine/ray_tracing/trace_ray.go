package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math"
	"math/rand/v2"

	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/model/material/core"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
	"gonum.org/v1/gonum/mat"
)

func (h *Handler) TraceRay(objTree *object.ObjectTree, ray *renderray.Ray, level int64) *mat.VecDense {
	if level > h.MaxRayLevel {
		ray.Color.ScaleVec(0, ray.Color)
		return ray.Color
	}

	hit, ok := objTree.GetSurfaceHit(ray.Origin, ray.Direction)
	if !ok {
		return math_lib.ScaleVec(ray.Color, 0, ray.Color)
	}

	ray.Origin.CopyVec(hit.Point)
	normal := hit.ShadingNormal
	obj := hit.Object

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

	if h.SpectrumMode != core.SpectrumRGB && ray.WaveLength > 0 {
		ctx.WavelengthsNM = []float64{ray.WaveLength}
	}
	prepareMediumContext(&ctx, media, ray, obj.MediumBoundary, hit.FrontFace)

	woLocal, frameOK := worldToLocalNegated(ray.Direction, normal)
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

	if !localToWorldInto(ray.Direction, sample.Wi, normal) {
		ray.Color.ScaleVec(0, ray.Color)
		return ray.Color
	}
	math_lib.Normalize(ray.Direction)

	return h.TraceRay(objTree, ray, level+1)
}

func prepareMediumContext(ctx *core.ShadingContext, media *medium.Registry, ray *renderray.Ray, boundary medium.Boundary, frontFace bool) {
	transition := ray.MediumStack.ResolveTransition(boundary, frontFace)
	incident := transition.Incident
	transmit := transition.Transmit
	entering := transition.Entering

	ctx.IncidentMedium = incident
	ctx.TransmitMedium = transmit
	ctx.Entering = entering

	if !boundary.Active() {
		return
	}
	ctx.EtaIncident = media.IOR(incident, *ctx)
	ctx.EtaTransmit = media.IOR(transmit, *ctx)
	ctx.CurrentIOR = ctx.EtaIncident
	ray.RefractionIndex = ctx.EtaIncident
}

func applyMediumTransmission(media *medium.Registry, ray *renderray.Ray, ctx core.ShadingContext, boundary medium.Boundary, sample core.BxDFSample) {
	if boundary.Active() && sample.TransmitMedium != core.MediumNone {
		if !boundary.Thin {
			if ctx.Entering {
				ray.MediumStack.EnterBoundary(boundary)
			} else {
				ray.MediumStack.ExitBoundary(boundary)
			}
		}
		ray.RefractionIndex = media.IOR(ray.MediumStack.Current(), ctx)
		return
	}

	if sample.Eta > 0 {
		ray.RefractionIndex = sample.Eta
	}
}

func applySpectrum(color *mat.VecDense, spectrum core.Spectrum) {
	if spectrum.HasSamples() {
		power := spectrum.Average()
		color.SetVec(0, color.AtVec(0)*power)
		color.SetVec(1, color.AtVec(1)*power)
		color.SetVec(2, color.AtVec(2)*power)
		return
	}
	color.SetVec(0, color.AtVec(0)*spectrum.RGBChannel(0))
	color.SetVec(1, color.AtVec(1)*spectrum.RGBChannel(1))
	color.SetVec(2, color.AtVec(2)*spectrum.RGBChannel(2))
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

func worldToLocalNegated(v, normal *mat.VecDense) (core.Direction, bool) {
	tangent, bitangent, ok := tangentFrame(normal)
	if !ok {
		return core.Direction{}, false
	}
	return core.NewDirection(
		-mat.Dot(v, tangent),
		-mat.Dot(v, bitangent),
		-mat.Dot(v, normal),
	), true
}

func localToWorld(v core.Direction, normal *mat.VecDense) *mat.VecDense {
	res := mat.NewVecDense(normal.Len(), nil)
	if !localToWorldInto(res, v, normal) {
		return res
	}
	return res
}

func localToWorldInto(res *mat.VecDense, v core.Direction, normal *mat.VecDense) bool {
	tangent, bitangent, ok := tangentFrame(normal)
	if !ok {
		return false
	}

	res.Zero()
	res.AddScaledVec(res, v.X, tangent)
	res.AddScaledVec(res, v.Y, bitangent)
	res.AddScaledVec(res, v.Z, normal)
	return true
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
