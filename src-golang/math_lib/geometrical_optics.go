package math_lib

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
)

// Reflect 计算光线的反射方向
func Reflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	// 归一化法向量
	n := Normalize(normal)

	// 计算点积: N·I
	dot := dotProduct(n, incidentRay)

	// 反射方向: I - 2*(N·I)*N
	scale := scaleVec(n, 2*dot)
	reflected := new(mat.VecDense)
	reflected.SubVec(incidentRay, scale)
	return reflected
}

// Refract 计算光线的折射方向
func Refract(incidentRay, normal *mat.VecDense, eta float64) *mat.VecDense {
	// 归一化入射光线和法向量
	I := Normalize(incidentRay)
	N := Normalize(normal)

	// 计算入射角余弦
	cosI := dotProduct(N, I)

	// 计算 sin²θ_t
	sin2T := eta * eta * (1.0 - cosI*cosI)

	// 检查全反射
	if sin2T > 1.0 {
		return Reflect(I, N)
	}

	// 计算折射方向: ηI + (ηcosθ_i - √(1-sin²θ_t))N
	cosT := math.Sqrt(1.0 - sin2T)
	term1 := scaleVec(I, eta)
	term2 := scaleVec(N, eta*cosI-cosT)
	refracted := new(mat.VecDense)
	refracted.AddVec(term1, term2)
	return refracted
}

// DiffuseReflect 计算漫反射方向
func DiffuseReflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	// 创建正交基
	N := Normalize(normal)
	U := createOrthogonalBasis(N)
	V := Cross(N, U)
	V = Normalize(V)

	// 生成随机角度和半径
	phi := 2 * math.Pi * rand.Float64()
	r := math.Sqrt(rand.Float64()) // 余弦加权采样

	// 计算偏移分量
	uScale := scaleVec(U, r*math.Cos(phi))
	vScale := scaleVec(V, r*math.Sin(phi))
	nScale := scaleVec(N, math.Sqrt(1-r*r))

	// 组合最终方向
	direction := new(mat.VecDense)
	direction.AddVec(uScale, vScale)
	direction.AddVec(direction, nScale)
	return Normalize(direction)
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
	transmitted := scaleVec(intensity, transmission)

	// 计算环境光分量
	ambient := scaleVec(ambientLight, 1-transmission)

	// 组合结果
	result := new(mat.VecDense)
	result.AddVec(transmitted, ambient)
	return result
}

// 辅助函数：向量缩放
func scaleVec(v *mat.VecDense, s float64) *mat.VecDense {
	result := new(mat.VecDense)
	result.ScaleVec(s, v)
	return result
}

// 辅助函数：点积计算
func dotProduct(a, b *mat.VecDense) float64 {
	return mat.Dot(a, b)
}

// 创建正交基向量
func createOrthogonalBasis(normal *mat.VecDense) *mat.VecDense {
	// 选择一个非平行向量
	ref := mat.NewVecDense(3, []float64{1, 0, 0})
	if math.Abs(mat.Dot(normal, ref)) > 0.9 {
		ref = mat.NewVecDense(3, []float64{0, 1, 0})
	}

	// 计算叉积得到正交向量
	u := Cross(normal, ref)
	return Normalize(u)
}
