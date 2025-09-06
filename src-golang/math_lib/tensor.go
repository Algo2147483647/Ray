package math_lib

import "reflect"

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~complex64 | ~complex128
}

type Tensor[T Number] struct {
	Data   []T   `json:"data"`
	Shape  []int `json:"shape"`
	Stride []int `json:"stride"`
	Offset int   `json:"offset"`
}

func NewTensor[T Number](shape []int) *Tensor[T] {
	t := &Tensor[T]{
		Shape: shape,
	}

	var total int
	t.Stride, total = CalculateStrideForTensor(shape)
	t.Data = make([]T, total)
	return t
}

func NewTensorFromSlice[T Number](data []T, shape []int) *Tensor[T] {
	t := NewTensor[T](shape)
	copy(t.Data, data)
	return t
}

func CalculateStrideForTensor(shape []int) ([]int, int) {
	stride := make([]int, len(shape))
	total := 1
	for i := 0; i < len(shape); i++ {
		stride[i] = total
		total *= shape[i]
	}
	return stride, total
}

func (t *Tensor[T]) GetCoordinates(i int) []int {
	if i < 0 || i >= len(t.Data) {
		return nil // 或者返回错误，索引越界
	}

	actualIndex := i
	coords := make([]int, len(t.Shape))

	for dim := len(t.Stride) - 1; dim >= 0; dim-- {
		coords[dim] = actualIndex / t.Stride[dim]
		actualIndex = actualIndex % t.Stride[dim]
	}

	return coords
}

func (t *Tensor[T]) CoordinateToIndex(coordinate ...int) int {
	if len(coordinate) != len(t.Shape) {
		panic("维度不匹配")
	}

	index := t.Offset
	for i, idx := range coordinate {
		if idx < 0 || idx >= t.Shape[i] {
			panic("索引超出范围")
		}
		index += idx * t.Stride[i]
	}

	return index
}

func (t *Tensor[T]) Get(coordinate ...int) T {
	return t.Data[t.CoordinateToIndex(coordinate...)]
}

func (t *Tensor[T]) Set(value T, coordinate ...int) {
	t.Data[t.CoordinateToIndex(coordinate...)] = value
}

func (t *Tensor[T]) Reshape(newShape []int) *Tensor[T] {
	stride, total := CalculateStrideForTensor(newShape)
	if total != len(t.Data) {
		panic("新形状的元素总数不匹配")
	}

	return &Tensor[T]{
		Data:   t.Data,
		Shape:  newShape,
		Stride: stride,
		Offset: t.Offset,
	}
}

func (t *Tensor[T]) Add(a, b *Tensor[T]) *Tensor[T] {
	if !reflect.DeepEqual(a.Shape, b.Shape) {
		panic("张量维度不匹配")
	} else if !reflect.DeepEqual(a.Shape, t.Shape) {
		t = NewTensor[T](a.Shape)
	}

	for i := range t.Data {
		t.Data[i] = a.Data[i] + b.Data[i]
	}

	return t
}

func (t *Tensor[T]) Sub(a, b *Tensor[T]) *Tensor[T] {
	if !reflect.DeepEqual(a.Shape, b.Shape) {
		panic("张量维度不匹配")
	} else if !reflect.DeepEqual(a.Shape, t.Shape) {
		t = NewTensor[T](a.Shape)
	}

	for i := range t.Data {
		t.Data[i] = a.Data[i] - b.Data[i]
	}

	return t
}

func (t *Tensor[T]) ScalarMul(scalar T) *Tensor[T] {
	for i := range t.Data {
		t.Data[i] = t.Data[i] * scalar
	}
	return t
}
