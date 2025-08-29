package math_lib

import "reflect"

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~complex64 | ~complex128
}

type Tensor[T Number] struct {
	Data   []T     `json:"data"`
	Shape  []int64 `json:"shape"`
	Stride []int64 `json:"stride"`
	Offset int64   `json:"offset"`
}

func NewTensor[T Number](shape []int64) *Tensor[T] {
	t := &Tensor[T]{
		Shape: shape,
	}

	var total int64
	t.Stride, total = CalculateStrideForTensor(shape)
	t.Data = make([]T, total)
	return t
}

func NewTensorFromSlice[T Number](data []T, shape []int64) *Tensor[T] {
	t := NewTensor[T](shape)
	copy(t.Data, data)
	return t
}

func CalculateStrideForTensor(shape []int64) ([]int64, int64) {
	stride := make([]int64, len(shape))
	total := int64(1)
	for i := len(shape) - 1; i >= 0; i-- {
		stride[i] = total
		total *= shape[i]
	}
	return stride, total
}

func (t *Tensor[T]) Get(indices ...int64) T {
	if len(indices) != len(t.Shape) {
		panic("维度不匹配")
	}

	pos := t.Offset
	for i, idx := range indices {
		if idx < 0 || idx >= t.Shape[i] {
			panic("索引超出范围")
		}
		pos += idx * t.Stride[i]
	}

	return t.Data[pos]
}

func (t *Tensor[T]) Set(value T, indices ...int64) {
	if len(indices) != len(t.Shape) {
		panic("维度不匹配")
	}

	pos := t.Offset
	for i, idx := range indices {
		if idx < 0 || idx >= t.Shape[i] {
			panic("索引超出范围")
		}
		pos += idx * t.Stride[i]
	}

	t.Data[pos] = value
}

func (t *Tensor[T]) Reshape(newShape []int64) *Tensor[T] {
	stride, total := CalculateStrideForTensor(newShape)
	if total != int64(len(t.Data)) {
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

func (t *Tensor[T]) GetCoordinates(i int64) []int64 {
	if i < 0 || i >= int64(len(t.Data)) {
		return nil // 或者返回错误，索引越界
	}

	actualIndex := t.Offset + i
	coords := make([]int64, len(t.Shape))

	for dim := len(t.Stride) - 1; dim >= 0; dim-- {
		coords[dim] = actualIndex / t.Stride[dim]
		actualIndex = actualIndex % t.Stride[dim]
	}

	return coords
}
