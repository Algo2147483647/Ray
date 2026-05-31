package camera

import (
	"encoding/binary"
	"errors"
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"reflect"
)

type Film struct {
	Data          [3]maths.Tensor[float64] `json:"data"`                      // RGB or tristimulus channel data.
	Samples       int64                    `json:"samples"`                   // Number of accumulated samples.
	ColorSpace    FilmColorSpace           `json:"color_space"`               // Color encoding used for output.
	SpectralBins  []maths.Tensor[float64]  `json:"spectral_bins,omitempty"`   // Per-wavelength-band accumulated spectral data.
	SpectralMinNM float64                  `json:"spectral_min_nm,omitempty"` // Lower bound of the spectral range, in nm.
	SpectralMaxNM float64                  `json:"spectral_max_nm,omitempty"` // Upper bound of the spectral range, in nm.
}

type FilmColorSpace string

const (
	FilmColorSpaceLinearSRGB FilmColorSpace = "linear_srgb"
	FilmColorSpaceXYZ        FilmColorSpace = "xyz"
	FilmColorSpaceACEScg     FilmColorSpace = "acescg"
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

var spectralFilmMagic = [4]byte{'S', 'P', 'C', 'T'}

func NewFilm(width ...int) *Film {
	shape := make([]int, len(width))
	copy(shape, width)

	return &Film{
		Data: [3]maths.Tensor[float64]{
			*maths.NewTensor[float64](shape),
			*maths.NewTensor[float64](shape),
			*maths.NewTensor[float64](shape),
		},
		Samples:    0,
		ColorSpace: FilmColorSpaceLinearSRGB,
	}
}

func (f *Film) Init(width ...int) *Film {
	shape := make([]int, len(width))
	copy(shape, width)

	f.Data = [3]maths.Tensor[float64]{
		*maths.NewTensor[float64](shape),
		*maths.NewTensor[float64](shape),
		*maths.NewTensor[float64](shape),
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
	f.SpectralBins = make([]maths.Tensor[float64], count)
	for i := range f.SpectralBins {
		f.SpectralBins[i] = *maths.NewTensor[float64](append([]int(nil), f.Data[0].Shape...))
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

func (f *Film) SpectralBinCenterNM(bin int) float64 {
	if !f.HasSpectralBins() || bin < 0 || bin >= len(f.SpectralBins) {
		return 0
	}
	width := (f.SpectralMaxNM - f.SpectralMinNM) / float64(len(f.SpectralBins))
	return f.SpectralMinNM + (float64(bin)+0.5)*width
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

func (f *Film) ConvertSpectralBinsToFilmColorSpace() {
	if !f.HasSpectralBins() {
		return
	}
	for pixel := range f.Data[0].Data {
		x, y, z := f.spectralXYZAt(pixel)
		a, b, c := XYZToFilmColorSpace(x, y, z, f.ColorSpace)
		f.Data[0].Data[pixel] = a
		f.Data[1].Data[pixel] = b
		f.Data[2].Data[pixel] = c
	}
}

func (f *Film) spectralXYZAt(pixel int) (float64, float64, float64) {
	var x, y, z float64
	for bin := range f.SpectralBins {
		if pixel < 0 || pixel >= len(f.SpectralBins[bin].Data) {
			continue
		}
		xyz := optics.SpectralRadianceToXYZ(f.SpectralBinCenterNM(bin), f.SpectralBins[bin].Data[pixel])
		x += xyz[0]
		y += xyz[1]
		z += xyz[2]
	}
	return x, y, z
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
	} else if len(f.Data[0].Shape) > 2 {
		width := f.Data[0].Shape[0]
		height := f.Data[0].Shape[1]
		slices := 1
		for _, extent := range f.Data[0].Shape[2:] {
			slices *= extent
		}
		atlasCols := int(math.Ceil(math.Sqrt(float64(slices))))
		atlasRows := (slices + atlasCols - 1) / atlasCols
		imgout := image.NewRGBA(image.Rect(0, 0, width*atlasCols, height*atlasRows))
		for i := 0; i < len(f.Data[0].Data); i++ {
			red, green, blue := f.outputRGBAt(i)
			r := encodeOutputChannel(red, options)
			g := encodeOutputChannel(green, options)
			b := encodeOutputChannel(blue, options)
			ind := f.Data[0].GetCoordinates(i)
			slice := flattenedSliceIndex(ind[2:], f.Data[0].Shape[2:])
			atlasX := slice % atlasCols
			atlasY := slice / atlasCols
			imgout.Set(ind[0]+atlasX*width, ind[1]+atlasY*height, color.RGBA{r, g, b, 255})
		}

		return imgout
	}
	return nil
}

func flattenedSliceIndex(coords, shape []int) int {
	index := 0
	stride := 1
	for i := 0; i < len(coords) && i < len(shape); i++ {
		index += coords[i] * stride
		stride *= shape[i]
	}
	return index
}

func (f *Film) outputRGBAt(i int) (float64, float64, float64) {
	a := f.Data[0].Data[i]
	b := f.Data[1].Data[i]
	c := f.Data[2].Data[i]
	switch f.ColorSpace {
	case FilmColorSpaceXYZ:
		return optics.XYZToLinearSRGB(a, b, c)
	case FilmColorSpaceACEScg:
		return optics.ACEScgToLinearSRGB(a, b, c)
	default:
		return a, b, c
	}
}

func XYZToFilmColorSpace(x, y, z float64, space FilmColorSpace) (float64, float64, float64) {
	switch space {
	case FilmColorSpaceXYZ:
		return x, y, z
	case FilmColorSpaceACEScg:
		return optics.XYZToACEScg(x, y, z)
	default:
		return optics.XYZToLinearSRGB(x, y, z)
	}
}

func LinearSRGBToFilmColorSpace(r, g, b float64, space FilmColorSpace) (float64, float64, float64) {
	switch space {
	case FilmColorSpaceXYZ:
		xyz := optics.LinearSRGBToXYZ(r, g, b)
		return xyz[0], xyz[1], xyz[2]
	case FilmColorSpaceACEScg:
		return optics.LinearSRGBToACEScg(r, g, b)
	default:
		return r, g, b
	}
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

	f.Data = [3]maths.Tensor[float64]{
		*maths.NewTensor[float64](shape),
		*maths.NewTensor[float64](shape),
		*maths.NewTensor[float64](shape),
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
	if err = f.readOptionalSpectralBins(file); err != nil {
		return err
	}

	return nil
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

	if err = f.writeOptionalSpectralBins(file); err != nil {
		return err
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
	case FilmColorSpaceLinearSRGB, FilmColorSpaceXYZ, FilmColorSpaceACEScg:
		f.ColorSpace = FilmColorSpace(buf)
	}
	return nil
}

func (f *Film) writeOptionalSpectralBins(file *os.File) error {
	if !f.HasSpectralBins() {
		return nil
	}
	if _, err := file.Write(spectralFilmMagic[:]); err != nil {
		return err
	}
	count := int32(len(f.SpectralBins))
	if err := binary.Write(file, binary.LittleEndian, count); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, f.SpectralMinNM); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, f.SpectralMaxNM); err != nil {
		return err
	}
	for bin := range f.SpectralBins {
		for i := range f.SpectralBins[bin].Data {
			if err := binary.Write(file, binary.LittleEndian, f.SpectralBins[bin].Data[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Film) readOptionalSpectralBins(file *os.File) error {
	var magic [4]byte
	if _, err := io.ReadFull(file, magic[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}
		return err
	}
	if magic != spectralFilmMagic {
		return nil
	}

	var count int32
	if err := binary.Read(file, binary.LittleEndian, &count); err != nil {
		return err
	}
	var minNM, maxNM float64
	if err := binary.Read(file, binary.LittleEndian, &minNM); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &maxNM); err != nil {
		return err
	}
	f.InitSpectralBins(int(count), minNM, maxNM)
	for bin := range f.SpectralBins {
		for i := range f.SpectralBins[bin].Data {
			if err := binary.Read(file, binary.LittleEndian, &f.SpectralBins[bin].Data[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
