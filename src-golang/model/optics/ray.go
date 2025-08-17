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
	baseColor = math_lib.ScaleVec(baseColor, 3, baseColor)

	ray.Color.SetVec(0, baseColor.AtVec(0)*ray.Color.AtVec(0))
	ray.Color.SetVec(1, baseColor.AtVec(1)*ray.Color.AtVec(1))
	ray.Color.SetVec(2, baseColor.AtVec(2)*ray.Color.AtVec(2))
}

func WaveLengthToRGB(wavelength float64) *mat.VecDense {
	if wavelength >= math_lib.WavelengthMax || wavelength <= math_lib.WavelengthMin {
		return mat.NewVecDense(3, []float64{0, 0, 0})
	}

	r, g, b := 0.0, 0.0, 0.0 // 简化色散模型
	factor := 0.0
	switch {
	case wavelength < 440:
		r = (440 - wavelength) / (440 - 380)
		b = 1.0
	case wavelength < 490:
		g = (wavelength - 440) / (490 - 440)
		b = 1.0
	case wavelength < 510:
		g = 1.0
		b = (510 - wavelength) / (510 - 490)
	case wavelength < 580:
		r = (wavelength - 510) / (580 - 510)
		g = 1.0
	case wavelength < 645:
		r = 1.0
		g = (645 - wavelength) / (645 - 580)
	default: // 645-750
		r = 1.0
	}

	// 计算亮度衰减因子
	switch {
	case wavelength < 420:
		factor = 0.3 + 0.7*(wavelength-380)/(420-380)
	case wavelength < 700:
		factor = 1.0
	default: // 700-750
		factor = 0.3 + 0.7*(750-wavelength)/(750-700)
	}

	return mat.NewVecDense(3, []float64{
		math.Max(0, math.Min(1, r*factor)),
		math.Max(0, math.Min(1, g*factor)),
		math.Max(0, math.Min(1, b*factor)),
	})
}
