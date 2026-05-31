package optics

import (
	"math"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type Ray struct {
	Origin               *mat.VecDense     `json:"origin"`
	Direction            *mat.VecDense     `json:"direction"`
	Color                RGB               `json:"color"`
	SpectralPower        float64           `json:"spectral_power"`
	SpectralPath         bool              `json:"spectral_path"`
	RGBCompatibility     RGB               `json:"rgb_compatibility"`
	RGBCompatibilityPath bool              `json:"rgb_compatibility_path"`
	WaveLength           float64           `json:"wave_length"` // (nm)
	WavelengthPDF        float64           `json:"wavelength_pdf"`
	RefractionIndex      float64           `json:"refraction_index"`
	MediumStack          medium.Stack      `json:"-"`
	Geometry             geometry.Geometry `json:"-"` // nil ⇒ Euclidean (back-compat default)
	ArcTraveled          float64           `json:"-"` // geodesic arc length traveled so far (S^3 wrap)
}

func (r *Ray) Init() {
	if r.Origin == nil {
		r.Origin = mat.NewVecDense(utils.Dimension, nil)
	} else {
		r.Origin.Zero()
	}

	if r.Direction == nil {
		r.Direction = mat.NewVecDense(utils.Dimension, nil)
	} else {
		r.Direction.Zero()
	}

	r.Color = RGB{1, 1, 1}
	r.RGBCompatibility = RGB{1, 1, 1}

	r.SpectralPower = 1
	r.SpectralPath = false
	r.RGBCompatibilityPath = false
	r.RefractionIndex = 1
	r.MediumStack.Reset(0)
	r.WaveLength = 0
	r.WavelengthPDF = 0

	r.ArcTraveled = 0
	// Geometry is intentionally NOT reset: it is set per-render by the
	// renderer when handing out a Ray, and Init may be called from a
	// pool that pre-assigns it. Setting it to nil here would break the
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
	if r.SpectralPower == 0 {
		r.SpectralPower = 1
	}
	r.WaveLength = wavelength
	r.WavelengthPDF = sample.PDF
	if r.WavelengthPDF <= 0 || math.IsNaN(r.WavelengthPDF) || math.IsInf(r.WavelengthPDF, 0) {
		r.WavelengthPDF = UniformWavelengthPDF()
	}
}

func (r *Ray) DisableSpectralSampling() {
	r.WaveLength = 0
	r.WavelengthPDF = 0
	r.SpectralPower = 1
	r.SpectralPath = false
	r.RGBCompatibilityPath = false
	r.Color = RGB{1, 1, 1}
	r.RGBCompatibility = RGB{1, 1, 1}
}

// G returns the ray's geometry, falling back to Euclidean if unset.
func (r *Ray) G() geometry.Geometry {
	return geometry.Get(r.Geometry)
}
