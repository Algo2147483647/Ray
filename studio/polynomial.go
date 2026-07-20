package main

import (
	"fmt"
	"strconv"
	"strings"

	"gonum.org/v1/gonum/mat"
)

func requiredPolynomialCoefficients(object map[string]interface{}, order int) ([]float64, error) {
	value, fieldName, err := requiredPolynomialCoefficientValue(object)
	if err != nil {
		return nil, err
	}

	total := 1
	for i := 0; i < order; i++ {
		total *= 4
	}

	if values, err := toFloat64Slice(value); err == nil {
		if len(values) != total {
			return nil, fmt.Errorf("field %q must contain %d values, got %d", fieldName, total, len(values))
		}
		return values, nil
	}

	sparse, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected array or object, got %T", fieldName, value)
	}
	return parseSparsePolynomialCoefficients(fieldName, sparse, order, total)
}

func requiredPolynomialCoefficientValue(object map[string]interface{}) (interface{}, string, error) {
	lower, hasLower := object["a"]
	upper, hasUpper := object["A"]
	if hasLower && hasUpper {
		return nil, "", fmt.Errorf(`fields "a" and "A" cannot both be provided`)
	}
	if hasLower {
		return lower, "a", nil
	}
	if hasUpper {
		return upper, "A", nil
	}
	return nil, "", fmt.Errorf(`missing required field "a"`)
}

func parseSparsePolynomialCoefficients(fieldName string, sparse map[string]interface{}, order, total int) ([]float64, error) {
	coefficients := make([]float64, total)
	keyStyle := ""
	for key, rawValue := range sparse {
		index, style, err := sparsePolynomialIndex(key, order, total)
		if err != nil {
			return nil, fmt.Errorf("field %q key %q: %w", fieldName, key, err)
		}
		if keyStyle == "" {
			keyStyle = style
		} else if keyStyle != style {
			return nil, fmt.Errorf("field %q cannot mix flat and coordinate sparse keys", fieldName)
		}
		value, err := toFloat64(rawValue)
		if err != nil {
			return nil, fmt.Errorf("field %q key %q: %w", fieldName, key, err)
		}
		coefficients[index] = value
	}
	return coefficients, nil
}

func sparsePolynomialIndex(key string, order, total int) (int, string, error) {
	if strings.Contains(key, ",") {
		index, err := sparsePolynomialCoordinateIndex(key, order)
		return index, "coordinate", err
	}
	index, err := strconv.Atoi(strings.TrimSpace(key))
	if err != nil {
		return 0, "", fmt.Errorf("expected integer flat index")
	}
	if index < 0 || index >= total {
		return 0, "", fmt.Errorf("flat index must be in [0,%d], got %d", total-1, index)
	}
	return index, "flat", nil
}

func sparsePolynomialCoordinateIndex(key string, order int) (int, error) {
	parts := strings.Split(key, ",")
	if len(parts) != order {
		return 0, fmt.Errorf("coordinate key must contain %d indices, got %d", order, len(parts))
	}
	index := 0
	for position, part := range parts {
		coordinate, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return 0, fmt.Errorf("coordinate %d must be an integer", position)
		}
		if coordinate < 0 || coordinate >= 4 {
			return 0, fmt.Errorf("coordinate %d must be in [0,3], got %d", position, coordinate)
		}
		index = index*4 + coordinate
	}
	return index, nil
}

func bakeCubicCoefficients(a []float64, center, scale [3]float64) []float64 {
	if center == [3]float64{} && scale == [3]float64{1, 1, 1} {
		result := make([]float64, len(a))
		copy(result, a)
		return result
	}

	matrix := [4][4]float64{
		{1, 0, 0, 0},
		{-center[0] / scale[0], 1 / scale[0], 0, 0},
		{-center[1] / scale[1], 0, 1 / scale[1], 0},
		{-center[2] / scale[2], 0, 0, 1 / scale[2]},
	}
	result := make([]float64, 64)
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				coef := a[cubicOffset(i, j, k)]
				if coef == 0 {
					continue
				}
				for wi := 0; wi < 4; wi++ {
					for wj := 0; wj < 4; wj++ {
						for wk := 0; wk < 4; wk++ {
							result[cubicOffset(wi, wj, wk)] += coef * matrix[i][wi] * matrix[j][wj] * matrix[k][wk]
						}
					}
				}
			}
		}
	}
	return result
}

func cubicOffset(i, j, k int) int {
	return (i*4+j)*4 + k
}

func bakeQuadraticCoefficients(aValues, bValues []float64, c float64, center, scale [3]float64) ([]float64, []float64, float64) {
	if center == [3]float64{} && scale == [3]float64{1, 1, 1} {
		return append([]float64(nil), aValues...), append([]float64(nil), bValues...), c
	}

	a := mat.NewDense(3, 3, aValues)
	b := mat.NewVecDense(3, bValues)
	d := mat.NewDense(3, 3, []float64{
		1 / scale[0], 0, 0,
		0, 1 / scale[1], 0,
		0, 0, 1 / scale[2],
	})
	e := mat.NewVecDense(3, []float64{
		-center[0] / scale[0],
		-center[1] / scale[1],
		-center[2] / scale[2],
	})

	var aD mat.Dense
	aD.Mul(a, d)

	var worldA mat.Dense
	worldA.Mul(d.T(), &aD)

	var aPlusAT mat.Dense
	aPlusAT.Add(a, a.T())

	tmp := mat.NewVecDense(3, nil)
	tmp.MulVec(&aPlusAT, e)
	worldB := mat.NewVecDense(3, nil)
	worldB.MulVec(d.T(), tmp)
	worldB.AddVec(worldB, scaledByDiagonal(b, d))

	aE := mat.NewVecDense(3, nil)
	aE.MulVec(a, e)
	worldC := mat.Dot(e, aE) + mat.Dot(b, e) + c

	return denseValues(&worldA), vecValues(worldB), worldC
}

func scaledByDiagonal(v *mat.VecDense, d *mat.Dense) *mat.VecDense {
	result := mat.NewVecDense(3, nil)
	for i := 0; i < 3; i++ {
		result.SetVec(i, v.AtVec(i)*d.At(i, i))
	}
	return result
}

func denseValues(m *mat.Dense) []float64 {
	rows, cols := m.Dims()
	values := make([]float64, 0, rows*cols)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			values = append(values, m.At(row, col))
		}
	}
	return values
}

func vecValues(v *mat.VecDense) []float64 {
	values := make([]float64, v.Len())
	for i := range values {
		values[i] = v.AtVec(i)
	}
	return values
}
