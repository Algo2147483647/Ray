package utils

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
)

// FormatVec 格式化向量为可读字符串
func FormatVec(v *mat.VecDense) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("(%.2f, %.2f, %.2f)", v.At(0, 0), v.At(1, 0), v.At(2, 0))
}

// 将单个 mat.Dense 矩阵转换为二维 float64 切片
func MatrixToSlice(m *mat.Dense) [][]float64 {
	rows, cols := m.Dims()
	result := make([][]float64, rows)
	for i := range result {
		result[i] = make([]float64, cols)
		for j := range result[i] {
			result[i][j] = m.At(i, j)
		}
	}
	return result
}
