package camera

import (
	"encoding/binary"
	"errors"
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"reflect"
)

type Film struct {
	Data          [3]math_lib.Tensor[float64] `json:"data"`
	Samples       int64                       `json:"samples"`
	ColorSpace    FilmColorSpace              `json:"color_space"`
	SpectralBins  []math_lib.Tensor[float64]  `json:"spectral_bins,omitempty"`
	SpectralMinNM float64                     `json:"spectral_min_nm,omitempty"`
	SpectralMaxNM float64                     `json:"spectral_max_nm,omitempty"`
}

type FilmColorSpace string

const (
	FilmColorSpaceLinearSRGB FilmColorSpace = "linear_srgb"
	FilmColorSpaceXYZ        FilmColorSpace = "xyz"
)

type ToneMapping string

const (
	ToneMappingLinear   ToneMapping = "linear"
	ToneMappingReinhard ToneMapping = "reinhard"
	ToneMappingACES     ToneMapping = "aces"
)

type ImageOptions struct {
	Exposure    float64
	ToneMapping ToneMapping
	Gamma       float64
}

type SpectralSample struct {
	WavelengthNM float64
	Value        float64
}

func NewFilm(width ...int) *Film {
	shape := make([]int, len(width))
	copy(shape, width)

	return &Film{
		Data: [3]math_lib.Tensor[float64]{
			*math_lib.NewTensor[float64](shape),
			*math_lib.NewTensor[float64](shape),
			*math_lib.NewTensor[float64](shape),
		},
		Samples:    0,
		ColorSpace: FilmColorSpaceLinearSRGB,
	}
}

func (f *Film) Init(width ...int) *Film {
	shape := make([]int, len(width))
	copy(shape, width)

	f.Data = [3]math_lib.Tensor[float64]{
		*math_lib.NewTensor[float64](shape),
		*math_lib.NewTensor[float64](shape),
		*math_lib.NewTensor[float64](shape),
	}
	f.Samples = 0
	f.ColorSpace = FilmColorSpaceLinearSRGB
	f.SpectralBins = nil
	f.SpectralMinNM = 0
	f.SpectralMaxNM = 0
	return f
}

func (f *Film) Merge(a *Film) *Film {
	if !reflect.DeepEqual(f.Data[0].Shape, a.Data[0].Shape) {
		panic("Dimension of a and b is not matched ")
	} else if f.ColorSpace != "" && a.ColorSpace != "" && f.ColorSpace != a.ColorSpace {
		panic("Working space of a and b is not matched")
	}

	totalSamples := f.Samples + a.Samples
	if totalSamples == 0 {
		return f
	}

	for ch := 0; ch < 3; ch++ {
		for i := range f.Data[ch].Data {
			f.Data[ch].Data[i] = (f.Data[ch].Data[i]*float64(f.Samples) + a.Data[ch].Data[i]*float64(a.Samples)) / float64(totalSamples)
		}
	}
	f.mergeSpectralDiagnostics(a, totalSamples)
	f.Samples = totalSamples
	return f
}

func (f *Film) InitSpectralBins(count int, minNM, maxNM float64) {
	if count <= 0 || len(f.Data[0].Shape) == 0 {
		f.SpectralBins = nil
		f.SpectralMinNM = 0
		f.SpectralMaxNM = 0
		return
	}
	if minNM <= 0 {
		minNM = 380
	}
	if maxNM <= minNM {
		maxNM = 750
	}
	f.SpectralBins = make([]math_lib.Tensor[float64], count)
	for i := range f.SpectralBins {
		f.SpectralBins[i] = *math_lib.NewTensor[float64](append([]int(nil), f.Data[0].Shape...))
	}
	f.SpectralMinNM = minNM
	f.SpectralMaxNM = maxNM
}

func (f *Film) HasSpectralBins() bool {
	return len(f.SpectralBins) > 0 && f.SpectralMaxNM > f.SpectralMinNM
}

func (f *Film) RecordSpectralSample(pixel int, wavelengthNM, value float64) {
	if !f.HasSpectralBins() || pixel < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	bin := f.SpectralBinIndex(wavelengthNM)
	if bin < 0 || bin >= len(f.SpectralBins) || pixel >= len(f.SpectralBins[bin].Data) {
		return
	}
	f.SpectralBins[bin].Data[pixel] += value
}

func (f *Film) SpectralBinIndex(wavelengthNM float64) int {
	if !f.HasSpectralBins() || wavelengthNM < f.SpectralMinNM || wavelengthNM >= f.SpectralMaxNM {
		return -1
	}
	t := (wavelengthNM - f.SpectralMinNM) / (f.SpectralMaxNM - f.SpectralMinNM)
	idx := int(t * float64(len(f.SpectralBins)))
	if idx < 0 || idx >= len(f.SpectralBins) {
		return -1
	}
	return idx
}

func (f *Film) mergeSpectralDiagnostics(a *Film, totalSamples int64) {
	if !f.compatibleSpectralBins(a) {
		return
	}
	for bin := range f.SpectralBins {
		for i := range f.SpectralBins[bin].Data {
			f.SpectralBins[bin].Data[i] = (f.SpectralBins[bin].Data[i]*float64(f.Samples) + a.SpectralBins[bin].Data[i]*float64(a.Samples)) / float64(totalSamples)
		}
	}
}

func (f *Film) compatibleSpectralBins(a *Film) bool {
	return f != nil && a != nil &&
		len(f.SpectralBins) > 0 &&
		len(f.SpectralBins) == len(a.SpectralBins) &&
		f.SpectralMinNM == a.SpectralMinNM &&
		f.SpectralMaxNM == a.SpectralMaxNM
}

func (f *Film) ToImage() *image.RGBA {
	return f.ToImageWithOptions(ImageOptions{})
}

