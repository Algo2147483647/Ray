package utils

import (
	"gonum.org/v1/gonum/mat"
	"sync"
)

var VectorPool sync.Pool

func init() {
	ResetVectorPool()
}

func ResetVectorPool() {
	VectorPool = sync.Pool{
		New: func() interface{} {
			return mat.NewVecDense(Dimension, nil)
		},
	}
}
