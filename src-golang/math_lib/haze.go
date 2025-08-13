package math_lib

import (
	"gonum.org/v1/gonum/mat"
	"math"
)

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
