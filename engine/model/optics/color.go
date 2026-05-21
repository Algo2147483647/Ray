package optics

import "gonum.org/v1/gonum/mat"

func SpectralRayToXYZ(color *mat.VecDense, ray *Ray) *mat.VecDense {
	if ray == nil || ray.WaveLength <= 0 {
		return mat.NewVecDense(3, nil)
	}
	power := ray.SpectralPower
	xyz := SpectralPowerToXYZ(ray.WaveLength, ray.WavelengthPDF, power)
	compatibility := ray.RGBCompatibility
	if compatibility == nil {
		compatibility = color
	}
	if !ray.SpectralPath {
		if compatibility == nil || compatibility.Len() < 3 {
			return LinearSRGBToXYZ(power, power, power)
		}
		return LinearSRGBToXYZ(
			power*compatibility.AtVec(0),
			power*compatibility.AtVec(1),
			power*compatibility.AtVec(2),
		)
	}
	if compatibility == nil || compatibility.Len() < 3 || isWhiteRGB(compatibility) {
		return xyz
	}
	r, g, b := XYZToLinearSRGB(xyz.AtVec(0), xyz.AtVec(1), xyz.AtVec(2))
	return LinearSRGBToXYZ(
		r*compatibility.AtVec(0),
		g*compatibility.AtVec(1),
		b*compatibility.AtVec(2),
	)
}

func LinearSRGBToXYZ(r, g, b float64) *mat.VecDense {
	return mat.NewVecDense(3, []float64{
		0.4124564*r + 0.3575761*g + 0.1804375*b,
		0.2126729*r + 0.7151522*g + 0.0721750*b,
		0.0193339*r + 0.1191920*g + 0.9503041*b,
	})
}

func XYZToLinearSRGB(x, y, z float64) (float64, float64, float64) {
	return 3.2404542*x - 1.5371385*y - 0.4985314*z,
		-0.9692660*x + 1.8760108*y + 0.0415560*z,
		0.0556434*x - 0.2040259*y + 1.0572252*z
}

func isWhiteRGB(v *mat.VecDense) bool {
	const eps = 1e-9
	return v.Len() >= 3 &&
		abs(v.AtVec(0)-1) <= eps &&
		abs(v.AtVec(1)-1) <= eps &&
		abs(v.AtVec(2)-1) <= eps
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
