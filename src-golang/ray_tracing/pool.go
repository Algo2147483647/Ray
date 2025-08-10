package ray_tracing

import (
	"gonum.org/v1/gonum/mat"
	"src-golang/model"
	"sync"
)

// RayPool 光线对象池
var RayPool = sync.Pool{
	New: func() interface{} {
		return &model.Ray{
			Origin:    mat.NewVecDense(3, nil),
			Direction: mat.NewVecDense(3, nil),
			Color:     mat.NewVecDense(3, nil),
		}
	},
}
