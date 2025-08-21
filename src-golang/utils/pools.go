package utils

import (
	"gonum.org/v1/gonum/mat"
	"sync"
)

var VectorPool = sync.Pool{
	New: func() interface{} {
		return mat.NewVecDense(3, nil)
	},
}
