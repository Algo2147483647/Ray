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
