package math_lib

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
)

// Reflect 计算光线的反射方向
func Reflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	scale := ScaleVec2(normal, 2*mat.Dot(normal, incidentRay))
	incidentRay.SubVec(incidentRay, scale)
	return Normalize(incidentRay)
}

// Refract 计算光线的折射方向
func Refract(incidentRay, normal *mat.VecDense, eta float64) *mat.VecDense {
	// refractionIndexRatio = refractive index of the incident medium / refractive index of the outgoing medium
	cosI := mat.Dot(normal, incidentRay)
	sin2T := eta * eta * (1.0 - cosI*cosI)
	if sin2T > 1.0 { // Total internal reflection
		return Reflect(incidentRay, normal)
	}

	incidentRay.AddVec(ScaleVec2(incidentRay, eta), ScaleVec2(normal, eta*cosI-math.Sqrt(1.0-sin2T)))
	return Normalize(incidentRay)
}

// DiffuseReflect 计算漫反射方向
func DiffuseReflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	angle := 2 * math.Pi * rand.Float64()
	r := math.Sqrt(rand.Float64()) // 余弦加权采样

	// 选择切向量基底
	var tangent, u, v, t *mat.VecDense
	if math.Abs(normal.AtVec(0)) > EPS {
		tangent = mat.NewVecDense(3, []float64{0, 1, 0}) // UnitY
	} else {
		tangent = mat.NewVecDense(3, []float64{1, 0, 0}) // UnitX
	}

	scale := math.Sqrt(r)
	u.ScaleVec(math.Cos(angle)*scale, Normalize(Cross(tangent, normal)))
	v.ScaleVec(math.Sin(angle)*scale, Normalize(Cross(normal, u)))
	t.ScaleVec(math.Sqrt(1-r), normal)
	incidentRay.AddVec(t, u)
	incidentRay.AddVec(incidentRay, v)
	return Normalize(incidentRay)
}

// ComputeHaze 计算单色光的雾效
func ComputeHaze(intensity, ambientLight, distance, attenuationCoefficient float64) float64 {
	transmission := math.Exp(-attenuationCoefficient * distance)
	return intensity*transmission + ambientLight*(1-transmission)
}

// ComputeHazeColor 计算彩色光的雾效
func ComputeHazeColor(intensity, ambientLight *mat.VecDense, distance, attenuationCoefficient float64) *mat.VecDense {
	transmission := math.Exp(-attenuationCoefficient * distance)

	// 计算传输光分量
	transmitted := ScaleVec2(intensity, transmission)

	// 计算环境光分量
	ambient := ScaleVec2(ambientLight, 1-transmission)

	// 组合结果
	result := new(mat.VecDense)
	result.AddVec(transmitted, ambient)
	return result
}

func ScaleVec2(v *mat.VecDense, s float64) *mat.VecDense {
	result := new(mat.VecDense)
	result.ScaleVec(s, v)
	return result
}
