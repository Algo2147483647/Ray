package optics

func SpectralRayToScalar(ray *Ray) float64 {
	if ray == nil || ray.Path.Wavelength == nil || !ray.Path.Radiance.HasSamples() {
		return 0
	}
	return ray.Path.Radiance.Sample(0)
}

func XYZToLinearSRGB(x, y, z float64) (float64, float64, float64) {
	return 3.2404542*x - 1.5371385*y - 0.4985314*z,
		-0.9692660*x + 1.8760108*y + 0.0415560*z,
		0.0556434*x - 0.2040259*y + 1.0572252*z
}
