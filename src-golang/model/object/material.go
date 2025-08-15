package object

import (
	"gonum.org/v1/gonum/mat"
	"math/rand"
	"src-golang/math_lib"
	"src-golang/model/object/optics"
)

// Material 表示物体的材质属性
type Material struct {
	Color           *mat.VecDense `json:"color"`           // 材质的基础颜色
	Radiation       bool          `json:"radiation"`       // 是否自发光
	Reflectivity    float64       `json:"reflectivity"`    // 反射系数 [0, 1]
	Refractivity    float64       `json:"refractivity"`    // 折射系数 [0, 1]
	RefractiveIndex float64       `json:"refractiveIndex"` // 折射率
	DiffuseLoss     float64       `json:"diffuseLoss"`     // 漫反射损失系数
	ReflectLoss     float64       `json:"reflectLoss"`     // 反射损失系数
	RefractLoss     float64       `json:"refractLoss"`     // 折射损失系数
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
func (m *Material) DielectricSurfacePropagation(ray *optics.Ray, norm *mat.VecDense) bool {
	if m.Radiation {
		for i := 0; i < norm.Len(); i++ {
			ray.Color.SetVec(i, ray.Color.AtVec(i)*m.Color.AtVec(i))
		}
		return true
	}

	randNum := rand.Float64()
	switch {
	case randNum <= m.Reflectivity: // 漫反射
		ray.Direction = math_lib.DiffuseReflect(ray.Direction, norm)
		ray.Color.ScaleVec(m.ReflectLoss, ray.Color)

	case randNum <= m.Reflectivity+m.Refractivity: // 镜面反射
		ray.Direction = math_lib.Reflect(ray.Direction, norm)
		ray.Color.ScaleVec(m.RefractLoss, ray.Color)

	default: // 折射
		refractionIndex := m.RefractiveIndex
		if ray.Refractivity == 1.0 {
			refractionIndex = 1.0 / m.RefractiveIndex
		}
		ray.Direction = math_lib.Refract(ray.Direction, norm, refractionIndex)
		ray.Refractivity = refractionIndex
	}

	for i := 0; i < norm.Len(); i++ { // 应用基础颜色（分量乘法）
		ray.Color.SetVec(i, ray.Color.AtVec(i)*m.Color.AtVec(i))
	}
	return false
}
