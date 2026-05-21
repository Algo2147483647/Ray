package medium

type Coefficient interface {
	Eval(ctx WavelengthContext) CoefficientSpectrum
}

type CoefficientSpectrum struct {
	RGB     [3]float64
	Samples []float64
}

func NewRGBCoefficientSpectrum(r, g, b float64) CoefficientSpectrum {
	return CoefficientSpectrum{RGB: [3]float64{r, g, b}}
}

func NewSampledCoefficientSpectrum(samples []float64) CoefficientSpectrum {
	return CoefficientSpectrum{Samples: append([]float64(nil), samples...)}
}

func (s CoefficientSpectrum) HasSamples() bool {
	return len(s.Samples) > 0
}

func (s CoefficientSpectrum) Sample(i int) float64 {
	if i < 0 || i >= len(s.Samples) {
		return 0
	}
	return s.Samples[i]
}

func (s CoefficientSpectrum) RGBChannel(i int) float64 {
	if i < 0 || i >= len(s.RGB) {
		return 0
	}
	return s.RGB[i]
}

type ConstantCoefficient float64

func (c ConstantCoefficient) Eval(WavelengthContext) CoefficientSpectrum {
	v := float64(c)
	return NewRGBCoefficientSpectrum(v, v, v)
}
