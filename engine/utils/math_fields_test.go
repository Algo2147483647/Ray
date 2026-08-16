package utils

import "testing"

func TestNewVecs(t *testing.T) {
	vectors := NewVecs([][]float64{{1, 2}, {3, 4}})
	if len(vectors) != 2 || vectors[0].AtVec(1) != 2 || vectors[1].AtVec(0) != 3 {
		t.Fatalf("unexpected vectors: %v", vectors)
	}
}
