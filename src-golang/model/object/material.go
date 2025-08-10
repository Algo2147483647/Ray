package object

import (
	"gonum.org/v1/gonum/mat"
	"math/rand"
	"src-golang/math_lib"
	"src-golang/model/ray"
)

// Material 表示物体的材质属性
type Material struct {
	Color           *mat.VecDense // 材质的基础颜色
	Radiation       bool          // 是否自发光
	Reflectivity    float64       // 反射系数 [0, 1]
	Refractivity    float64       // 折射系数 [0, 1]
	RefractiveIndex float64       // 折射率
	DiffuseLoss     float64       // 漫反射损失系数
	ReflectLoss     float64       // 反射损失系数
	RefractLoss     float64       // 折射损失系数
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
func (m *Material) DielectricSurfacePropagation(ray *ray.Ray, norm *mat.VecDense, rng *rand.Rand) bool {
	if m.Radiation {
		ray.Color = m.Color
		return true
	}

	randNum := rng.Float64()
	dim := norm.Len() // 获取向量维度（应为3）

	switch {
	case randNum <= m.Reflectivity:
		// 漫反射
		ray.Direction = math_lib.DiffuseReflect(ray.Direction, norm, rand.New(rand.NewSource(0)))
		ray.Color.ScaleVec(m.ReflectLoss, ray.Color)

	case randNum <= m.Reflectivity+m.Refractivity:
		// 镜面反射
		ray.Direction = math_lib.Reflect(ray.Direction, norm)
		ray.Color.ScaleVec(m.RefractLoss, ray.Color)

	default:
		// 折射
		refractionIndex := m.RefractiveIndex
		if ray.Refractivity == 1.0 {
			refractionIndex = 1.0 / m.RefractiveIndex
		}
		ray.Direction = math_lib.Refract(ray.Direction, norm, refractionIndex)
		ray.Refractivity = refractionIndex
	}

	// 应用基础颜色（分量乘法）
	for i := 0; i < dim; i++ {
		ray.Color.SetVec(i, ray.Color.AtVec(i)*m.Color.AtVec(i))
	}
	return false
}
