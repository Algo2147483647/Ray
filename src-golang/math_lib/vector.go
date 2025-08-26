package math_lib

import (
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

// Cross4 计算四维向量叉积（需要三个向量）
// 在四维空间中，需要三个线性无关的向量来定义一个垂直方向
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

// GramSchmidt4 对四个四维向量进行格拉姆-施密特正交化
func GramSchmidt4(v1, v2, v3, v4 *mat.VecDense) (*mat.VecDense, *mat.VecDense, *mat.VecDense, *mat.VecDense) {
	u1 := mat.VecDenseCopyOf(v1)
	Normalize(u1)

	u2 := mat.NewVecDense(4, nil)
	u2.SubVec(v2, project(v2, u1))
	Normalize(u2)

	u3 := mat.NewVecDense(4, nil)
	u3.SubVec(v3, project(v3, u1))
	temp := project(v3, u2)
	u3.SubVec(u3, temp)
	Normalize(u3)

	u4 := mat.NewVecDense(4, nil)
	u4.SubVec(v4, project(v4, u1))
	temp = project(v4, u2)
	u4.SubVec(u4, temp)
	temp = project(v4, u3)
	u4.SubVec(u4, temp)
	Normalize(u4)

	return u1, u2, u3, u4
}

// project 计算向量投影: proj_u(v) = (v·u / u·u) * u
func project(v, u *mat.VecDense) *mat.VecDense {
	dotProduct := mat.Dot(v, u)
	uNormSq := mat.Dot(u, u)
	coef := dotProduct / uNormSq

	result := mat.NewVecDense(u.Len(), nil)
	result.ScaleVec(coef, u)
	return result
}
