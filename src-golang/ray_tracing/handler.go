package ray_tracing

import (
	"gonum.org/v1/gonum/mat"
	"src-golang/model/optics"
	"sync"
)

type Handler struct {
	MaxRayLevel int64     `json:"max_ray_level"` // 最大光线递归深度
	ThreadNum   int       `json:"thread_num"`    // 并发线程数
	BlockCols   int       `json:"block_cols"`
	BlockRows   int       `json:"block_rows"`
	RayPool     sync.Pool `json:"ray_pool"` // RayPool 光线对象池
}

func NewHandler() *Handler {
	return &Handler{
		MaxRayLevel: 6,
		ThreadNum:   30,
		BlockCols:   8,
		BlockRows:   8,
		RayPool: sync.Pool{
			New: func() interface{} {
				return &optics.Ray{
					Origin:    mat.NewVecDense(3, nil),
					Direction: mat.NewVecDense(3, nil),
					Color:     mat.NewVecDense(3, nil),
				}
			},
		},
	}
}
