package optics

import (
	"github.com/Algo2147483647/ray/engine/go/internal/material/medium"
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
	MediumStack     medium.Stack  `json:"-"`
}

const (
	WavelengthMin = 380.0 // Minimum wavelength (nm)
	WavelengthMax = 750.0 // Maximum wavelength (nm)
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
	} else if r.Color.Len() == 3 {
		r.Color.SetVec(0, 1)
		r.Color.SetVec(1, 1)
		r.Color.SetVec(2, 1)
	} else {
		r.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	}

	r.RefractionIndex = 1
	r.MediumStack.Reset(0)
	r.WaveLength = 0
	r.WavelengthPDF = 0
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

func (ray *Ray) DisableSpectralSampling() {
	ray.WaveLength = 0
	ray.WavelengthPDF = 0
	if ray.Color == nil {
		ray.Color = mat.NewVecDense(3, []float64{1, 1, 1})
		return
	}
	ray.Color.SetVec(0, 1)
	ray.Color.SetVec(1, 1)
	ray.Color.SetVec(2, 1)
}

func UniformWavelengthPDF() float64 {
	return 1 / (WavelengthMax - WavelengthMin)
}

func SpectralRGBWeight(wavelength float64) *mat.VecDense {
	rgb := WavelengthToLinearSRGB(wavelength)
	white := spectralWhitePoint
	return mat.NewVecDense(3, []float64{
		safeDivide(rgb.AtVec(0), white[0]),
		safeDivide(rgb.AtVec(1), white[1]),
		safeDivide(rgb.AtVec(2), white[2]),
	})
}

func WaveLengthToRGB(wavelength float64) *mat.VecDense {
	return WavelengthToLinearSRGB(wavelength)
}

func WavelengthToXYZ(wavelength float64) *mat.VecDense {
	if wavelength >= WavelengthMax || wavelength <= WavelengthMin {
		return mat.NewVecDense(3, nil)
	}
	x, y, z := cie1931Approximation(wavelength)
	return mat.NewVecDense(3, []float64{
		math.Max(0, x),
		math.Max(0, y),
		math.Max(0, z),
	})
}

func WavelengthToLinearSRGB(wavelength float64) *mat.VecDense {
	xyz := WavelengthToXYZ(wavelength)
	r, g, b := xyzToLinearSRGB(xyz.AtVec(0), xyz.AtVec(1), xyz.AtVec(2))
	return mat.NewVecDense(3, []float64{
		math.Max(0, r),
		math.Max(0, g),
		math.Max(0, b),
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

func cie1931Approximation(wavelength float64) (float64, float64, float64) {
	x1 := gaussianPiece(wavelength, 442.0, 0.0624, 0.0374)
	x2 := gaussianPiece(wavelength, 599.8, 0.0264, 0.0323)
	x3 := gaussianPiece(wavelength, 501.1, 0.0490, 0.0382)
	x := 0.362*x1 + 1.056*x2 - 0.065*x3

	y1 := gaussianPiece(wavelength, 568.8, 0.0213, 0.0247)
	y2 := gaussianPiece(wavelength, 530.9, 0.0613, 0.0322)
	y := 0.821*y1 + 0.286*y2

	z1 := gaussianPiece(wavelength, 437.0, 0.0845, 0.0278)
	z2 := gaussianPiece(wavelength, 459.0, 0.0385, 0.0725)
	z := 1.217*z1 + 0.681*z2

	return x, y, z
}

func gaussianPiece(wavelength, center, leftScale, rightScale float64) float64 {
	scale := rightScale
	if wavelength < center {
		scale = leftScale
	}
	t := (wavelength - center) * scale
	return math.Exp(-0.5 * t * t)
}

func xyzToLinearSRGB(x, y, z float64) (float64, float64, float64) {
	return 3.2404542*x - 1.5371385*y - 0.4985314*z,
		-0.9692660*x + 1.8760108*y + 0.0415560*z,
		0.0556434*x - 0.2040259*y + 1.0572252*z
}

func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
