package factory

import (
	"math"

	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type gyroidField struct {
	frequency float64
	offset    float64
}

func parseImplicitGyroidField(
	fieldDef map[string]interface{},
) (
	func(*mat.VecDense) float64,
	func(point, res *mat.VecDense) *mat.VecDense,
	error,
) {
	frequency, err := utils.RequiredPositiveFloat(fieldDef, "frequency")
	if err != nil {
		return nil, nil, err
	}
	offset, err := utils.RequiredFloat64Field(fieldDef, "offset")
	if err != nil {
		return nil, nil, err
	}

	field := &gyroidField{
		frequency: frequency,
		offset:    offset,
	}
	return field.evaluate, field.gradient, nil
}

func (f *gyroidField) evaluate(point *mat.VecDense) float64 {
	if f == nil || point == nil || point.Len() < 3 {
		return math.NaN()
	}

	x := f.frequency * point.AtVec(0)
	y := f.frequency * point.AtVec(1)
	z := f.frequency * point.AtVec(2)
	sx, cx := math.Sincos(x)
	sy, cy := math.Sincos(y)
	sz, cz := math.Sincos(z)
	return sx*cy + sy*cz + sz*cx - f.offset
}

func (f *gyroidField) gradient(point, res *mat.VecDense) *mat.VecDense {
	if f == nil || point == nil || point.Len() < 3 {
		return nil
	}
	if res == nil || res.Len() != point.Len() {
		res = mat.NewVecDense(point.Len(), nil)
	} else {
		res.Zero()
	}

	x := f.frequency * point.AtVec(0)
	y := f.frequency * point.AtVec(1)
	z := f.frequency * point.AtVec(2)
	sx, cx := math.Sincos(x)
	sy, cy := math.Sincos(y)
	sz, cz := math.Sincos(z)

	res.SetVec(0, f.frequency*(cx*cy-sz*sx))
	res.SetVec(1, f.frequency*(-sx*sy+cy*cz))
	res.SetVec(2, f.frequency*(-sy*sz+cz*cx))
	return res
}
