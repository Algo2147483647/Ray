package bxdf

import (
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
	"github.com/Algo2147483647/ray/engine/utils/maths"
)

type SpecularReflection struct {
	Reflectance optics.SpectralParameter
}

func NewSpecularReflection(reflectance optics.Spectrum) SpecularReflection {
	return NewSpecularReflectionParameter(spectrum_parameter.NewRGBParameter(reflectance))
}

func NewSpecularReflectionParameter(reflectance optics.SpectralParameter) SpecularReflection {
	return SpecularReflection{Reflectance: reflectance}
}

func (s SpecularReflection) Eval(ShadingContext, maths.Direction, maths.Direction) optics.Spectrum {
	return optics.Spectrum{}
}

func (s SpecularReflection) Sample(ctx ShadingContext, wo maths.Direction, _ maths.Sample2D) BxDFSample {
	if !maths.IsUpperHemisphere(wo) {
		return BxDFSample{}
	}

	wi := reflectLocal(wo)
	cos := maths.AbsCosTheta(wi)
	return BxDFSample{
		Wi:    wi,
		F:     s.Reflectance.Eval(ctx).DivScalar(cos),
		PDF:   1,
		Flags: DeltaReflection,
	}
}

func (s SpecularReflection) PDF(ShadingContext, maths.Direction, maths.Direction) float64 {
	return 0
}

func (s SpecularReflection) AlbedoBound(ShadingContext) optics.Spectrum {
	return s.Reflectance.Bounds().Max
}

func (s SpecularReflection) RoughnessInfo(ShadingContext) RoughnessInfo {
	return RoughnessInfo{IsDelta: true}
}

func (s SpecularReflection) DeltaFlags() DeltaFlags {
	return DeltaReflection
}

type SpecularDielectric struct {
	Reflectance   optics.SpectralParameter
	Transmittance optics.SpectralParameter
	EtaOutside    float64
	InsideIOR     medium.Model
}
