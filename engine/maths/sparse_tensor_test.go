package maths

import "testing"

func TestSparseTensorSetGetAndDeleteDefault(t *testing.T) {
	tensor := NewSparseTensor[float64]([]int{4, 5, 6}, SparseTensorHash)

	if err := tensor.Set([]int{2, 3, 4}, 7); err != nil {
		t.Fatalf("set sparse tensor value: %v", err)
	}
	if got := tensor.MustGet([]int{2, 3, 4}); got != 7 {
		t.Fatalf("expected stored value 7, got %v", got)
	}
	if got := tensor.MustGet([]int{0, 0, 0}); got != 0 {
		t.Fatalf("expected default zero, got %v", got)
	}

	if err := tensor.Set([]int{2, 3, 4}, 0); err != nil {
		t.Fatalf("delete sparse tensor value: %v", err)
	}
	if tensor.NNZ() != 0 {
		t.Fatalf("expected empty sparse tensor, got nnz=%d", tensor.NNZ())
	}
}

func TestSparseTensorAdd(t *testing.T) {
	a, err := NewSparseTensorFromEntries([]int{3, 3}, SparseTensorHash, []SparseTensorEntry[float64]{
		{Index: []int{0, 1}, Value: 2},
		{Index: []int{2, 2}, Value: 3},
	})
	if err != nil {
		t.Fatalf("create tensor a: %v", err)
	}

	b, err := NewSparseTensorFromEntries([]int{3, 3}, SparseTensorCOO, []SparseTensorEntry[float64]{
		{Index: []int{0, 1}, Value: -2},
		{Index: []int{1, 1}, Value: 5},
	})
	if err != nil {
		t.Fatalf("create tensor b: %v", err)
	}

	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("add sparse tensors: %v", err)
	}

	if got := sum.MustGet([]int{0, 1}); got != 0 {
		t.Fatalf("expected cancellation to default, got %v", got)
	}
	if got := sum.MustGet([]int{1, 1}); got != 5 {
		t.Fatalf("expected summed value 5, got %v", got)
	}
	if got := sum.MustGet([]int{2, 2}); got != 3 {
		t.Fatalf("expected retained value 3, got %v", got)
	}
}
