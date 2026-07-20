package utils

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

func NewVec(values []float64) *mat.VecDense {
	return mat.NewVecDense(len(values), values)
}

func RequiredVec(data map[string]interface{}, name string, dimension int) (*mat.VecDense, error) {
	values, err := RequiredFloat64SliceField(data, name, dimension)
	if err != nil {
		if name == "center" {
			if fallback, fallbackErr := RequiredFloat64SliceField(data, "position", dimension); fallbackErr == nil {
				return NewVec(fallback), nil
			}
		}
		return nil, err
	}

	return NewVec(values), nil
}

func RequiredNonZeroVec(data map[string]interface{}, name string, dimension int) (*mat.VecDense, error) {
	v, err := RequiredVec(data, name, dimension)
	if err != nil {
		return nil, err
	}

	if mat.Norm(v, 2) < EPS {
		return nil, fmt.Errorf("field %q must not be zero", name)
	}

	return v, nil
}

func RequiredPositiveFloat(data map[string]interface{}, name string) (float64, error) {
	value, err := RequiredFloat64Field(data, name)
	if err != nil {
		return 0, err
	} else if value <= 0 {
		return 0, fmt.Errorf("field %q must be > 0", name)
	}

	return value, nil
}
