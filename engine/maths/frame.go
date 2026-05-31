package maths

import (
	"gonum.org/v1/gonum/mat"
)

type Frame struct {
	Tangent   *mat.VecDense
	Bitangent *mat.VecDense
	Normal    *mat.VecDense
	Tangents  []*mat.VecDense
}

func NewFrameFromNormal(normal *mat.VecDense) (Frame, bool) {
	if normal == nil || normal.Len() < 2 || mat.Norm(normal, 2) == 0 {
		return Frame{}, false
	}

	n := mat.VecDenseCopyOf(normal)
	Normalize(n)

	tangents := orthonormalTangents(n)
	if len(tangents) != normal.Len()-1 {
		return Frame{}, false
	}

	var tangent, bitangent *mat.VecDense
	if len(tangents) > 0 {
		tangent = tangents[0]
	}
	if normal.Len() == 3 {
		bitangent = Cross2(n, tangent)
		Normalize(bitangent)
		tangents[1] = bitangent
	} else if len(tangents) > 1 {
		bitangent = tangents[1]
	}

	return Frame{
		Tangent:   tangent,
		Bitangent: bitangent,
		Normal:    n,
		Tangents:  tangents,
	}, true
}

func (f Frame) WorldToLocal(v *mat.VecDense) Direction {
	components := make([]float64, len(f.Tangents)+1)
	for i, tangent := range f.Tangents {
		components[i] = mat.Dot(v, tangent)
	}
	components[len(components)-1] = mat.Dot(v, f.Normal)
	return NewDirectionFromComponents(components)
}

func (f Frame) WorldToLocalNegated(v *mat.VecDense) Direction {
	return f.WorldToLocal(v).MulScalar(-1)
}

func (f Frame) LocalToWorld(v Direction) *mat.VecDense {
	res := mat.NewVecDense(f.Normal.Len(), nil)
	f.LocalToWorldInto(res, v)
	return res
}

func (f Frame) LocalToWorldInto(res *mat.VecDense, v Direction) {
	res.Zero()
	for i, tangent := range f.Tangents {
		res.AddScaledVec(res, v.Component(i), tangent)
	}
	res.AddScaledVec(res, v.Component(len(f.Tangents)), f.Normal)
}

func orthonormalTangents(normal *mat.VecDense) []*mat.VecDense {
	dim := normal.Len()
	tangents := make([]*mat.VecDense, 0, dim-1)
	for axis := 0; axis < dim && len(tangents) < dim-1; axis++ {
		candidate := mat.NewVecDense(dim, nil)
		candidate.SetVec(axis, 1)
		candidate.AddScaledVec(candidate, -mat.Dot(candidate, normal), normal)
		for _, tangent := range tangents {
			candidate.AddScaledVec(candidate, -mat.Dot(candidate, tangent), tangent)
		}
		if mat.Norm(candidate, 2) <= 1e-12 {
			continue
		}
		tangents = append(tangents, Normalize(candidate))
	}
	return tangents
}
