package utils

import (
	"encoding/json"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"os"
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

// 将三个矩阵保存到 JSON 文件
func SaveMatricesToJSON(matrices [3]*mat.Dense, filename string) error {
	// 创建包含三个二维切片的结构
	data := struct {
		Matrix1 [][]float64 `json:"matrix1"`
		Matrix2 [][]float64 `json:"matrix2"`
		Matrix3 [][]float64 `json:"matrix3"`
	}{
		Matrix1: MatrixToSlice(matrices[0]),
		Matrix2: MatrixToSlice(matrices[1]),
		Matrix3: MatrixToSlice(matrices[2]),
	}

	// 创建并写入 JSON 文件
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // 可选：美化输出格式
	return encoder.Encode(data)
}
