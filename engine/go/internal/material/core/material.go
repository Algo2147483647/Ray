package core

type Material struct {
	Surface  BSDF
	Emission Emitter
	Metadata MaterialMetadata
}

type Emitter interface {
	Emit(ctx ShadingContext, wo Direction) Spectrum
	IsDelta() bool
}

type MaterialMetadata struct {
	Name                     string
	Units                    string
	ColorSpace               string
	SpectrumMode             SpectrumMode
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
