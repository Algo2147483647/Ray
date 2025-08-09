package model

import (
	"gonum.org/v1/gonum/mat"
)

// Ray 表示光线
type Ray struct {
	Origin       *mat.VecDense
	Direction    *mat.VecDense
	Color        *mat.VecDense
	Refractivity float64
}
