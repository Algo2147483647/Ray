package camera

import (
	"encoding/binary"
	"image"
	"image/color"
	"os"
	"reflect"
	"src-golang/math_lib"
)

type Film struct {
	Data    [3]math_lib.Tensor[float64] `json:"data"`
	Samples int64                       `json:"samples"`
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

	for i := range f.Data[0].Data {
		f.Data[0].Data[i] = (f.Data[0].Data[i]*float64(f.Samples) + a.Data[0].Data[i]*float64(a.Samples)) / float64(f.Samples+a.Samples)
	}
	f.Samples += a.Samples
	return f
}

func (f *Film) ToImage() *image.RGBA {
	imgout := image.NewRGBA(image.Rect(0, 0, int(f.Data[0].Shape[0]), int(f.Data[0].Shape[1]*f.Data[0].Shape[2])))
	for i := 0; i < len(f.Data[0].Data); i++ {
		r := uint8(min(f.Data[0].Data[i]*255, 255))
		g := uint8(min(f.Data[1].Data[i]*255, 255))
		b := uint8(min(f.Data[2].Data[i]*255, 255))
		ind := f.Data[0].GetCoordinates(i)
		imgout.Set(int(ind[0]), int(ind[1]+ind[2]*f.Data[0].Shape[1]), color.RGBA{r, g, b, 255})
	}

	return imgout
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
		var dim int
		if err = binary.Read(file, binary.LittleEndian, &dim); err != nil {
			return err
		}
		shape[i] = dim
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
