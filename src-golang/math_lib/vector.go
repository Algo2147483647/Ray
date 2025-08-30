package math_lib

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
	"math"
)

func Normalize(v *mat.VecDense) *mat.VecDense {
	norm := mat.Norm(v, 2)
	if norm == 0 {
		return v
	}
	v.ScaleVec(1/norm, v)
	return v
}

func ScaleVec(res *mat.VecDense, s float64, v *mat.VecDense) *mat.VecDense {
	res.ScaleVec(s, v)
	return res
}

func ScaleVec2(s float64, v *mat.VecDense) *mat.VecDense {
	return ScaleVec(mat.NewVecDense(v.Len(), nil), s, v)
}

func AddVec(res, a, b *mat.VecDense) *mat.VecDense {
	res.AddVec(a, b)
	return res
}

func AddVecs(res *mat.VecDense, vecs ...*mat.VecDense) *mat.VecDense {
	if len(vecs) == 0 {
		return res
	}

	res.CopyVec(vecs[0])
	for _, v := range vecs[1:] {
		res.AddVec(res, v)
	}

	return res
}

func SubVec(res, a, b *mat.VecDense) *mat.VecDense {
	res.SubVec(a, b)
	return res
}

func MulVec(res *mat.VecDense, a *mat.Dense, b *mat.VecDense) *mat.VecDense {
	res.MulVec(a, b)
	return res
}

func MinVec(a, b *mat.VecDense) *mat.VecDense {
	res := mat.NewVecDense(a.Len(), nil)
	for i := 0; i < a.Len(); i++ {
		res.SetVec(i, math.Min(a.AtVec(i), b.AtVec(i)))
	}
	return res
}

func MaxVec(a, b *mat.VecDense) *mat.VecDense {
	res := mat.NewVecDense(a.Len(), nil)
	for i := 0; i < a.Len(); i++ {
		res.SetVec(i, math.Max(a.AtVec(i), b.AtVec(i)))
	}
	return res
}

func Cross(res, u, v *mat.VecDense) *mat.VecDense {
	if res.Len() != 3 || u.Len() != 3 || v.Len() != 3 {
		panic("The cross product requires that the vector must be three-dimensional.")
	}
	res.SetVec(0, u.AtVec(1)*v.AtVec(2)-u.AtVec(2)*v.AtVec(1))
	res.SetVec(1, u.AtVec(2)*v.AtVec(0)-u.AtVec(0)*v.AtVec(2))
	res.SetVec(2, u.AtVec(0)*v.AtVec(1)-u.AtVec(1)*v.AtVec(0))
	return res
}

func Cross2(u, v *mat.VecDense) *mat.VecDense {
	return Cross(mat.NewVecDense(u.Len(), nil), u, v)
}

func Cross4(u, v, w *mat.VecDense) *mat.VecDense {
	if u.Len() != 4 || v.Len() != 4 || w.Len() != 4 {
		panic("The 4D cross product requires three 4-dimensional vectors.")
	}

	result := mat.NewVecDense(4, nil)

	// 四维叉积的计算基于行列式
	result.SetVec(0, u.AtVec(1)*(v.AtVec(2)*w.AtVec(3)-v.AtVec(3)*w.AtVec(2))-
		u.AtVec(2)*(v.AtVec(1)*w.AtVec(3)-v.AtVec(3)*w.AtVec(1))+
		u.AtVec(3)*(v.AtVec(1)*w.AtVec(2)-v.AtVec(2)*w.AtVec(1)))

	result.SetVec(1, -u.AtVec(0)*(v.AtVec(2)*w.AtVec(3)-v.AtVec(3)*w.AtVec(2))+
		u.AtVec(2)*(v.AtVec(0)*w.AtVec(3)-v.AtVec(3)*w.AtVec(0))-
		u.AtVec(3)*(v.AtVec(0)*w.AtVec(2)-v.AtVec(2)*w.AtVec(0)))

	result.SetVec(2, u.AtVec(0)*(v.AtVec(1)*w.AtVec(3)-v.AtVec(3)*w.AtVec(1))-
		u.AtVec(1)*(v.AtVec(0)*w.AtVec(3)-v.AtVec(3)*w.AtVec(0))+
		u.AtVec(3)*(v.AtVec(0)*w.AtVec(1)-v.AtVec(1)*w.AtVec(0)))

	result.SetVec(3, -u.AtVec(0)*(v.AtVec(1)*w.AtVec(2)-v.AtVec(2)*w.AtVec(1))+
		u.AtVec(1)*(v.AtVec(0)*w.AtVec(2)-v.AtVec(2)*w.AtVec(0))-
		u.AtVec(2)*(v.AtVec(0)*w.AtVec(1)-v.AtVec(1)*w.AtVec(0)))

	return result
}

// GramSchmidt Perform Gram Schmidt orthogonalization on any number of vectors
func GramSchmidt(v ...*mat.VecDense) []*mat.VecDense {
	if len(v) == 0 {
		return []*mat.VecDense{}
	}

	dim := v[0].Len()
	res := make([]*mat.VecDense, len(v))
	res[0] = Normalize(mat.VecDenseCopyOf(v[0]))

	for i := 1; i < len(v); i++ {
		res[i] = mat.NewVecDense(dim, nil) // 创建新向量，初始值为原向量的副本
		res[i].CopyVec(v[i])

		for j := 0; j < i; j++ {
			res[i].SubVec(res[i], Project(v[i], res[j])) // 减去在之前所有已正交化向量上的投影
		}

		Normalize(res[i]) // 归一化
	}

	return res
}

// project 计算向量投影: proj_u(v) = (v·u / u·u) * u
func Project(v, u *mat.VecDense) *mat.VecDense {
	dotProduct := mat.Dot(v, u)
	uNormSq := mat.Dot(u, u)
	coef := dotProduct / uNormSq

	result := mat.NewVecDense(u.Len(), nil)
	result.ScaleVec(coef, u)
	return result
}

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
