package optics

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"src-golang/math_lib"
)

const (
	WavelengthMin = 380.0 // 最小波长(nm)
	WavelengthMax = 750.0 // 最大波长(nm)
)

// Material 表示物体的材质属性
type Material struct {
	Color           *mat.VecDense `json:"color"`            // 材质的基础颜色
	Radiation       bool          `json:"radiation"`        // 是否自发光
	Reflectivity    float64       `json:"reflectivity"`     // 反射系数 [0, 1]
	Refractivity    float64       `json:"refractivity"`     // 折射系数 [0, 1]
	RefractiveIndex *mat.VecDense `json:"refractive_index"` // 折射率
	DiffuseLoss     float64       `json:"diffuse_loss"`     // 漫反射损失系数
	ReflectLoss     float64       `json:"reflect_loss"`     // 反射损失系数
	RefractLoss     float64       `json:"refract_loss"`     // 折射损失系数
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
		for i := 0; i < norm.Len(); i++ {
			ray.Color.SetVec(i, ray.Color.AtVec(i)*m.Color.AtVec(i))
		}
		return true
	}

	randNum := rand.Float64()
	switch {
	case randNum <= m.Reflectivity:
		ray.Direction = math_lib.Reflect(ray.Direction, norm)
		ray.Color.ScaleVec(m.ReflectLoss, ray.Color)

	case randNum <= m.Reflectivity+m.Refractivity:
		refractionIndex := m.GetRefractionIndex(ray)
		ray.Direction = math_lib.Refract(ray.Direction, norm, refractionIndex/ray.RefractionIndex)
		ray.Color.ScaleVec(m.RefractLoss, ray.Color)
		ray.RefractionIndex = refractionIndex

	default:
		ray.Direction = math_lib.DiffuseReflect(ray.Direction, norm)
		ray.Color.ScaleVec(m.DiffuseLoss, ray.Color)
	}

	for i := 0; i < norm.Len(); i++ { // 应用基础颜色（分量乘法）
		ray.Color.SetVec(i, ray.Color.AtVec(i)*m.Color.AtVec(i))
	}
	return false
}

func (m *Material) GetRefractionIndex(ray *Ray) (res float64) {
	if m.RefractiveIndex.Len() == 1 {
		res = m.RefractiveIndex.AtVec(0)

	} else if m.RefractiveIndex.Len() == 3 {
		if ray.WaveLength < WavelengthMin {
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

func CalculateLuminance(c *mat.VecDense) float64 {
	r := c.AtVec(0)
	g := c.AtVec(1)
	b := c.AtVec(2)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func WaveLengthToRGB(wavelength float64) *mat.VecDense {
	if wavelength >= WavelengthMax || wavelength <= WavelengthMin {
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
