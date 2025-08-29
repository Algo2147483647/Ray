package optics

import (
	"gonum.org/v1/gonum/mat"
	"math/rand"
	"src-golang/math_lib"
	"src-golang/utils"
)

// Material 表示物体的材质属性
type Material struct {
	Color           *mat.VecDense                                    `json:"color"` // 材质的基础颜色
	ColorFunc       func(ray *Ray, norm *mat.VecDense) *mat.VecDense `json:"color_func"`
	Radiation       bool                                             `json:"radiation"` // 是否自发光
	RadiationType   string                                           `json:"radiation_type"`
	Reflectivity    float64                                          `json:"reflectivity"`     // 反射系数 [0, 1]
	Refractivity    float64                                          `json:"refractivity"`     // 折射系数 [0, 1]
	RefractiveIndex *mat.VecDense                                    `json:"refractive_index"` // 折射率
	DiffuseLoss     float64                                          `json:"diffuse_loss"`     // 漫反射损失系数
	ReflectLoss     float64                                          `json:"reflect_loss"`     // 反射损失系数
	RefractLoss     float64                                          `json:"refract_loss"`     // 折射损失系数
}

// NewMaterial 创建新材质
func NewMaterial(color *mat.VecDense) *Material {
	return &Material{
		Color:       color,
		DiffuseLoss: 1.0,
		ReflectLoss: 1.0,
		RefractLoss: 1.0,
	}
}

// DielectricSurfacePropagation 处理光线在介质表面的传播
func (m *Material) DielectricSurfacePropagation(ray *Ray, norm *mat.VecDense) bool {
	if m.Radiation {
		m.LightSource(ray, norm)
		return true
	}

	randNum := rand.Float64()
	switch {
	case randNum <= m.Reflectivity:
		ray.Direction = math_lib.Reflect(ray.Direction, norm)
		ray.Color.ScaleVec(m.ReflectLoss, ray.Color)

	case randNum <= m.Reflectivity+m.Refractivity:
		refractionIndex := m.GetRefractionIndex(ray)
		ray.Direction = math_lib.Refract(ray.Direction, norm, ray.RefractionIndex/refractionIndex)
		ray.Color.ScaleVec(m.RefractLoss, ray.Color)
		ray.RefractionIndex = refractionIndex

	default:
		ray.Direction = math_lib.DiffuseReflect(ray.Direction, norm)
		ray.Color.ScaleVec(m.DiffuseLoss, ray.Color)
	}

	MaterialColor := m.GetColor(ray, norm)
	for i := 0; i < 3; i++ { // 材质基础颜色
		ray.Color.SetVec(i, ray.Color.AtVec(i)*MaterialColor.AtVec(i))
	}
	return false
}

func (m *Material) GetRefractionIndex(ray *Ray) (res float64) {
	if m.RefractiveIndex.Len() == 1 {
		res = m.RefractiveIndex.AtVec(0)

	} else if m.RefractiveIndex.Len() == 3 {
		if ray.WaveLength < math_lib.WavelengthMin {
			ray.ConvertToMonochrome()
		}

		res = math_lib.CauchyDispersion(ray.WaveLength,
			m.RefractiveIndex.AtVec(0),
			m.RefractiveIndex.AtVec(1),
			m.RefractiveIndex.AtVec(2),
		)
	}

	if ray.RefractionIndex == res { // 出射折射率
		res = 1.0
	}
	return
}

func (m *Material) GetColor(ray *Ray, norm *mat.VecDense) *mat.VecDense {
	if m.ColorFunc != nil {
		return m.ColorFunc(ray, norm)
	}
	return m.Color
}

func (m *Material) LightSource(ray *Ray, norm *mat.VecDense) {
	MaterialColor := m.GetColor(ray, norm)
	switch m.RadiationType {
	case "":
		for i := 0; i < norm.Len(); i++ {
			ray.Color.SetVec(i, ray.Color.AtVec(i)*MaterialColor.AtVec(i))
		}
	case "directional light source":
		v := mat.Dot(ray.Direction, norm)
		v = v * v
		if v < 0.98 {
			ray.Color.Zero()
			return
		}
		if utils.IsDebug && v > 0.98 {
			ray.DebugSwitch = true
		}
		for i := 0; i < norm.Len(); i++ {
			ray.Color.SetVec(i, v*ray.Color.AtVec(i)*MaterialColor.AtVec(i))
		}
	}
}
