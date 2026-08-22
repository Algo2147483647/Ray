package film

import (
	"fmt"
	"image"
	"image/color"
	"math"

	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
)

type ToneMapping string

type ColorSpace string

const (
	ToneMappingLinear       ToneMapping = "linear"
	ToneMappingReinhard     ToneMapping = "reinhard"
	ToneMappingACES         ToneMapping = "aces"
	ToneMappingSpectralTanh ToneMapping = "spectral_tanh"
)

const (
	ColorSpaceLinearSRGB ColorSpace = "linear_srgb"
	ColorSpaceXYZ        ColorSpace = "xyz"
	ColorSpaceACEScg     ColorSpace = "acescg"
)

type ImageOptions struct {
	Exposure    float64
	ToneMapping ToneMapping
	TanhOmega   float64
	Gamma       float64
	ColorSpace  ColorSpace
}

func ToImage(film *modelcamera.Film, options ImageOptions) (*image.RGBA, error) {
	if film == nil || !film.HasSpectralBins() || film.ElementCount() == 0 {
		return nil, fmt.Errorf("cannot image an empty spectral Film")
	}
	options = normalizeImageOptions(options)
	if err := validateImageOptions(options); err != nil {
		return nil, err
	}
	shape := film.Shape
	if len(shape) < 2 {
		return nil, fmt.Errorf("image output requires a Film rank of at least 2")
	}
	width, height := shape[0], shape[1]
	slices := 1
	for _, extent := range shape[2:] {
		slices *= extent
	}
	atlasCols := 1
	if slices > 1 {
		atlasCols = int(math.Ceil(math.Sqrt(float64(slices))))
	}
	atlasRows := (slices + atlasCols - 1) / atlasCols
	output := image.NewRGBA(image.Rect(0, 0, width*atlasCols, height*atlasRows))
	whiteY := filmSpectralWhiteY(film)
	if whiteY <= 0 {
		return nil, fmt.Errorf("Film wavelength range has no visible CIE Y response")
	}
	for pixel := 0; pixel < film.ElementCount(); pixel++ {
		encodeOptions := options
		var x, y, z float64
		if options.ToneMapping == ToneMappingSpectralTanh {
			var brightness float64
			var err error
			x, y, z, brightness, err = spectralXYZAndBrightnessAt(film, pixel, whiteY)
			if err != nil {
				return nil, err
			}
			spectralScale := spectralTanhScale(brightness, options.Exposure, options.TanhOmega)
			x *= spectralScale
			y *= spectralScale
			z *= spectralScale
			// Exposure and tone mapping have already been applied as one scalar
			// to the complete physical spectrum.
			encodeOptions.Exposure = 1
			encodeOptions.ToneMapping = ToneMappingLinear
		} else {
			x, y, z = spectralXYZAt(film, pixel, whiteY)
		}
		r, g, b := xyzToOutputRGB(x, y, z, options.ColorSpace)
		if options.ToneMapping == ToneMappingSpectralTanh {
			r, g, b = fitRGBWithoutChannelClipping(r, g, b)
		}
		coords := film.SpectralBins[0].GetCoordinates(pixel)
		slice := flattenedSliceIndex(coords[2:], shape[2:])
		output.Set(
			coords[0]+(slice%atlasCols)*width,
			coords[1]+(slice/atlasCols)*height,
			color.RGBA{
				R: encodeOutputChannel(r, encodeOptions),
				G: encodeOutputChannel(g, encodeOptions),
				B: encodeOutputChannel(b, encodeOptions),
				A: 255,
			},
		)
	}
	return output, nil
}

