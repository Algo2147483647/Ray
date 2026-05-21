package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"math"
	"math/rand/v2"

	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
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

func prepareMediumContext(ctx *bxdf.ShadingContext, media *medium.Registry, ray *renderray.Ray, boundary medium.Boundary, frontFace bool) {
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

func applyMediumTransmission(media *medium.Registry, ray *renderray.Ray, ctx bxdf.ShadingContext, boundary medium.Boundary, sample bxdf.BxDFSample) {
	if boundary.Active() && sample.TransmitMedium != medium.MediumNone {
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

func applyMediumAbsorption(media *medium.Registry, ray *renderray.Ray, distance float64, ctx bxdf.ShadingContext) {
	if media == nil || ray == nil || distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return
	}
	sigmaA := media.SigmaA(ray.MediumStack.Current(), ctx)
	if sigmaA.HasSamples() {
		transmittance := math.Exp(-sigmaA.Sample(0) * distance)
		ray.SpectralPower *= transmittance
		ray.SpectralPath = true
		return
	}
	transmittance := optics.NewRGBSpectrum(
		math.Exp(-sigmaA.RGBChannel(0)*distance),
		math.Exp(-sigmaA.RGBChannel(1)*distance),
		math.Exp(-sigmaA.RGBChannel(2)*distance),
	)
	applySpectrum(ray, transmittance)
}

func applySpectrum(ray *renderray.Ray, spectrum optics.Spectrum) {
	if ray.WaveLength > 0 {
		if spectrum.HasSamples() {
			ray.SpectralPower *= spectrum.Sample(0)
			ray.SpectralPath = true
			return
		}
		ensureRGBCompatibility(ray)
		ray.RGBCompatibility.SetVec(0, ray.RGBCompatibility.AtVec(0)*spectrum.RGBChannel(0))
		ray.RGBCompatibility.SetVec(1, ray.RGBCompatibility.AtVec(1)*spectrum.RGBChannel(1))
		ray.RGBCompatibility.SetVec(2, ray.RGBCompatibility.AtVec(2)*spectrum.RGBChannel(2))
		return
	}

	color := ray.Color
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

func terminateRay(ray *renderray.Ray) *mat.VecDense {
	if ray == nil {
		return mat.NewVecDense(3, nil)
	}
	if ray.Color == nil {
		ray.Color = mat.NewVecDense(3, nil)
	} else {
		ray.Color.ScaleVec(0, ray.Color)
	}
	ray.SpectralPower = 0
	ray.SpectralPath = false
	if ray.RGBCompatibility != nil {
		ray.RGBCompatibility.ScaleVec(0, ray.RGBCompatibility)
	}
	return ray.Color
}

func ensureRGBCompatibility(ray *renderray.Ray) {
	if ray.RGBCompatibility == nil || ray.RGBCompatibility.Len() != 3 {
		ray.RGBCompatibility = mat.NewVecDense(3, []float64{1, 1, 1})
	}
}

func negateVec(v *mat.VecDense) *mat.VecDense {
	res := mat.NewVecDense(v.Len(), nil)
	res.ScaleVec(-1, v)
	return res
}

func worldToLocal(v, normal *mat.VecDense) (maths.Direction, bool) {
	tangent, bitangent, ok := tangentFrame(normal)
	if !ok {
		return maths.Direction{}, false
	}
	return maths.NewDirection(
		mat.Dot(v, tangent),
		mat.Dot(v, bitangent),
		mat.Dot(v, normal),
	), true
}

func worldToLocalNegated(v, normal *mat.VecDense) (maths.Direction, bool) {
	tangent, bitangent, ok := tangentFrame(normal)
	if !ok {
		return maths.Direction{}, false
	}
	return maths.NewDirection(
		-mat.Dot(v, tangent),
		-mat.Dot(v, bitangent),
		-mat.Dot(v, normal),
	), true
}

func localToWorld(v maths.Direction, normal *mat.VecDense) *mat.VecDense {
	res := mat.NewVecDense(normal.Len(), nil)
	if !localToWorldInto(res, v, normal) {
		return res
	}
	return res
}

func localToWorldInto(res *mat.VecDense, v maths.Direction, normal *mat.VecDense) bool {
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
