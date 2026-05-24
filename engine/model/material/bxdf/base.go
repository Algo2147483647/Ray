package bxdf

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

type Scattering interface {
	Eval(ctx ShadingContext, wi, wo maths.Direction) optics.Spectrum            // Evaluates the scattering function value for an incoming direction wi and outgoing direction wo.
	Sample(ctx ShadingContext, wo maths.Direction, u maths.Sample2D) BxDFSample // Samples an incoming direction given an outgoing direction and a 2D random sample.
	PDF(ctx ShadingContext, wi, wo maths.Direction) float64                     // Returns the probability density of sampling wi given wo.
	AlbedoBound(ctx ShadingContext) optics.Spectrum                             // Returns an upper bound estimate of the scattering albedo.
	RoughnessInfo(ctx ShadingContext) RoughnessInfo                             // Returns roughness-related metadata for the scattering model.
	DeltaFlags() DeltaFlags                                                     // Returns flags describing whether the scattering contains delta/discrete components.
}

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

type DeltaFlags uint32

const (
	DeltaNone         DeltaFlags = 0         // Non-delta scattering; the event has a finite PDF over directions.
	DeltaReflection   DeltaFlags = 1 << iota // Perfect specular reflection; the outgoing direction is deterministic.
	DeltaTransmission                        // Perfect specular transmission/refraction; the outgoing direction is deterministic.
	NonReciprocal                            // Scattering is not reciprocal; swapping wi and wo may change the value or PDF.
	TransmissionEvent                        // The sampled event crosses the surface and should update medium state.
)

type RoughnessInfo struct {
	IsDelta bool
	AlphaX  float64
	AlphaY  float64
}

type ShadingContext struct {
	TransportMode  TransportMode       // Light transport evaluation mode.
	SpectrumMode   optics.SpectrumMode // Spectral evaluation mode.
	CurrentIOR     float64             // Index of refraction at the current point.
	WavelengthNM   float64             // Selected wavelength in nanometers.
	WavelengthsNM  []float64           // Sampled wavelengths in nanometers.
	WavelengthPDF  float64             // Probability density of wavelength sampling.
	EtaIncident    float64             // Incident-side index of refraction.
	EtaTransmit    float64             // Transmitted-side index of refraction.
	IncidentMedium medium.MediumID     // Medium on the incident side.
	TransmitMedium medium.MediumID     // Medium on the transmitted side.
	Entering       bool                // True when crossing into the surface.
}

func (ctx ShadingContext) SpectralWavelengthNM() float64 {
	return ctx.WavelengthNM
}

func (ctx ShadingContext) SpectralWavelengthsNM() []float64 {
	return ctx.WavelengthsNM
}

type BxDFSample struct {
	Wi             maths.Direction // Sampled incident direction.
	F              optics.Spectrum // Sampled BxDF value.
	PDF            float64         // Sampling probability density.
	Flags          DeltaFlags      // Scattering event flags.
	Eta            float64         // Relative index of refraction.
	WavelengthNM   float64         // Sampled wavelength in nanometers.
	TransmitMedium medium.MediumID // Medium entered after transmission.
}
