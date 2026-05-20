package material

import (
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/core"
)

type Material struct {
	Surface  bsdf.BSDF
	Emission Emitter
	Metadata MaterialMetadata
}

type Emitter interface {
	Emit(ctx core.ShadingContext, wo core.Direction) core.Spectrum
	IsDelta() bool
}

type MaterialMetadata struct {
	Name                     string
	Units                    string
	ColorSpace               string
	SpectrumMode             core.SpectrumMode
	NonReciprocal            bool
	DifferentiabilitySupport bool
	ParameterRanges          map[string]ParameterRange
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
