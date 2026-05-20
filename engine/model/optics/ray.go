package optics

import (
	"math"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type Ray struct {
	Origin          *mat.VecDense `json:"origin"`
	Direction       *mat.VecDense `json:"direction"`
	Color           *mat.VecDense `json:"color"`
	WaveLength      float64       `json:"wave_length"` // (nm)
	WavelengthPDF   float64       `json:"wavelength_pdf"`
	RefractionIndex float64       `json:"refraction_index"`
	MediumStack     medium.Stack  `json:"-"`
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

	if r.Color == nil {
		r.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	} else if r.Color.Len() == 3 {
		r.Color.SetVec(0, 1)
		r.Color.SetVec(1, 1)
		r.Color.SetVec(2, 1)
	} else {
		r.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	}

	r.RefractionIndex = 1
	r.MediumStack.Reset(0)
	r.WaveLength = 0
	r.WavelengthPDF = 0
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
	wavelength = math.Max(WavelengthMin+1e-6, math.Min(WavelengthMax-1e-6, wavelength))
	r.WaveLength = wavelength
	r.WavelengthPDF = UniformWavelengthPDF()
}

func (r *Ray) DisableSpectralSampling() {
	r.WaveLength = 0
	r.WavelengthPDF = 0
	if r.Color == nil {
		r.Color = mat.NewVecDense(3, []float64{1, 1, 1})
		return
	}
	r.Color.SetVec(0, 1)
	r.Color.SetVec(1, 1)
	r.Color.SetVec(2, 1)
}
