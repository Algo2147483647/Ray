package ray_tracing

import (
	"math"

	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/utils/maths"
	"gonum.org/v1/gonum/mat"
)

func worldToLocal(v, normal *mat.VecDense) (maths.Direction, bool) {
	tangent, bitangent, ok := tangentFrame(normal)
	if !ok {
		return maths.Direction{}, false
	}
	return maths.NewDirection(
		mat.Dot(v, tangent),
		mat.Dot(v, bitangent),
		mat.Dot(v, normal),
	), true
}

func worldToLocalNegated(v, normal *mat.VecDense) (maths.Direction, bool) {
	tangent, bitangent, ok := tangentFrame(normal)
	if !ok {
		return maths.Direction{}, false
	}
	return maths.NewDirection(
		-mat.Dot(v, tangent),
		-mat.Dot(v, bitangent),
		-mat.Dot(v, normal),
	), true
}

func localToWorld(v maths.Direction, normal *mat.VecDense) *mat.VecDense {
	res := mat.NewVecDense(normal.Len(), nil)
	if !localToWorldInto(res, v, normal) {
		return res
	}
	return res
}

func localToWorldInto(res *mat.VecDense, v maths.Direction, normal *mat.VecDense) bool {
	tangent, bitangent, ok := tangentFrame(normal)
	if !ok {
		return false
	}

	res.Zero()
	res.AddScaledVec(res, v.X, tangent)
	res.AddScaledVec(res, v.Y, bitangent)
	res.AddScaledVec(res, v.Z, normal)
	return true
}

func tangentFrame(normal *mat.VecDense) (*mat.VecDense, *mat.VecDense, bool) {
	if normal.Len() != 3 {
		return nil, nil, false
	}

	n := mat.VecDenseCopyOf(normal)
	math_lib.Normalize(n)

	var tangent *mat.VecDense
	if math.Abs(n.AtVec(2)) < 0.999999 {
		tangent = mat.NewVecDense(3, []float64{-n.AtVec(1), n.AtVec(0), 0})
	} else {
		tangent = mat.NewVecDense(3, []float64{0, 1, 0})
	}
	math_lib.Normalize(tangent)

	bitangent := math_lib.Cross2(n, tangent)
	math_lib.Normalize(bitangent)
	return tangent, bitangent, true
}
