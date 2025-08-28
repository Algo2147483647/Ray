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
	Data    math_lib.Tensor[float64] `json:"data"`
	Samples int64                    `json:"samples"`
}

func NewFilm(width ...int) *Film {
	shape := make([]int, len(width))
	copy(shape, width)

	return &Film{
		Data:    *math_lib.NewTensor[float64](shape),
		Samples: 0,
	}
}

func (f *Film) Init(width ...int) *Film {
	shape := make([]int, len(width))
	copy(shape, width)

	f.Data = *math_lib.NewTensor[float64](shape)
	f.Samples = 0
	return f
}

func (f *Film) Merge(a *Film) *Film {
	if !reflect.DeepEqual(f.Data.Shape, a.Data.Shape) {
		panic("Dimension of a and b is not matched ")
	}

	for i := range f.Data.Data {
		f.Data.Data[i] = (f.Data.Data[i]*float64(f.Samples) + a.Data.Data[i]*float64(a.Samples)) / float64(f.Samples+a.Samples)
	}
	f.Samples += a.Samples
	return f
}

func (f *Film) ToImage() *image.RGBA {
	imgout := image.NewRGBA(image.Rect(0, 0, f.Data.Shape[1], f.Data.Shape[2]))
	for i := 0; i < f.Data.Shape[1]; i++ {
		for j := 0; j < f.Data.Shape[2]; j++ {
			r := uint8(min(f.Data.Get(0, i, j)*255, 255))
			g := uint8(min(f.Data.Get(1, i, j)*255, 255))
			b := uint8(min(f.Data.Get(2, i, j)*255, 255))
			imgout.Set(i, j, color.RGBA{r, g, b, 255})
		}
	}

	return imgout
}

func (f *Film) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := binary.Read(file, binary.LittleEndian, &f.Samples); err != nil {
		return err
	}

	var shapeLen int32
	if err := binary.Read(file, binary.LittleEndian, &shapeLen); err != nil {
		return err
	}

	shape := make([]int, shapeLen)
	for i := range shape {
		var dim int32
		if err := binary.Read(file, binary.LittleEndian, &dim); err != nil {
			return err
		}
		shape[i] = int(dim)
	}

	f.Data = *math_lib.NewTensor[float64](shape)

	for i := range f.Data.Data {
		if err := binary.Read(file, binary.LittleEndian, &f.Data.Data[i]); err != nil {
			return err
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

	if err := binary.Write(file, binary.LittleEndian, f.Samples); err != nil {
		return err
	}

	shapeLen := int32(len(f.Data.Shape))
	if err := binary.Write(file, binary.LittleEndian, shapeLen); err != nil {
		return err
	}

	for _, dim := range f.Data.Shape {
		if err := binary.Write(file, binary.LittleEndian, int32(dim)); err != nil {
			return err
		}
	}

	for i := range f.Data.Data {
		if err := binary.Write(file, binary.LittleEndian, f.Data.Data[i]); err != nil {
			return err
		}
	}

	return nil
}
