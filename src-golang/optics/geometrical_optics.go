package optics

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"

	"gonum.org/v1/gonum/spatial/r3"
)

// Reflect 计算光线的反射方向
func Reflect(incidentRay, normal *mat.VecDense) *mat.VecDense {
	// 归一化法向量
	n := r3.Unit(normal)
	// 计算反射方向: I - 2*(N·I)*N
	dot := r3.Dot(n, incidentRay)
	return r3.Sub(incidentRay, r3.Scale(2*dot, n))
}

// Refract 计算光线的折射方向
func Refract(incidentRay, normal *mat.VecDense, refractionIndexRatio float64) *mat.VecDense {
	// 归一化入射光线和法向量
	I := r3.Unit(incidentRay)
	N := r3.Unit(normal)

	// 计算入射角余弦
	cosIncident := r3.Dot(N, I)

	// 计算折射角余弦
	cosTransmitted := 1 - refractionIndexRatio*refractionIndexRatio*(1-cosIncident*cosIncident)

	// 检查是否发生全反射
	if cosTransmitted < 0 {
		return Reflect(I, N)
	}

	// 计算折射方向: ηI + (ηcosθ_i - √cos²θ_t)N
	term1 := r3.Scale(refractionIndexRatio, I)
	term2 := r3.Scale(refractionIndexRatio*cosIncident-math.Sqrt(cosTransmitted), N)
	return r3.Unit(r3.Add(term1, term2))
}

// DiffuseReflect 计算漫反射方向
func DiffuseReflect(incidentRay, normal *mat.VecDense, rng *rand.Rand) *mat.VecDense {
	// 生成随机角度和半径
	randomAngle := 2 * math.Pi * rng.Float64()
	randomRadius := rng.Float64()

	// 创建切向量
	tangent := *mat.VecDense{X: 0, Y: 1} // 默认Y轴
	if math.Abs(normal.X) <= EPS {
		tangent = *mat.VecDense{X: 1, Y: 0} // 如果法线接近X轴，则使用X轴
	}

	// 计算正交基
	u := r3.Cross(tangent, normal)
	u = r3.Unit(u)
	v := r3.Cross(normal, u)
	v = r3.Unit(v)

	// 应用随机偏移
	u = r3.Scale(math.Cos(randomAngle)*math.Sqrt(randomRadius), u)
	v = r3.Scale(math.Sin(randomAngle)*math.Sqrt(randomRadius), v)
	normalComponent := r3.Scale(math.Sqrt(1-randomRadius), normal)

	// 组合最终方向
	return r3.Unit(r3.Add(normalComponent, r3.Add(u, v)))
}

// ComputeHaze 计算单色光的雾效
func ComputeHaze(intensity, ambientLight, distance, attenuationCoefficient float64) float64 {
	transmissionFactor := math.Exp(-attenuationCoefficient * distance)
	return intensity*transmissionFactor + ambientLight*(1-transmissionFactor)
}

// ComputeHazeColor 计算彩色光的雾效
func ComputeHazeColor(intensity, ambientLight *mat.VecDense, distance, attenuationCoefficient float64) *mat.VecDense {
	transmissionFactor := math.Exp(-attenuationCoefficient * distance)
	return r3.Add(
		r3.Scale(transmissionFactor, intensity),
		r3.Scale(1-transmissionFactor, ambientLight),
	)
}
