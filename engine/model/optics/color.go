package optics

func SpectralRayToScalar(ray *Ray) float64 {
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

func XYZToLinearSRGB(x, y, z float64) (float64, float64, float64) {
	return 3.2404542*x - 1.5371385*y - 0.4985314*z,
		-0.9692660*x + 1.8760108*y + 0.0415560*z,
		0.0556434*x - 0.2040259*y + 1.0572252*z
}
