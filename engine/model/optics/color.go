package optics

func SpectralRayToXYZ(color RGB, ray *Ray) XYZ {
	if ray == nil || ray.WaveLength <= 0 {
		return XYZ{}
	}
	power := ray.SpectralPower
	xyz := SpectralPowerToXYZ(ray.WaveLength, ray.WavelengthPDF, power)
	compatibility := ray.RGBCompatibility
	if !ray.SpectralPath && ray.RGBCompatibilityPath {
		return LinearSRGBToXYZ(
			power*compatibility[0],
			power*compatibility[1],
			power*compatibility[2],
		)
	}
	if !ray.RGBCompatibilityPath {
		return xyz
	}
	r, g, b := XYZToLinearSRGB(xyz[0], xyz[1], xyz[2])
	return LinearSRGBToXYZ(
		r*compatibility[0],
		g*compatibility[1],
		b*compatibility[2],
	)
}

func SpectralRayToScalar(color RGB, ray *Ray) float64 {
	if ray == nil || ray.WaveLength <= 0 {
		return 0
	}
	power := ray.SpectralPower
	compatibility := ray.RGBCompatibility
	if !ray.RGBCompatibilityPath {
		return power
	}
	return power * NewRGBSpectrum(
		compatibility[0],
		compatibility[1],
		compatibility[2],
	).RGBPowerAtWavelength(ray.WaveLength)
}

func LinearSRGBToXYZ(r, g, b float64) XYZ {
	return XYZ{
		0.4124564*r + 0.3575761*g + 0.1804375*b,
		0.2126729*r + 0.7151522*g + 0.0721750*b,
		0.0193339*r + 0.1191920*g + 0.9503041*b,
	}
}

func XYZToLinearSRGB(x, y, z float64) (float64, float64, float64) {
	return 3.2404542*x - 1.5371385*y - 0.4985314*z,
		-0.9692660*x + 1.8760108*y + 0.0415560*z,
		0.0556434*x - 0.2040259*y + 1.0572252*z
}

func XYZToACEScg(x, y, z float64) (float64, float64, float64) {
	return 1.6410233797*x - 0.3248032942*y - 0.2364246952*z,
		-0.6636628587*x + 1.6153315917*y + 0.0167563477*z,
		0.0117218943*x - 0.0082844420*y + 0.9883948585*z
}

func ACEScgToXYZ(r, g, b float64) XYZ {
	return XYZ{
		0.6624541811*r + 0.1340042065*g + 0.1561876870*b,
		0.2722287168*r + 0.6740817658*g + 0.0536895174*b,
		-0.0055746495*r + 0.0040607335*g + 1.0103391003*b,
	}
}

func LinearSRGBToACEScg(r, g, b float64) (float64, float64, float64) {
	xyz := LinearSRGBToXYZ(r, g, b)
	return XYZToACEScg(xyz[0], xyz[1], xyz[2])
}

func ACEScgToLinearSRGB(r, g, b float64) (float64, float64, float64) {
	xyz := ACEScgToXYZ(r, g, b)
	return XYZToLinearSRGB(xyz[0], xyz[1], xyz[2])
}
