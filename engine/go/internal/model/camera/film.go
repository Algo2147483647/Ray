package camera

import (
	"encoding/binary"
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"image"
	"image/color"
	"math"
	"os"
	"reflect"
)

type Film struct {
	Data    [3]math_lib.Tensor[float64] `json:"data"`
	Samples int64                       `json:"samples"`
}

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

func NewFilm(width ...int) *Film {
	shape := make([]int, len(width))
	copy(shape, width)

	return &Film{
		Data: [3]math_lib.Tensor[float64]{
			*math_lib.NewTensor[float64](shape),
			*math_lib.NewTensor[float64](shape),
			*math_lib.NewTensor[float64](shape),
		},
		Samples: 0,
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
	return f
}

func (f *Film) Merge(a *Film) *Film {
	if !reflect.DeepEqual(f.Data[0].Shape, a.Data[0].Shape) {
		panic("Dimension of a and b is not matched ")
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
	f.Samples = totalSamples
	return f
}

func (f *Film) ToImage() *image.RGBA {
	return f.ToImageWithOptions(ImageOptions{})
}

func (f *Film) ToImageWithOptions(options ImageOptions) *image.RGBA {
	options = normalizeImageOptions(options)
	if len(f.Data[0].Shape) == 2 {
		imgout := image.NewRGBA(image.Rect(0, 0, f.Data[0].Shape[0], f.Data[0].Shape[1]))
		for i := 0; i < len(f.Data[0].Data); i++ {
			r := encodeOutputChannel(f.Data[0].Data[i], options)
			g := encodeOutputChannel(f.Data[1].Data[i], options)
			b := encodeOutputChannel(f.Data[2].Data[i], options)
			ind := f.Data[0].GetCoordinates(i)
			imgout.Set(ind[0], ind[1], color.RGBA{r, g, b, 255})
		}

		return imgout
	} else if len(f.Data[0].Shape) == 3 {
		imgout := image.NewRGBA(image.Rect(0, 0, f.Data[0].Shape[0], f.Data[0].Shape[1]*f.Data[0].Shape[2]))
		for i := 0; i < len(f.Data[0].Data); i++ {
			r := encodeOutputChannel(f.Data[0].Data[i], options)
			g := encodeOutputChannel(f.Data[1].Data[i], options)
			b := encodeOutputChannel(f.Data[2].Data[i], options)
			ind := f.Data[0].GetCoordinates(i)
			imgout.Set(ind[0], ind[1]+ind[2]*f.Data[0].Shape[1], color.RGBA{r, g, b, 255})
		}

		return imgout
	}
	return nil
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
	case ToneMappingLinear:
	default:
		v = v
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

	for ch := 0; ch < 3; ch++ {
		for i := range f.Data[ch].Data {
			if err = binary.Read(file, binary.LittleEndian, &f.Data[ch].Data[i]); err != nil {
				return err
			}
		}
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

	return nil
}
