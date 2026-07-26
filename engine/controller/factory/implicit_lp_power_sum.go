package factory

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

const defaultLPPowerSumGradientEps = 1e-12

type lpPowerSumField struct {
	p           float64
	radius      float64
	gradientEps float64
}

func parseImplicitLPPowerSumField(
	fieldDef map[string]interface{},
) (
	func(*mat.VecDense) float64,
	func(point, res *mat.VecDense) *mat.VecDense,
	error,
) {
	p, err := utils.RequiredPositiveFloat(fieldDef, "power")
	if err != nil {
		return nil, nil, err
	}
	radius, err := utils.RequiredPositiveFloat(fieldDef, "radius")
	if err != nil {
		return nil, nil, err
	}
	gradientEps := defaultLPPowerSumGradientEps
	if value, ok, err := utils.OptionalFloat64Field(fieldDef, "gradient_epsilon"); err != nil {
		return nil, nil, err
	} else if ok {
		if value <= 0 {
			return nil, nil, fmt.Errorf("gradient_epsilon must be > 0")
		}
		gradientEps = value
	}

	field := &lpPowerSumField{
		p:           p,
		radius:      radius,
		gradientEps: gradientEps,
	}
	return field.evaluate, field.gradient, nil
}

func (f *lpPowerSumField) evaluate(point *mat.VecDense) float64 {
	if f == nil || point == nil || point.Len() < 3 {
		return math.NaN()
	}
	return lpPowerValue(point.AtVec(0), f.p) +
		lpPowerValue(point.AtVec(1), f.p) +
		lpPowerValue(point.AtVec(2), f.p) -
		f.radius
}

func (f *lpPowerSumField) gradient(point, res *mat.VecDense) *mat.VecDense {
	if f == nil || point == nil || point.Len() < 3 {
		return nil
	}
	if res == nil || res.Len() != point.Len() {
		res = mat.NewVecDense(point.Len(), nil)
	} else {
		res.Zero()
	}

	res.SetVec(0, lpPowerDerivative(point.AtVec(0), f.p, f.gradientEps))
	res.SetVec(1, lpPowerDerivative(point.AtVec(1), f.p, f.gradientEps))
	res.SetVec(2, lpPowerDerivative(point.AtVec(2), f.p, f.gradientEps))
	return res
}

func lpPowerValue(value, p float64) float64 {
	absValue := math.Abs(value)
	if nearlyEqual(p, 1) {
		return absValue
	}
	if nearlyEqual(p, 2) {
		return absValue * absValue
	}
	if nearlyEqual(p, 2.0/3.0) {
		cubeRoot := math.Cbrt(absValue)
		return cubeRoot * cubeRoot
	}
	return math.Pow(absValue, p)
}

func lpPowerDerivative(value, p, eps float64) float64 {
	if value == 0 {
		return 0
	}
	sign := 1.0
	if value < 0 {
		sign = -1.0
	}
	absValue := math.Abs(value)
	if absValue < eps {
		absValue = eps
	}
	if nearlyEqual(p, 1) {
		return sign
	}
	if nearlyEqual(p, 2) {
		return 2 * value
	}
	if nearlyEqual(p, 2.0/3.0) {
		return sign * p / math.Cbrt(absValue)
	}
	return sign * p * math.Pow(absValue, p-1)
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-12
}
