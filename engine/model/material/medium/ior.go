package medium

import "math"

const (
	DefaultWavelengthNM = 550.0
	WavelengthMinNM     = 380.0
	WavelengthMaxNM     = 750.0
)

type Model interface {
	Evaluate(wavelengthNM float64) float64
	IsDispersive() bool
}

type Constant struct {
	Eta float64
}

func NewConstant(eta float64) Constant {
	return Constant{Eta: eta}
}

func (c Constant) Evaluate(float64) float64 {
	return c.Eta
}

func (c Constant) IsDispersive() bool {
	return false
}

type Cauchy struct {
	A float64
	B float64
	C float64
}

func NewCauchy(a, b, c float64) Cauchy {
	return Cauchy{A: a, B: b, C: c}
}

func (c Cauchy) Evaluate(wavelengthNM float64) float64 {
	if wavelengthNM <= 0 || math.IsNaN(wavelengthNM) || math.IsInf(wavelengthNM, 0) {
		wavelengthNM = DefaultWavelengthNM
	}
	wavelengthUM := wavelengthNM / 1000
	wavelength2 := wavelengthUM * wavelengthUM
	return c.A + c.B/wavelength2 + c.C/(wavelength2*wavelength2)
}

func (c Cauchy) IsDispersive() bool {
	return c.B != 0 || c.C != 0
}

func IsValidEta(eta float64) bool {
	return eta > 0 && !math.IsNaN(eta) && !math.IsInf(eta, 0)
}
