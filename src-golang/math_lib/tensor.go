package math_lib

type Tensor[T any] struct {
	data   []T
	shape  []int
	stride []int
	offset int
}

// NewTensor 创建新张量
func NewTensor[T any](shape []int) *Tensor[T] {
	t := &Tensor[T]{
		shape:  shape,
		stride: make([]int, len(shape)),
	}

	// 计算总元素数和步长
	total := 1
	for i := len(shape) - 1; i >= 0; i-- {
		t.stride[i] = total
		total *= shape[i]
	}

	t.data = make([]T, total)
	return t
}

// FromSlice 从切片创建张量
func FromSlice[T any](data []T, shape []int) *Tensor[T] {
	t := NewTensor[T](shape)
	copy(t.data, data)
	return t
}

// Get 获取元素
func (t *Tensor[T]) Get(indices ...int) T {
	if len(indices) != len(t.shape) {
		panic("维度不匹配")
	}

	pos := t.offset
	for i, idx := range indices {
		if idx < 0 || idx >= t.shape[i] {
			panic("索引超出范围")
		}
		pos += idx * t.stride[i]
	}

	return t.data[pos]
}

// Set 设置元素
func (t *Tensor[T]) Set(value T, indices ...int) {
	if len(indices) != len(t.shape) {
		panic("维度不匹配")
	}

	pos := t.offset
	for i, idx := range indices {
		if idx < 0 || idx >= t.shape[i] {
			panic("索引超出范围")
		}
		pos += idx * t.stride[i]
	}

	t.data[pos] = value
}

// Shape 返回形状
func (t *Tensor[T]) Shape() []int {
	return t.shape
}

// Reshape 改变形状
func (t *Tensor[T]) Reshape(newShape []int) *Tensor[T] {
	// 验证新形状的元素总数是否匹配
	total := 1
	for _, dim := range newShape {
		total *= dim
	}
	if total != len(t.data) {
		panic("新形状的元素总数不匹配")
	}

	return &Tensor[T]{
		data:   t.data,
		shape:  newShape,
		stride: calculateStride(newShape),
		offset: t.offset,
	}
}

// calculateStride 计算步长
func calculateStride(shape []int) []int {
	stride := make([]int, len(shape))
	total := 1
	for i := len(shape) - 1; i >= 0; i-- {
		stride[i] = total
		total *= shape[i]
	}
	return stride
}
