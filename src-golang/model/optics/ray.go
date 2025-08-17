package optics

import (
	"gonum.org/v1/gonum/mat"
	"math/rand/v2"
)

type Ray struct {
	Origin          *mat.VecDense `json:"origin"`
	Direction       *mat.VecDense `json:"direction"`
	Color           *mat.VecDense `json:"color"`
	WaveLength      float64       `json:"wave_length"`
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

	baseColor := WaveLengthToRGB(ray.WaveLength)       // 2. 计算波长对应的基础颜色
	originalLuminance := CalculateLuminance(ray.Color) // 3. 调整光线颜色：保留原始亮度但转换为单色
	newLuminance := CalculateLuminance(baseColor)
	if newLuminance < 0.001 { // 缩放颜色以保持原始亮度
		newLuminance = 0.001
	}

	scale := originalLuminance / newLuminance // 避免除以零
	ray.Color.ScaleVec(scale, baseColor)
}
