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
	WavelengthsNM []float64
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
	Eval(ctx ShadingContext, wi, wo Direction) Spectrum             // Evaluates the scattering function value for an incoming direction wi and outgoing direction wo.
	Sample(ctx ShadingContext, wo Direction, u Sample2D) BxDFSample // Samples an incoming direction given an outgoing direction and a 2D random sample.
	PDF(ctx ShadingContext, wi, wo Direction) float64               // Returns the probability density of sampling wi given wo.
	AlbedoBound(ctx ShadingContext) Spectrum                        // Returns an upper bound estimate of the scattering albedo.
	RoughnessInfo(ctx ShadingContext) RoughnessInfo                 // Returns roughness-related metadata for the scattering model.
	DeltaFlags() DeltaFlags                                         // Returns flags describing whether the scattering contains delta/discrete components.
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
