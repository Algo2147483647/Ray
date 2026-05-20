package optics

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

const (
	WavelengthMin = 380.0
	WavelengthMax = 750.0
)

var d65WhiteXYZ = [3]float64{0.95047, 1.0, 1.08883}

func UniformWavelengthPDF() float64 {
	return 1 / (WavelengthMax - WavelengthMin)
}

func RGBWeight(wavelength float64) *mat.VecDense {
	rgb := WavelengthToLinearSRGB(wavelength)
	white := spectralWhitePoint
	return mat.NewVecDense(3, []float64{
		safeDivide(rgb.AtVec(0), white[0]),
		safeDivide(rgb.AtVec(1), white[1]),
		safeDivide(rgb.AtVec(2), white[2]),
	})
}

func WavelengthToRGB(wavelength float64) *mat.VecDense {
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

func WavelengthToNormalizedXYZ(wavelength, pdf float64) *mat.VecDense {
	xyz := WavelengthToXYZ(wavelength)
	white := spectralXYZWhitePoint
	pdfScale := wavelengthPDFScale(pdf)

	return mat.NewVecDense(3, []float64{
		safeDivide(xyz.AtVec(0)*pdfScale*d65WhiteXYZ[0], white[0]),
		safeDivide(xyz.AtVec(1)*pdfScale*d65WhiteXYZ[1], white[1]),
		safeDivide(xyz.AtVec(2)*pdfScale*d65WhiteXYZ[2], white[2]),
	})
}

func SpectralPowerToXYZ(wavelength, pdf, power float64) *mat.VecDense {
	xyz := WavelengthToNormalizedXYZ(wavelength, pdf)
	return mat.NewVecDense(3, []float64{
		xyz.AtVec(0) * power,
		xyz.AtVec(1) * power,
		xyz.AtVec(2) * power,
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
var spectralXYZWhitePoint = computeSpectralXYZWhitePoint()

func computeSpectralWhitePoint() [3]float64 {
	const steps = 2048
	var sum [3]float64
	for i := 0; i < steps; i++ {
		t := (float64(i) + 0.5) / steps
		wavelength := WavelengthMin + t*(WavelengthMax-WavelengthMin)
		rgb := WavelengthToRGB(wavelength)
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

func computeSpectralXYZWhitePoint() [3]float64 {
	const steps = 2048
	var sum [3]float64
	for i := 0; i < steps; i++ {
		t := (float64(i) + 0.5) / steps
		wavelength := WavelengthMin + t*(WavelengthMax-WavelengthMin)
		xyz := WavelengthToXYZ(wavelength)
		sum[0] += xyz.AtVec(0)
		sum[1] += xyz.AtVec(1)
		sum[2] += xyz.AtVec(2)
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

func wavelengthPDFScale(pdf float64) float64 {
	if pdf <= 0 {
		return 0
	}
	return 1 / (pdf * (WavelengthMax - WavelengthMin))
}
