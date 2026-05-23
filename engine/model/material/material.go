package material

import (
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils/maths"
)

type Material struct {
	Surface  bsdf.BSDF
	Emission Emitter
	Metadata MaterialMetadata
}

type Emitter interface {
	Emit(ctx bxdf.ShadingContext, wo maths.Direction) optics.Spectrum
	IsDelta() bool
}

type MaterialMetadata struct {
	Name                     string                    // Material name.
	Units                    string                    // Units used by material parameters.
	ColorSpace               string                    // Color space for color parameters.
	SpectrumMode             optics.SpectrumMode       // Supported spectral representation mode.
	NonReciprocal            bool                      // True for non-reciprocal scattering.
	DifferentiabilitySupport bool                      // True if differentiable rendering is supported.
	ParameterRanges          map[string]ParameterRange // Valid ranges for named parameters.
}

type ParameterRange struct {
	Min float64
	Max float64
}

func (m Material) HasSurface() bool {
	return m.Surface != nil
}

func (m Material) HasEmission() bool {
	return m.Emission != nil
}
