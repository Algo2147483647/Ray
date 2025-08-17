package optics

import (
	"gonum.org/v1/gonum/mat"
	"math/rand/v2"
	"src-golang/math_lib"
)

type Ray struct {
	Origin          *mat.VecDense `json:"origin"`
	Direction       *mat.VecDense `json:"direction"`
	Color           *mat.VecDense `json:"color"`
	WaveLength      float64       `json:"wave_length"` // (nm)
	RefractionIndex float64       `json:"refraction_index"`
	DebugSwitch     bool          `json:"debug_switch"`
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
	r.WaveLength = 0
	r.DebugSwitch = false
}
func (ray *Ray) ConvertToMonochrome() {
	ray.WaveLength = WavelengthMin + rand.Float64()*(WavelengthMax-WavelengthMin)

	baseColor := math_lib.Normalize(WaveLengthToRGB(ray.WaveLength))
	baseColor = math_lib.ScaleVec(baseColor, 3, baseColor)

	ray.Color.SetVec(0, baseColor.AtVec(0)*ray.Color.AtVec(0))
	ray.Color.SetVec(0, baseColor.AtVec(1)*ray.Color.AtVec(1))
	ray.Color.SetVec(0, baseColor.AtVec(2)*ray.Color.AtVec(2))
}
