package utils

import (
	"github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
)

// Reflect 计算光线的反射方向
func Reflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	return linear_algebra.Normalize(linear_algebra.SubVec(incidentRay, incidentRay, linear_algebra.ScaleVec2(2*mat.Dot(normal, incidentRay), normal)))
}

// Refract 计算光线的折射方向, n = n_I / n_T, normal 与入射光线方向相反
func Refract(incidentRay, normal *mat.VecDense, eta float64) *mat.VecDense {
	cosI := math.Abs(mat.Dot(normal, incidentRay))
	sin2T := eta * eta * (1.0 - cosI*cosI) // sin^2 T
	if sin2T > 1.0 {                       // Total internal reflection
		return Reflect(incidentRay, normal)
	}
	cosT := math.Sqrt(1.0 - sin2T)

	return linear_algebra.Normalize(linear_algebra.AddVec(incidentRay, linear_algebra.ScaleVec(incidentRay, eta, incidentRay), linear_algebra.ScaleVec2(-cosT+eta*cosI, normal)))
}

// DiffuseReflect 计算漫反射方向
func DiffuseReflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	if normal.Len() == 4 {
		return DiffuseReflect4D(incidentRay, normal)
	}

	u := VectorPool.Get().(*mat.VecDense)
	v := VectorPool.Get().(*mat.VecDense)
	t := VectorPool.Get().(*mat.VecDense)
	tangent := VectorPool.Get().(*mat.VecDense)
	defer func() {
		VectorPool.Put(u)
		VectorPool.Put(v)
		VectorPool.Put(t)
		VectorPool.Put(tangent)
	}()

	angle := 2 * math.Pi * rand.Float64()
	r := rand.Float64() // 余弦加权采样

	tangent.Zero()
	if math.Abs(normal.AtVec(0)) > EPS {
		tangent.SetVec(1, 1) // UnitY
	} else {
		tangent.SetVec(0, 1) // UnitX
	}

	linear_algebra.ScaleVec(u, math.Cos(angle)*math.Sqrt(r), linear_algebra.Normalize(linear_algebra.Cross(t, tangent, normal)))
	linear_algebra.ScaleVec(v, math.Sin(angle)*math.Sqrt(r), linear_algebra.Normalize(linear_algebra.Cross(t, normal, u)))
	return linear_algebra.Normalize(linear_algebra.AddVecs(incidentRay, linear_algebra.ScaleVec(t, math.Sqrt(1-r), normal), u, v))
}

func DiffuseReflect4D(incidentRay, normal *mat.VecDense) *mat.VecDense {
	r := rand.Float64()
	rsqrt := math.Sqrt(r)
	invSqrt := math.Sqrt(1 - r)

	// 在4维超球面上生成均匀随机方向, 生成4个独立的高斯随机数
	u := mat.NewVecDense(4, nil)
	for i := 0; i < u.Len(); i++ {
		u.SetVec(i, rand.NormFloat64())
	}
	linear_algebra.Normalize(u)

	// 确保u与法线正交
	dot := mat.Dot(u, normal)
	u.AddScaledVec(u, -dot, normal)
	linear_algebra.Normalize(u)

	// 组合最终方向
	res := mat.NewVecDense(4, nil)
	res.AddScaledVec(res, invSqrt, normal)
	res.AddScaledVec(res, rsqrt, u)
	return linear_algebra.Normalize(res)
}

// CauchyDispersion Cauchy 公式, 计算给定波长下的折射率
func CauchyDispersion(wavelength, A, B, C float64) float64 {
	wl2 := wavelength * wavelength
	res := A + B/wl2 + C/(wl2*wl2)
	return res
}
