package factory

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type metaballField struct {
	k     float64
	iso   float64
	balls []metaball
}

type metaball struct {
	weight float64
	center [3]float64
}

func parseImplicitMetaballsField(
	fieldDef map[string]interface{},
) (
	func(*mat.VecDense) float64,
	func(point, res *mat.VecDense) *mat.VecDense,
	error,
) {
	k, err := utils.RequiredPositiveFloat(fieldDef, "k")
	if err != nil {
		return nil, nil, err
	}
	iso, err := utils.RequiredFloat64Field(fieldDef, "iso")
	if err != nil {
		return nil, nil, err
	}
	balls, err := parseMetaballs(fieldDef)
	if err != nil {
		return nil, nil, err
	}

	field := &metaballField{
		k:     k,
		iso:   iso,
		balls: balls,
	}
	return field.evaluate, field.gradient, nil
}

func parseMetaballs(fieldDef map[string]interface{}) ([]metaball, error) {
	raw, ok := fieldDef["balls"]
	if !ok {
		return nil, fmt.Errorf(`missing required field "balls"`)
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected array, got %T", "balls", raw)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("field %q must not be empty", "balls")
	}

	balls := make([]metaball, len(items))
	for i, item := range items {
		def, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("balls[%d]: expected object, got %T", i, item)
		}
		weight, err := utils.RequiredFloat64Field(def, "weight")
		if err != nil {
			return nil, fmt.Errorf("balls[%d].%w", i, err)
		}
		center, err := utils.RequiredFloat64SliceField(def, "center", 3)
		if err != nil {
			return nil, fmt.Errorf("balls[%d].%w", i, err)
		}
		balls[i] = metaball{
			weight: weight,
			center: [3]float64{center[0], center[1], center[2]},
		}
	}
	return balls, nil
}

func (f *metaballField) evaluate(point *mat.VecDense) float64 {
	if f == nil || point == nil || point.Len() < 3 {
		return math.NaN()
	}
	x, y, z := point.AtVec(0), point.AtVec(1), point.AtVec(2)
	sum := -f.iso
	for _, ball := range f.balls {
		dx := x - ball.center[0]
		dy := y - ball.center[1]
		dz := z - ball.center[2]
		value := math.Exp(-f.k * (dx*dx + dy*dy + dz*dz))
		sum += ball.weight * value
	}
	return sum
}

func (f *metaballField) gradient(point, res *mat.VecDense) *mat.VecDense {
	if f == nil || point == nil || point.Len() < 3 {
		return nil
	}
	if res == nil || res.Len() != point.Len() {
		res = mat.NewVecDense(point.Len(), nil)
	} else {
		res.Zero()
	}

	x, y, z := point.AtVec(0), point.AtVec(1), point.AtVec(2)
	gx, gy, gz := 0.0, 0.0, 0.0
	for _, ball := range f.balls {
		dx := x - ball.center[0]
		dy := y - ball.center[1]
		dz := z - ball.center[2]
		value := ball.weight * math.Exp(-f.k*(dx*dx+dy*dy+dz*dz)) * -2 * f.k
		gx += value * dx
		gy += value * dy
		gz += value * dz
	}
	res.SetVec(0, gx)
	res.SetVec(1, gy)
	res.SetVec(2, gz)
	return res
}
