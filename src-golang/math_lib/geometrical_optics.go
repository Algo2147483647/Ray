package math_lib

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
)

// Reflect 计算光线的反射方向
func Reflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	return Normalize(SubVec(incidentRay, incidentRay, ScaleVec2(2*mat.Dot(normal, incidentRay), normal)))
}

// Refract 计算光线的折射方向
func Refract(incidentRay, normal *mat.VecDense, eta float64) *mat.VecDense {
	cosI := mat.Dot(normal, incidentRay)
	sin2T := eta * eta * (1.0 - cosI*cosI)
	if sin2T > 1.0 { // Total internal reflection
		return Reflect(incidentRay, normal)
	}

	return Normalize(AddVec(incidentRay, ScaleVec(incidentRay, eta, incidentRay), ScaleVec2(eta*cosI-math.Sqrt(1.0-sin2T), normal)))
}

// DiffuseReflect 计算漫反射方向
func DiffuseReflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	angle := 2 * math.Pi * rand.Float64()
	r := rand.Float64() // 余弦加权采样

	var tangent *mat.VecDense // 选择切向量基底
	if math.Abs(normal.AtVec(0)) > EPS {
		tangent = mat.NewVecDense(3, []float64{0, 1, 0}) // UnitY
	} else {
		tangent = mat.NewVecDense(3, []float64{1, 0, 0}) // UnitX
	}

	u := ScaleVec2(math.Cos(angle)*math.Sqrt(r), Normalize(Cross(tangent, normal)))
	v := ScaleVec2(math.Sin(angle)*math.Sqrt(r), Normalize(Cross(normal, u)))
	return Normalize(AddVecs(incidentRay, ScaleVec2(math.Sqrt(1-r), normal), u, v))
}

// CauchyDispersion Cauchy 公式, 计算给定波长下的折射率
func CauchyDispersion(wavelength, A, B, C float64) float64 {
	wl2 := wavelength * wavelength
	return A + B/wl2 + C/(wl2*wl2)
}
