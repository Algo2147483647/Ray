package object

import (
	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/spatial/r3"
	"src-golang/model"
)

// Material 表示物体的材质属性
type Material struct {
	Color           r3.Vec  // 材质的基础颜色
	Radiation       bool    // 是否自发光
	Reflectivity    float64 // 反射系数 [0, 1]
	Refractivity    float64 // 折射系数 [0, 1]
	RefractiveIndex float64 // 折射率
	DiffuseLoss     float64 // 漫反射损失系数
	ReflectLoss     float64 // 反射损失系数
	RefractLoss     float64 // 折射损失系数
}

// NewMaterial 创建新材质
func NewMaterial(color r3.Vec) *Material {
	return &Material{
		Color:       color,
		DiffuseLoss: 1.0,
		ReflectLoss: 1.0,
		RefractLoss: 1.0,
	}
}

// DielectricSurfacePropagation 处理光线在介质表面的传播
func (m *Material) DielectricSurfacePropagation(ray *model.Ray, norm r3.Vec, rng *rand.Rand) bool {
	if m.Radiation {
		ray.Color = m.Color
		return true
	}

	randNum := rng.Float64()

	switch {
	case randNum <= m.Reflectivity:
		ray.Direction = model.DiffuseReflect(ray.Direction, norm)
		ray.Color = r3.Scale(m.ReflectLoss, ray.Color)

	case randNum <= m.Reflectivity+m.Refractivity:
		ray.Direction = model.Reflect(ray.Direction, norm)
		ray.Color = r3.Scale(m.RefractLoss, ray.Color)

	default:
		refractionIndex := m.RefractiveIndex
		if ray.Refractivity == 1.0 {
			refractionIndex = 1.0 / m.RefractiveIndex
		}
		ray.Direction = model.Refract(ray.Direction, norm, refractionIndex)
		ray.Refractivity = refractionIndex
	}

	// 应用基础颜色
	ray.Color = r3.Vec{
		X: ray.Color.X * m.Color.X,
		Y: ray.Color.Y * m.Color.Y,
		Z: ray.Color.Z * m.Color.Z,
	}
	return false
}