func validateImageOptions(options ImageOptions) error {
	if math.IsNaN(options.Exposure) || math.IsInf(options.Exposure, 0) || options.Exposure < 0 {
		return fmt.Errorf("exposure must be finite and >= 0")
	}
	if math.IsNaN(options.Gamma) || math.IsInf(options.Gamma, 0) || options.Gamma <= 0 {
		return fmt.Errorf("gamma must be finite and > 0")
	}
	switch options.ToneMapping {
	case ToneMappingLinear, ToneMappingReinhard, ToneMappingACES, ToneMappingSpectralTanh:
	default:
		return fmt.Errorf("unsupported tone mapping %q", options.ToneMapping)
	}
	if math.IsNaN(options.TanhOmega) || math.IsInf(options.TanhOmega, 0) || options.TanhOmega <= 0 {
		return fmt.Errorf("tanh omega must be finite and > 0")
	}
	switch options.ColorSpace {
	case ColorSpaceLinearSRGB, ColorSpaceXYZ, ColorSpaceACEScg:
	default:
		return fmt.Errorf("unsupported output color space %q", options.ColorSpace)
	}
	return nil
}

func spectralXYZAt(film *modelcamera.Film, pixel int, whiteY float64) (float64, float64, float64) {
	var x, y, z float64
	for bin := range film.SpectralBins {
		wavelength := film.SpectralBinCenterNM(bin)
		cx, cy, cz := cie1931Approximation(wavelength)
		// Film bins store Monte Carlo contribution mass, not a sampled spectral
		// density. Summation therefore needs no extra bin-width factor. One Y
		// normalization is shared by X, Y, and Z to preserve chromaticity.
		scale := film.SpectralBins[bin].Data[pixel] / whiteY
		x += math.Max(0, cx) * scale
		y += math.Max(0, cy) * scale
		z += math.Max(0, cz) * scale
	}
	return x, y, z
}

func spectralXYZAndBrightnessAt(film *modelcamera.Film, pixel int, whiteY float64) (float64, float64, float64, float64, error) {
	var x, y, z, brightness float64
	for bin := range film.SpectralBins {
		value := film.SpectralBins[bin].Data[pixel]
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return 0, 0, 0, 0, fmt.Errorf("spectral_tanh requires finite non-negative spectral values; bin %d pixel %d is %g", bin, pixel, value)
		}
		brightness += value
		wavelength := film.SpectralBinCenterNM(bin)
		cx, cy, cz := cie1931Approximation(wavelength)
		scale := value / whiteY
		x += math.Max(0, cx) * scale
		y += math.Max(0, cy) * scale
		z += math.Max(0, cz) * scale
	}
	if math.IsInf(brightness, 0) {
		return 0, 0, 0, 0, fmt.Errorf("spectral brightness overflow at pixel %d", pixel)
	}
	return x, y, z, brightness, nil
}

// spectralTanhScale maps the sum x of all bins to tanh(omega*exposure*x).
// Returning one shared scale preserves all spectral-bin ratios in the pixel.
func spectralTanhScale(brightness, exposure, omega float64) float64 {
	if brightness <= 0 {
		return 0
	}
	targetBrightness := math.Tanh(omega * exposure * brightness)
	return targetBrightness / brightness
}

// fitRGBWithoutChannelClipping uses one scale for all positive RGB channels.
// It avoids hue shifts from independently clipping over-range channels. Negative
// components are still outside the representable gamut of an RGB PNG.
func fitRGBWithoutChannelClipping(r, g, b float64) (float64, float64, float64) {
	maximum := math.Max(r, math.Max(g, b))
	if maximum > 1 {
		scale := 1 / maximum
		return r * scale, g * scale, b * scale
	}
	return r, g, b
}

func filmSpectralWhiteY(film *modelcamera.Film) float64 {
	if film == nil || len(film.SpectralBins) == 0 {
		return 0
	}
	var sum float64
	for bin := range film.SpectralBins {
		_, y, _ := cie1931Approximation(film.SpectralBinCenterNM(bin))
		sum += math.Max(0, y)
	}
	return sum / float64(len(film.SpectralBins))
}

