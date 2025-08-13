package math_lib

import (
	"gonum.org/v1/gonum/mat"
)

const (
	EPS = 1e-6 // 微小量，用于浮点数比较
)

func Normalize(v *mat.VecDense) *mat.VecDense {
	norm := mat.Norm(v, 2)
	if norm == 0 {
		return v
	}
	v.ScaleVec(1/norm, v)
	return v
}

func Cross(u, v *mat.VecDense) *mat.VecDense {
	if u.Len() != 3 || v.Len() != 3 {
		panic("The cross product requires that the vector must be three-dimensional.")
	}
	result := mat.NewVecDense(3, nil)
	result.SetVec(0, u.AtVec(1)*v.AtVec(2)-u.AtVec(2)*v.AtVec(1))
	result.SetVec(1, u.AtVec(2)*v.AtVec(0)-u.AtVec(0)*v.AtVec(2))
	result.SetVec(2, u.AtVec(0)*v.AtVec(1)-u.AtVec(1)*v.AtVec(0))
	return result
}

func ScaleVec(res *mat.VecDense, s float64, v *mat.VecDense) *mat.VecDense {
	res.ScaleVec(s, v)
	return res
}

func ScaleVec2(s float64, v *mat.VecDense) *mat.VecDense {
	result := new(mat.VecDense)
	result.ScaleVec(s, v)
	return result
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
