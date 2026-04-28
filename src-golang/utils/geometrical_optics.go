package utils

import (
	"github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
)

func Reflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	return linear_algebra.Normalize(linear_algebra.SubVec(incidentRay, incidentRay, linear_algebra.ScaleVec2(2*mat.Dot(normal, incidentRay), normal)))
}

func Refract(incidentRay, normal *mat.VecDense, eta float64) *mat.VecDense {
	cosI := math.Abs(mat.Dot(normal, incidentRay))
	sin2T := eta * eta * (1.0 - cosI*cosI) // sin^2 T
	if sin2T > 1.0 {                       // Total internal reflection
		return Reflect(incidentRay, normal)
	}
	cosT := math.Sqrt(1.0 - sin2T)

	return linear_algebra.Normalize(linear_algebra.AddVec(incidentRay, linear_algebra.ScaleVec(incidentRay, eta, incidentRay), linear_algebra.ScaleVec2(-cosT+eta*cosI, normal)))
}

func HasTotalInternalReflection(incidentRay, normal *mat.VecDense, eta float64) bool {
	cosI := math.Abs(mat.Dot(normal, incidentRay))
	sin2T := eta * eta * (1.0 - cosI*cosI)
	return sin2T > 1.0
}

func FresnelSchlick(cosTheta, n1, n2 float64) float64 {
	cosTheta = math.Max(0, math.Min(1, cosTheta))
	r0 := (n1 - n2) / (n1 + n2)
	r0 *= r0
	return r0 + (1-r0)*math.Pow(1-cosTheta, 5)
}

func DiffuseReflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	if normal.Len() == 4 {
		return DiffuseReflect4D(incidentRay, normal)
	}

	tangent := VectorPool.Get().(*mat.VecDense)
	bitangent := VectorPool.Get().(*mat.VecDense)
	helper := VectorPool.Get().(*mat.VecDense)
	surfaceNormal := VectorPool.Get().(*mat.VecDense)
	res := VectorPool.Get().(*mat.VecDense)
	defer func() {
		VectorPool.Put(tangent)
		VectorPool.Put(bitangent)
		VectorPool.Put(helper)
		VectorPool.Put(surfaceNormal)
		VectorPool.Put(res)
	}()

	r1 := rand.Float64()
	r2 := rand.Float64()
	phi := 2 * math.Pi * r1
	x := math.Cos(phi) * math.Sqrt(r2)
	y := math.Sin(phi) * math.Sqrt(r2)
	z := math.Sqrt(1 - r2)

	surfaceNormal.CloneFromVec(normal)
	linear_algebra.Normalize(surfaceNormal)

	helper.Zero()
	if math.Abs(surfaceNormal.AtVec(0)) < 0.9 {
		helper.SetVec(0, 1)
	} else {
		helper.SetVec(1, 1)
	}

	linear_algebra.Normalize(linear_algebra.Cross(tangent, helper, surfaceNormal))
	linear_algebra.Normalize(linear_algebra.Cross(bitangent, surfaceNormal, tangent))

	res.Zero()
	res.AddScaledVec(res, x, tangent)
	res.AddScaledVec(res, y, bitangent)
	res.AddScaledVec(res, z, surfaceNormal)
	return linear_algebra.Normalize(res)
}

func DiffuseReflect4D(incidentRay, normal *mat.VecDense) *mat.VecDense {
	r := rand.Float64()
	rsqrt := math.Sqrt(r)
	invSqrt := math.Sqrt(1 - r)

	// 鍦?缁磋秴鐞冮潰涓婄敓鎴愬潎鍖€闅忔満鏂瑰悜, 鐢熸垚4涓嫭绔嬬殑楂樻柉闅忔満鏁?
	u := mat.NewVecDense(4, nil)
	for i := 0; i < u.Len(); i++ {
		u.SetVec(i, rand.NormFloat64())
	}
	linear_algebra.Normalize(u)

	// 纭繚u涓庢硶绾挎浜?
	dot := mat.Dot(u, normal)
	u.AddScaledVec(u, -dot, normal)
	linear_algebra.Normalize(u)

	// 缁勫悎鏈€缁堟柟鍚?
	res := mat.NewVecDense(4, nil)
	res.AddScaledVec(res, invSqrt, normal)
	res.AddScaledVec(res, rsqrt, u)
	return linear_algebra.Normalize(res)
}

// CauchyDispersion Cauchy 鍏紡, 璁＄畻缁欏畾娉㈤暱涓嬬殑鎶樺皠鐜?
func CauchyDispersion(wavelength, A, B, C float64) float64 {
	wl2 := wavelength * wavelength
	res := A + B/wl2 + C/(wl2*wl2)
	return res
}
