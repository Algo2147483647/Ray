package math_lib

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
	"src-golang/utils"
)

const (
	WavelengthMin = 380.0 // 最小波长(nm)
	WavelengthMax = 750.0 // 最大波长(nm)
)

// Reflect 计算光线的反射方向
func Reflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	return Normalize(SubVec(incidentRay, incidentRay, ScaleVec2(2*mat.Dot(normal, incidentRay), normal)))
}

// Refract 计算光线的折射方向, n = n_I / n_T, normal 与入射光线方向相反
func Refract(incidentRay, normal *mat.VecDense, eta float64) *mat.VecDense {
	cosI := math.Abs(mat.Dot(normal, incidentRay))
	sin2T := eta * eta * (1.0 - cosI*cosI) // sin^2 T
	if sin2T > 1.0 {                       // Total internal reflection
		return Reflect(incidentRay, normal)
	}
	cosT := math.Sqrt(1.0 - sin2T)

	return Normalize(AddVec(incidentRay, ScaleVec(incidentRay, eta, incidentRay), ScaleVec2(-cosT+eta*cosI, normal)))
}

// DiffuseReflect 计算漫反射方向
func DiffuseReflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	u := utils.VectorPool.Get().(*mat.VecDense)
	v := utils.VectorPool.Get().(*mat.VecDense)
	t := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(u)
		utils.VectorPool.Put(v)
		utils.VectorPool.Put(t)
	}()

	angle := 2 * math.Pi * rand.Float64()
	r := rand.Float64() // 余弦加权采样

	var tangent *mat.VecDense // 选择切向量基底
	if math.Abs(normal.AtVec(0)) > EPS {
		tangent = mat.NewVecDense(3, []float64{0, 1, 0}) // UnitY
	} else {
		tangent = mat.NewVecDense(3, []float64{1, 0, 0}) // UnitX
	}

	ScaleVec(u, math.Cos(angle)*math.Sqrt(r), Normalize(Cross(t, tangent, normal)))
	ScaleVec(v, math.Sin(angle)*math.Sqrt(r), Normalize(Cross(t, normal, u)))
	return Normalize(AddVecs(incidentRay, ScaleVec(t, math.Sqrt(1-r), normal), u, v))
}

// CauchyDispersion Cauchy 公式, 计算给定波长下的折射率
func CauchyDispersion(wavelength, A, B, C float64) float64 {
	wl2 := wavelength * wavelength
	res := A + B/wl2 + C/(wl2*wl2)
	return res
}
