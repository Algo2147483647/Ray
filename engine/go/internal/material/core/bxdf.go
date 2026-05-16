package core

type TransportMode int

const (
	TransportRadiance TransportMode = iota
	TransportImportance
)

type SpectrumMode int

const (
	SpectrumRGB SpectrumMode = iota
	SpectrumSpectral
	SpectrumRGBAndSpectral
)

type DeltaFlags uint32

const (
	DeltaNone       DeltaFlags = 0
	DeltaReflection DeltaFlags = 1 << iota
	DeltaTransmission
	NonReciprocal
)

type RoughnessInfo struct {
	IsDelta bool
	AlphaX  float64
	AlphaY  float64
}

type ShadingContext struct {
	TransportMode TransportMode
	SpectrumMode  SpectrumMode
	CurrentIOR    float64
	WavelengthNM  float64
	WavelengthPDF float64
}

type BxDFSample struct {
	Wi           Direction
	F            Spectrum
	PDF          float64
	Flags        DeltaFlags
	Eta          float64
	WavelengthNM float64
}

type Scattering interface {
	Eval(ctx ShadingContext, wi, wo Direction) Spectrum
	Sample(ctx ShadingContext, wo Direction, u Sample2D) BxDFSample
	PDF(ctx ShadingContext, wi, wo Direction) float64

	AlbedoBound(ctx ShadingContext) Spectrum
	RoughnessInfo(ctx ShadingContext) RoughnessInfo
	DeltaFlags() DeltaFlags
}

type BxDF interface {
	Scattering
}

type BSDF interface {
	Scattering
}

type ParameterGradients struct {
	Values map[string]Spectrum
}

type DifferentiableBxDF interface {
	ParameterDerivatives(ctx ShadingContext, wi, wo Direction) ParameterGradients
}
