package optics

import (
	"gonum.org/v1/gonum/mat"
	"math"
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
	ray.WaveLength = math_lib.WavelengthMin + rand.Float64()*(math_lib.WavelengthMax-math_lib.WavelengthMin)

	baseColor := math_lib.Normalize(WaveLengthToRGB(ray.WaveLength))
	baseColor = math_lib.ScaleVec(baseColor, 2.318, baseColor)

	ray.Color.SetVec(0, baseColor.AtVec(0)*ray.Color.AtVec(0))
	ray.Color.SetVec(1, baseColor.AtVec(1)*ray.Color.AtVec(1))
	ray.Color.SetVec(2, baseColor.AtVec(2)*ray.Color.AtVec(2))
}

func WaveLengthToRGB(wavelength float64) *mat.VecDense {
	if wavelength >= math_lib.WavelengthMax || wavelength <= math_lib.WavelengthMin {
		return mat.NewVecDense(3, []float64{0, 0, 0})
	}
	t := (wavelength - math_lib.WavelengthMin) / (math_lib.WavelengthMax - math_lib.WavelengthMin)
	r, g, b := math_lib.SpectrumFiveDivided(t)
	return mat.NewVecDense(3, []float64{
		math.Max(0, math.Min(1, r)),
		math.Max(0, math.Min(1, g)),
		math.Max(0, math.Min(1, b)),
	})
}
