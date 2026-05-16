package optics

import (
	"github.com/Algo2147483647/golang_toolkit/image"
	"github.com/Algo2147483647/ray/engine/go/internal/utils"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
)

type Ray struct {
	Origin          *mat.VecDense `json:"origin"`
	Direction       *mat.VecDense `json:"direction"`
	Color           *mat.VecDense `json:"color"`
	WaveLength      float64       `json:"wave_length"` // (nm)
	WavelengthPDF   float64       `json:"wavelength_pdf"`
	RefractionIndex float64       `json:"refraction_index"`
	DebugSwitch     bool          `json:"debug_switch"`
}

const (
	WavelengthMin = 380.0 // 最小波长(nm)
	WavelengthMax = 750.0 // 最大波长(nm)
)

var DebugRayTraces []map[string]interface{}

func (r *Ray) Init() {
	if r.Origin == nil {
		r.Origin = mat.NewVecDense(utils.Dimension, nil)
	} else {
		r.Origin.Zero()
	}

	if r.Direction == nil {
		r.Direction = mat.NewVecDense(utils.Dimension, nil)
	} else {
		r.Direction.Zero()
	}

	if r.Color == nil {
		r.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	}

	r.RefractionIndex = 1
	r.WaveLength = 0
	r.WavelengthPDF = 0
	r.DebugSwitch = false
}

func (ray *Ray) ConvertToMonochrome() {
	ray.SampleWavelength(rand.Float64())
}

func (ray *Ray) SetMonochrome(wavelength float64) {
	ray.SetSpectralWavelength(wavelength)
}

func (ray *Ray) SampleWavelength(sample float64) {
	sample = math.Max(1e-6, math.Min(1-1e-6, sample))
	ray.SetSpectralWavelength(WavelengthMin + sample*(WavelengthMax-WavelengthMin))
}

func (ray *Ray) SetSpectralWavelength(wavelength float64) {
	wavelength = math.Max(WavelengthMin+1e-6, math.Min(WavelengthMax-1e-6, wavelength))
	ray.WaveLength = wavelength
	ray.WavelengthPDF = UniformWavelengthPDF()

	baseColor := SpectralRGBWeight(wavelength)

	ray.Color.SetVec(0, baseColor.AtVec(0)*ray.Color.AtVec(0))
	ray.Color.SetVec(1, baseColor.AtVec(1)*ray.Color.AtVec(1))
	ray.Color.SetVec(2, baseColor.AtVec(2)*ray.Color.AtVec(2))
}

func UniformWavelengthPDF() float64 {
	return 1 / (WavelengthMax - WavelengthMin)
}

func SpectralRGBWeight(wavelength float64) *mat.VecDense {
	rgb := WaveLengthToRGB(wavelength)
	white := spectralWhitePoint
	return mat.NewVecDense(3, []float64{
		safeDivide(rgb.AtVec(0), white[0]),
		safeDivide(rgb.AtVec(1), white[1]),
		safeDivide(rgb.AtVec(2), white[2]),
	})
}

func WaveLengthToRGB(wavelength float64) *mat.VecDense {
	if wavelength >= WavelengthMax || wavelength <= WavelengthMin {
		return mat.NewVecDense(3, nil)
	}
	t := (wavelength - WavelengthMin) / (WavelengthMax - WavelengthMin)
	r, g, b := image.SpectrumFiveDivided(t)
	return mat.NewVecDense(3, []float64{
		math.Max(0, math.Min(1, r)),
		math.Max(0, math.Min(1, g)),
		math.Max(0, math.Min(1, b)),
	})
}

var spectralWhitePoint = computeSpectralWhitePoint()

func computeSpectralWhitePoint() [3]float64 {
	const steps = 2048
	var sum [3]float64
	for i := 0; i < steps; i++ {
		t := (float64(i) + 0.5) / steps
		wavelength := WavelengthMin + t*(WavelengthMax-WavelengthMin)
		rgb := WaveLengthToRGB(wavelength)
		sum[0] += rgb.AtVec(0)
		sum[1] += rgb.AtVec(1)
		sum[2] += rgb.AtVec(2)
	}
	return [3]float64{
		sum[0] / steps,
		sum[1] / steps,
		sum[2] / steps,
	}
}

func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