func (f *Film) ToImageWithOptions(options ImageOptions) *image.RGBA {
	options = normalizeImageOptions(options)
	if len(f.Data[0].Shape) == 2 {
		imgout := image.NewRGBA(image.Rect(0, 0, f.Data[0].Shape[0], f.Data[0].Shape[1]))
		for i := 0; i < len(f.Data[0].Data); i++ {
			red, green, blue := f.outputRGBAt(i)
			r := encodeOutputChannel(red, options)
			g := encodeOutputChannel(green, options)
			b := encodeOutputChannel(blue, options)
			ind := f.Data[0].GetCoordinates(i)
			imgout.Set(ind[0], ind[1], color.RGBA{r, g, b, 255})
		}

		return imgout
	} else if len(f.Data[0].Shape) == 3 {
		imgout := image.NewRGBA(image.Rect(0, 0, f.Data[0].Shape[0], f.Data[0].Shape[1]*f.Data[0].Shape[2]))
		for i := 0; i < len(f.Data[0].Data); i++ {
			red, green, blue := f.outputRGBAt(i)
			r := encodeOutputChannel(red, options)
			g := encodeOutputChannel(green, options)
			b := encodeOutputChannel(blue, options)
			ind := f.Data[0].GetCoordinates(i)
			imgout.Set(ind[0], ind[1]+ind[2]*f.Data[0].Shape[1], color.RGBA{r, g, b, 255})
		}

		return imgout
	}
	return nil
}

func (f *Film) outputRGBAt(i int) (float64, float64, float64) {
	a := f.Data[0].Data[i]
	b := f.Data[1].Data[i]
	c := f.Data[2].Data[i]
	if f.ColorSpace == FilmColorSpaceXYZ {
		return xyzToLinearSRGB(a, b, c)
	}
	return a, b, c
}

func normalizeImageOptions(options ImageOptions) ImageOptions {
	if options.Exposure == 0 {
		options.Exposure = 1
	}
	if options.ToneMapping == "" {
		options.ToneMapping = ToneMappingLinear
	}
	if options.Gamma == 0 {
		options.Gamma = 1
	}
	return options
}

func encodeOutputChannel(v float64, options ImageOptions) uint8 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0
	}

	v *= options.Exposure
	switch options.ToneMapping {
	case ToneMappingReinhard:
		v = v / (1 + v)
	case ToneMappingACES:
		v = acesToneMap(v)
	}

	v = clamp01(v)
	if options.Gamma > 0 && options.Gamma != 1 {
		v = math.Pow(v, 1/options.Gamma)
	}
	return uint8(clamp01(v)*255 + 0.5)
}

func acesToneMap(v float64) float64 {
	return (v * (2.51*v + 0.03)) / (v*(2.43*v+0.59) + 0.14)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (f *Film) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if err = binary.Read(file, binary.LittleEndian, &f.Samples); err != nil {
		return err
	}

	var shapeLen int32
	if err = binary.Read(file, binary.LittleEndian, &shapeLen); err != nil {
		return err
	}

	shape := make([]int, shapeLen)
	for i := range shape {
		var dim int32
		if err = binary.Read(file, binary.LittleEndian, &dim); err != nil {
			return err
		}
		shape[i] = int(dim)
	}

	f.Data = [3]math_lib.Tensor[float64]{
		*math_lib.NewTensor[float64](shape),
		*math_lib.NewTensor[float64](shape),
		*math_lib.NewTensor[float64](shape),
	}
	f.ColorSpace = FilmColorSpaceLinearSRGB

	for ch := 0; ch < 3; ch++ {
		for i := range f.Data[ch].Data {
			if err = binary.Read(file, binary.LittleEndian, &f.Data[ch].Data[i]); err != nil {
				return err
			}
		}
	}

	if err = f.readOptionalColorSpace(file); err != nil {
		return err
	}

	return nil
}

func xyzToLinearSRGB(x, y, z float64) (float64, float64, float64) {
	return 3.2404542*x - 1.5371385*y - 0.4985314*z,
		-0.9692660*x + 1.8760108*y + 0.0415560*z,
		0.0556434*x - 0.2040259*y + 1.0572252*z
}

func (f *Film) SaveToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if err = binary.Write(file, binary.LittleEndian, f.Samples); err != nil {
		return err
	}

	shapeLen := int32(len(f.Data[0].Shape))
	if err = binary.Write(file, binary.LittleEndian, shapeLen); err != nil {
		return err
	}

	for _, dim := range f.Data[0].Shape {
		if err = binary.Write(file, binary.LittleEndian, int32(dim)); err != nil {
			return err
		}
	}

	for ch := 0; ch < 3; ch++ {
		for i := range f.Data[ch].Data {
			if err = binary.Write(file, binary.LittleEndian, f.Data[ch].Data[i]); err != nil {
				return err
			}
		}
	}

	space := []byte(f.ColorSpace)
	spaceLen := int32(len(space))
	if err = binary.Write(file, binary.LittleEndian, spaceLen); err != nil {
		return err
	}
	if spaceLen > 0 {
		if _, err = file.Write(space); err != nil {
			return err
		}
	}

	return nil
}

func (f *Film) readOptionalColorSpace(file *os.File) error {
	var spaceLen int32
	if err := binary.Read(file, binary.LittleEndian, &spaceLen); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	if spaceLen <= 0 {
		return nil
	}

	buf := make([]byte, spaceLen)
	if _, err := io.ReadFull(file, buf); err != nil {
		return err
	}
	switch FilmColorSpace(buf) {
	case FilmColorSpaceLinearSRGB, FilmColorSpaceXYZ:
		f.ColorSpace = FilmColorSpace(buf)
	}
	return nil
}
