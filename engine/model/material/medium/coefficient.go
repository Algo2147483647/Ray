package medium

type Coefficient interface {
	Eval(ctx WavelengthContext) float64
}

type ConstantCoefficient float64

func (c ConstantCoefficient) Eval(WavelengthContext) float64 {
	return float64(c)
}
