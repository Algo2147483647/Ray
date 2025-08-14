package ray_tracing

import (
	"gonum.org/v1/gonum/mat"
	"src-golang/model/object/optics"
	"sync"
)

// RayPool 光线对象池
var RayPool = sync.Pool{
	New: func() interface{} {
		return &optics.Ray{
			Origin:    mat.NewVecDense(3, nil),
			Direction: mat.NewVecDense(3, nil),
			Color:     mat.NewVecDense(3, nil),
		}
	},
}
