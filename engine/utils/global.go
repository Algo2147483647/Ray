package utils

const (
	EPS = 1e-5
)

var Dimension = 3

func SetDimension(dimension int) {
	if dimension <= 0 {
		dimension = 3
	}
	Dimension = dimension
	ResetVectorPool()
}
