package maths

import (
	"fmt"
	"reflect"
	"sort"
)

type SparseTensorFormat string

const (
	SparseTensorCOO   SparseTensorFormat = "coo"
	SparseTensorHash  SparseTensorFormat = "hash"
	SparseTensorCSR   SparseTensorFormat = "csr"
	SparseTensorCSC   SparseTensorFormat = "csc"
	SparseTensorBlock SparseTensorFormat = "block"
)

type SparseTensorEntry[T Number] struct {
	Index []int `json:"index"`
	Value T     `json:"value"`
}

type SparseTensor[T Number] struct {
	Shape   []int                  `json:"shape"`
	Format  SparseTensorFormat     `json:"format"`
	Default T                      `json:"default"`
	Entries []SparseTensorEntry[T] `json:"entries"`

	entryIndex map[int]int
	stride     []int
}

func NewSparseTensor[T Number](shape []int, format SparseTensorFormat) *SparseTensor[T] {
	if format == "" {
		format = SparseTensorHash
	}
	t := &SparseTensor[T]{
		Shape:  append([]int(nil), shape...),
		Format: format,
	}
	t.rebuildIndex()
	return t
}

func NewSparseTensorFromEntries[T Number](
	shape []int,
	format SparseTensorFormat,
	entries []SparseTensorEntry[T],
) (*SparseTensor[T], error) {
	t := NewSparseTensor[T](shape, format)
	for _, entry := range entries {
		if err := t.Set(entry.Index, entry.Value); err != nil {
			return nil, err
		}
	}
	return t, nil
}

func (t *SparseTensor[T]) NNZ() int {
	if t == nil {
		return 0
	}
	return len(t.Entries)
}

func (t *SparseTensor[T]) Get(index []int) (T, error) {
	var zero T
	if err := t.validateIndex(index); err != nil {
		return zero, err
	}
	t.ensureIndex()
	if position, ok := t.entryIndex[t.flatIndex(index)]; ok {
		return t.Entries[position].Value, nil
	}
	return t.Default, nil
}

func (t *SparseTensor[T]) MustGet(index []int) T {
	value, err := t.Get(index)
	if err != nil {
		panic(err)
	}
	return value
}

func (t *SparseTensor[T]) Set(index []int, value T) error {
	if err := t.validateIndex(index); err != nil {
		return err
	}
	t.ensureIndex()

	flat := t.flatIndex(index)
	if position, ok := t.entryIndex[flat]; ok {
		if value == t.Default {
			t.Entries = append(t.Entries[:position], t.Entries[position+1:]...)
			t.rebuildIndex()
			return nil
		}
		t.Entries[position].Value = value
		return nil
	}

	if value == t.Default {
		return nil
	}

	t.entryIndex[flat] = len(t.Entries)
	t.Entries = append(t.Entries, SparseTensorEntry[T]{
		Index: append([]int(nil), index...),
		Value: value,
	})
	return nil
}

func (t *SparseTensor[T]) IterNonZero(fn func(index []int, value T)) {
	if t == nil || fn == nil {
		return
	}
	for _, entry := range t.Entries {
		fn(append([]int(nil), entry.Index...), entry.Value)
	}
}

func (t *SparseTensor[T]) ToCOO() *SparseTensor[T] {
	return t.copyAs(SparseTensorCOO)
}

func (t *SparseTensor[T]) ToHash() *SparseTensor[T] {
	return t.copyAs(SparseTensorHash)
}

func (t *SparseTensor[T]) Add(other *SparseTensor[T]) (*SparseTensor[T], error) {
	if t == nil || other == nil {
		return nil, ErrInvalidInput
	}
	if !reflect.DeepEqual(t.Shape, other.Shape) {
		return nil, fmt.Errorf("%w: sparse tensor shapes differ", ErrInvalidInput)
	}

	result := NewSparseTensor[T](t.Shape, SparseTensorHash)
	for _, entry := range t.Entries {
		if err := result.Set(entry.Index, entry.Value); err != nil {
			return nil, err
		}
	}
	for _, entry := range other.Entries {
		current, err := result.Get(entry.Index)
		if err != nil {
			return nil, err
		}
		if err := result.Set(entry.Index, current+entry.Value); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (t *SparseTensor[T]) ScalarMul(scalar T) *SparseTensor[T] {
	result := NewSparseTensor[T](t.Shape, t.Format)
	result.Default = t.Default * scalar
	for _, entry := range t.Entries {
		_ = result.Set(entry.Index, entry.Value*scalar)
	}
	return result
}

func (t *SparseTensor[T]) copyAs(format SparseTensorFormat) *SparseTensor[T] {
	if t == nil {
		return nil
	}
	result := NewSparseTensor[T](t.Shape, format)
	result.Default = t.Default
	entries := append([]SparseTensorEntry[T](nil), t.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		return result.flatIndex(entries[i].Index) < result.flatIndex(entries[j].Index)
	})
	for _, entry := range entries {
		_ = result.Set(entry.Index, entry.Value)
	}
	return result
}

func (t *SparseTensor[T]) validateIndex(index []int) error {
	if t == nil {
		return ErrInvalidInput
	}
	if len(index) != len(t.Shape) {
		return fmt.Errorf("%w: index rank %d does not match tensor rank %d", ErrInvalidInput, len(index), len(t.Shape))
	}
	for axis, value := range index {
		if value < 0 || value >= t.Shape[axis] {
			return fmt.Errorf("%w: index %d on axis %d outside [0,%d)", ErrInvalidInput, value, axis, t.Shape[axis])
		}
	}
	return nil
}

func (t *SparseTensor[T]) ensureIndex() {
	if t.entryIndex == nil || len(t.stride) != len(t.Shape) {
		t.rebuildIndex()
	}
}

func (t *SparseTensor[T]) rebuildIndex() {
	t.stride, _ = CalculateStrideForTensor(t.Shape)
	t.entryIndex = make(map[int]int, len(t.Entries))
	for i, entry := range t.Entries {
		t.entryIndex[t.flatIndex(entry.Index)] = i
	}
}

func (t *SparseTensor[T]) flatIndex(index []int) int {
	flat := 0
	for axis, value := range index {
		flat += value * t.stride[axis]
	}
	return flat
}
