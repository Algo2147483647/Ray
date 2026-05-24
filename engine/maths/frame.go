package maths

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

type Frame struct {
	Tangent   *mat.VecDense
	Bitangent *mat.VecDense
	Normal    *mat.VecDense
}

func NewFrameFromNormal(normal *mat.VecDense) (Frame, bool) {
	if normal.Len() != 3 {
		return Frame{}, false
	}

	n := mat.VecDenseCopyOf(normal)
	Normalize(n)

	var tangent *mat.VecDense
	if math.Abs(n.AtVec(2)) < 0.999999 {
		tangent = mat.NewVecDense(3, []float64{-n.AtVec(1), n.AtVec(0), 0})
	} else {
		tangent = mat.NewVecDense(3, []float64{0, 1, 0})
	}
	Normalize(tangent)

	bitangent := Cross2(n, tangent)
	Normalize(bitangent)

	return Frame{
		Tangent:   tangent,
		Bitangent: bitangent,
		Normal:    n,
	}, true
}

func (f Frame) WorldToLocal(v *mat.VecDense) Direction {
	return NewDirection(
		mat.Dot(v, f.Tangent),
		mat.Dot(v, f.Bitangent),
		mat.Dot(v, f.Normal),
	)
}

func (f Frame) WorldToLocalNegated(v *mat.VecDense) Direction {
	return NewDirection(
		-mat.Dot(v, f.Tangent),
		-mat.Dot(v, f.Bitangent),
		-mat.Dot(v, f.Normal),
	)
}

func (f Frame) LocalToWorld(v Direction) *mat.VecDense {
	res := mat.NewVecDense(f.Normal.Len(), nil)
	f.LocalToWorldInto(res, v)
	return res
}

func (f Frame) LocalToWorldInto(res *mat.VecDense, v Direction) {
	res.Zero()
	res.AddScaledVec(res, v.X, f.Tangent)
	res.AddScaledVec(res, v.Y, f.Bitangent)
	res.AddScaledVec(res, v.Z, f.Normal)
}