func flattenedSliceIndex(coords, shape []int) int {
	index, stride := 0, 1
	for i := 0; i < len(coords) && i < len(shape); i++ {
		index += coords[i] * stride
		stride *= shape[i]
	}
	return index
}

func normalizeImageOptions(options ImageOptions) ImageOptions {
	if options.Exposure == 0 {
		options.Exposure = 1
	}
	if options.ToneMapping == "" {
		options.ToneMapping = ToneMappingLinear
	}
	if options.TanhOmega == 0 {
		options.TanhOmega = 1
	}
	if options.Gamma == 0 {
		options.Gamma = 1
	}
	if options.ColorSpace == "" {
		options.ColorSpace = ColorSpaceLinearSRGB
	}
	return options
}

func encodeOutputChannel(value float64, options ImageOptions) uint8 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0
	}
	value *= options.Exposure
	switch options.ToneMapping {
	case ToneMappingReinhard:
		value /= 1 + value
	case ToneMappingACES:
		value = value * (2.51*value + 0.03) / (value*(2.43*value+0.59) + 0.14)
	}
	value = math.Max(0, math.Min(1, value))
	if options.Gamma > 0 && options.Gamma != 1 {
		value = math.Pow(value, 1/options.Gamma)
	}
	return uint8(math.Max(0, math.Min(1, value))*255 + 0.5)
}

func xyzToLinearSRGB(x, y, z float64) (float64, float64, float64) {
	return 3.2404542*x - 1.5371385*y - 0.4985314*z,
		-0.9692660*x + 1.8760108*y + 0.0415560*z,
		0.0556434*x - 0.2040259*y + 1.0572252*z
}

func xyzToOutputRGB(x, y, z float64, workingSpace ColorSpace) (float64, float64, float64) {
	switch workingSpace {
	case ColorSpaceXYZ:
		return xyzToLinearSRGB(x, y, z)
	case ColorSpaceACEScg:
		r, g, b := xyzToACEScg(x, y, z)
		x, y, z = acescgToXYZ(r, g, b)
		return xyzToLinearSRGB(x, y, z)
	default:
		return xyzToLinearSRGB(x, y, z)
	}
}

func xyzToACEScg(x, y, z float64) (float64, float64, float64) {
	return 1.6410233797*x - 0.3248032942*y - 0.2364246952*z,
		-0.6636628587*x + 1.6153315917*y + 0.0167563477*z,
		0.0117218943*x - 0.0082844420*y + 0.9883948585*z
}

func acescgToXYZ(r, g, b float64) (float64, float64, float64) {
	return 0.6624541811*r + 0.1340042065*g + 0.1561876870*b,
		0.2722287168*r + 0.6740817658*g + 0.0536895174*b,
		-0.0055746495*r + 0.0040607335*g + 1.0103391003*b
}

func cie1931Approximation(wavelength float64) (float64, float64, float64) {
	x1 := gaussianPiece(wavelength, 442.0, 0.0624, 0.0374)
	x2 := gaussianPiece(wavelength, 599.8, 0.0264, 0.0323)
	x3 := gaussianPiece(wavelength, 501.1, 0.0490, 0.0382)
	y1 := gaussianPiece(wavelength, 568.8, 0.0213, 0.0247)
	y2 := gaussianPiece(wavelength, 530.9, 0.0613, 0.0322)
	z1 := gaussianPiece(wavelength, 437.0, 0.0845, 0.0278)
	z2 := gaussianPiece(wavelength, 459.0, 0.0385, 0.0725)
	return 0.362*x1 + 1.056*x2 - 0.065*x3,
		0.821*y1 + 0.286*y2,
		1.217*z1 + 0.681*z2
}

func gaussianPiece(wavelength, center, leftScale, rightScale float64) float64 {
	scale := rightScale
	if wavelength < center {
		scale = leftScale
	}
	t := (wavelength - center) * scale
	return math.Exp(-0.5 * t * t)
}
