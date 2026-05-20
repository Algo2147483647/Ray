package bxdf

import (
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils/maths"
)

type BxDF interface {
	Scattering
}

type ParameterGradients struct {
	Values map[string]optics.Spectrum
}

type DifferentiableBxDF interface {
	ParameterDerivatives(ctx ShadingContext, wi, wo maths.Direction) ParameterGradients
}

type TransportMode int

const (
	TransportRadiance TransportMode = iota
	TransportImportance
)

type SpectrumMode int

const (
	SpectrumRGB            SpectrumMode = iota // RGB-only mode; no wavelength sampling is performed.
	SpectrumSpectral                           // Hero-wavelength mode; each camera path carries one sampled wavelength.
	SpectrumRGBAndSpectral                     // Hybrid sampled mode; RGB compatibility fields are kept while spectral sample channels are evaluated.
)

type DeltaFlags uint32

const (
	DeltaNone         DeltaFlags = 0         // Non-delta scattering; the event has a finite PDF over directions.
	DeltaReflection   DeltaFlags = 1 << iota // Perfect specular reflection; the outgoing direction is deterministic.
	DeltaTransmission                        // Perfect specular transmission/refraction; the outgoing direction is deterministic.
	NonReciprocal                            // Scattering is not reciprocal; swapping wi and wo may change the value or PDF.
)

type RoughnessInfo struct {
	IsDelta bool
	AlphaX  float64
	AlphaY  float64
}

type ShadingContext struct {
	TransportMode  TransportMode
	SpectrumMode   SpectrumMode
	CurrentIOR     float64
	WavelengthNM   float64
	WavelengthsNM  []float64
	WavelengthPDF  float64
	EtaIncident    float64
	EtaTransmit    float64
	IncidentMedium medium.MediumID
	TransmitMedium medium.MediumID
	Entering       bool
}

func (ctx ShadingContext) SpectralWavelengthNM() float64 {
	return ctx.WavelengthNM
}

func (ctx ShadingContext) SpectralWavelengthsNM() []float64 {
	return ctx.WavelengthsNM
}

type BxDFSample struct {
	Wi             maths.Direction
	F              optics.Spectrum
	PDF            float64
	Flags          DeltaFlags
	Eta            float64
	WavelengthNM   float64
	TransmitMedium medium.MediumID
}

type Scattering interface {
	Eval(ctx ShadingContext, wi, wo maths.Direction) optics.Spectrum            // Evaluates the scattering function value for an incoming direction wi and outgoing direction wo.
	Sample(ctx ShadingContext, wo maths.Direction, u maths.Sample2D) BxDFSample // Samples an incoming direction given an outgoing direction and a 2D random sample.
	PDF(ctx ShadingContext, wi, wo maths.Direction) float64                     // Returns the probability density of sampling wi given wo.
	AlbedoBound(ctx ShadingContext) optics.Spectrum                             // Returns an upper bound estimate of the scattering albedo.
	RoughnessInfo(ctx ShadingContext) RoughnessInfo                             // Returns roughness-related metadata for the scattering model.
	DeltaFlags() DeltaFlags                                                     // Returns flags describing whether the scattering contains delta/discrete components.
}
