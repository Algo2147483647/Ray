package optics

import (
	"math"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"gonum.org/v1/gonum/mat"
)

type Ray struct {
	Origin          *mat.VecDense       `json:"origin"`
	Direction       *mat.VecDense       `json:"direction"`
	Path            PathState           `json:"path"`
	RefractionIndex float64             `json:"refraction_index"`
	MediumStack     medium.Stack        `json:"-"`
	Space           geometry.SceneSpace `json:"-"`
	ArcTraveled     float64             `json:"-"` // geodesic arc length traveled so far (S^3 wrap)
}

// PathState is the sole representation of transport state and spectral mode.
// A nil Wavelength means RGB transport. A non-nil Wavelength means Throughput
// and Radiance are sampled Spectra aligned to that wavelength.
type PathState struct {
	Throughput Spectrum          `json:"throughput"`
	Radiance   Spectrum          `json:"radiance"`
	Wavelength *WavelengthSample `json:"wavelength,omitempty"`
}

func (r *Ray) Init() {
	dimension := r.Space.Dimension
	if dimension <= 0 && r.Origin != nil {
		dimension = r.Origin.Len()
	}
	if dimension <= 0 && r.Direction != nil {
		dimension = r.Direction.Len()
	}
	if dimension <= 0 {
		dimension = 3
	}
	if r.Origin == nil {
		r.Origin = mat.NewVecDense(dimension, nil)
	} else {
		r.Origin.Zero()
	}

	if r.Direction == nil {
		r.Direction = mat.NewVecDense(dimension, nil)
	} else {
		r.Direction.Zero()
	}

	r.Path = PathState{Throughput: ConstantSpectrum(1), Radiance: ConstantSpectrum(0)}
	r.RefractionIndex = 1
	r.MediumStack.Reset(0)

	r.ArcTraveled = 0
	// Space is intentionally NOT reset: it is set per-render by the
	// renderer when handing out a Ray, and Init may be called from a
	// pool that pre-assigns it. Clearing it here would break the
	// non-Euclidean integrator.
}

func (r *Ray) ConvertToMonochrome() {
	r.SampleWavelength(rand.Float64())
}

func (r *Ray) SetMonochrome(wavelength float64) {
	r.SetSpectralWavelength(wavelength)
}

func (r *Ray) SampleWavelength(sample float64) {
	sample = math.Max(1e-6, math.Min(1-1e-6, sample))
	r.SetSpectralWavelength(WavelengthMin + sample*(WavelengthMax-WavelengthMin))
}

func (r *Ray) SetSpectralWavelength(wavelength float64) {
	r.SetSpectralSample(WavelengthSample{
		LambdaNM: wavelength,
		PDF:      UniformWavelengthPDF(),
	})
}

func (r *Ray) SetSpectralSample(sample WavelengthSample) {
	wavelength := sample.LambdaNM
	wavelength = math.Max(WavelengthMin+1e-6, math.Min(WavelengthMax-1e-6, wavelength))
	sample.LambdaNM = wavelength
	if sample.PDF <= 0 || math.IsNaN(sample.PDF) || math.IsInf(sample.PDF, 0) {
		sample.PDF = UniformWavelengthPDF()
	}
	r.Path = PathState{
		Throughput: NewSampledSpectrum([]float64{1}),
		Radiance:   NewSampledSpectrum([]float64{0}),
		Wavelength: &sample,
	}
}

func (r *Ray) DisableSpectralSampling() {
	r.Path = PathState{Throughput: ConstantSpectrum(1), Radiance: ConstantSpectrum(0)}
}

// G returns the ray's geometry, falling back to Euclidean if unset.
func (r *Ray) G() geometry.Geometry {
	return r.Space.G()
}
