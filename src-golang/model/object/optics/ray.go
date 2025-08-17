package optics

import (
	"gonum.org/v1/gonum/mat"
)

type Ray struct {
	Origin            *mat.VecDense `json:"origin"`
	Direction         *mat.VecDense `json:"direction"`
	Color             *mat.VecDense `json:"color"`
	RefractionIndex   float64       `json:"refraction_index"`
	RefractColorIndex int           `json:"refract_color_index"`
	DebugSwitch       bool          `json:"debug_switch"`
}

var DebugRayTraces []map[string]interface{}

func (r *Ray) Init() {
	if r.Origin == nil {
		r.Origin = mat.NewVecDense(3, nil)
	} else {
		r.Origin.Zero()
	}

	if r.Direction == nil {
		r.Direction = mat.NewVecDense(3, nil)
	} else {
		r.Origin.Zero()
	}

	if r.Color == nil {
		r.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	} else {
		r.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	}

	r.RefractionIndex = 1
	r.RefractColorIndex = -1
	r.DebugSwitch = false
}
