package adapt

import (
	"fmt"
	"math"
)

func worldToLocalTransformMatrix(ctx groupContext, localCenter, localScale []float64, basis [][]float64) [4][4]float64 {
	matrix := [4][4]float64{{1, 0, 0, 0}}
	for localAxis := 0; localAxis < 3; localAxis++ {
		scale := localScale[localAxis]
		for worldAxis := 0; worldAxis < 3; worldAxis++ {
			groupScale := ctx.scale[worldAxis]
			matrix[localAxis+1][0] -= basis[localAxis][worldAxis] * (ctx.center[worldAxis] + groupScale*localCenter[worldAxis]) / (groupScale * scale)
			matrix[localAxis+1][worldAxis+1] = basis[localAxis][worldAxis] / (groupScale * scale)
		}
	}
	return matrix
}

func optionalTransform(object map[string]interface{}) ([4][4]float64, bool, error) {
	raw, ok := object["transform"]
	if !ok {
		return [4][4]float64{}, false, nil
	}

	rows, err := transformRows(raw)
	if err != nil {
		return [4][4]float64{}, true, err
	}
	if len(rows) != 4 {
		return [4][4]float64{}, true, fmt.Errorf("field %q must contain 4 rows, got %d", "transform", len(rows))
	}

	transform := [4][4]float64{}
	for row, values := range rows {
		if len(values) != 4 {
			return [4][4]float64{}, true, fmt.Errorf("transform[%d] must contain 4 values, got %d", row, len(values))
		}
		for col, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return [4][4]float64{}, true, fmt.Errorf("transform[%d][%d] must be finite", row, col)
			}
			transform[row][col] = value
		}
	}
	return transform, true, nil
}

func transformRows(raw interface{}) ([][]float64, error) {
	switch rows := raw.(type) {
	case []interface{}:
		result := make([][]float64, len(rows))
		for row, rawRow := range rows {
			values, err := toFloat64Slice(rawRow)
			if err != nil {
				return nil, fmt.Errorf("transform[%d]: %w", row, err)
			}
			result[row] = values
		}
		return result, nil
	case [][]float64:
		result := make([][]float64, len(rows))
		for row := range rows {
			result[row] = append([]float64(nil), rows[row]...)
		}
		return result, nil
	case [4][4]float64:
		return transformToSlices(rows), nil
	}

	values, err := toFloat64Slice(raw)
	if err != nil {
		return nil, fmt.Errorf("field %q: expected 4x4 array, got %T", "transform", raw)
	}
	if len(values) != 16 {
		return nil, fmt.Errorf("field %q must contain 16 flat values, got %d", "transform", len(values))
	}
	result := make([][]float64, 4)
	for row := range result {
		result[row] = append([]float64(nil), values[row*4:(row+1)*4]...)
	}
	return result, nil
}

func composeWithGroupInverse(transform [4][4]float64, ctx groupContext) [4][4]float64 {
	if groupPlacementIsIdentity(ctx) {
		return transform
	}
	groupInverse := [4][4]float64{{1, 0, 0, 0}}
	for axis := 0; axis < 3; axis++ {
		groupInverse[axis+1][0] = -ctx.center[axis] / ctx.scale[axis]
		groupInverse[axis+1][axis+1] = 1 / ctx.scale[axis]
	}
	return multiplyTransform4(transform, groupInverse)
}

func multiplyTransform4(a, b [4][4]float64) [4][4]float64 {
	var result [4][4]float64
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			for k := 0; k < 4; k++ {
				result[row][col] += a[row][k] * b[k][col]
			}
		}
	}
	return result
}

func transformToSlices(transform [4][4]float64) [][]float64 {
	result := make([][]float64, 4)
	for row := range result {
		result[row] = make([]float64, 4)
		copy(result[row], transform[row][:])
	}
	return result
}

func optionalBasis(object map[string]interface{}, dimension int) ([][]float64, error) {
	raw, ok := object["basis"]
	if !ok {
		return identityBasis(dimension), nil
	}
	rows, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected array, got %T", "basis", raw)
	}
	if len(rows) != dimension {
		return nil, fmt.Errorf("field %q must contain %d vectors, got %d", "basis", dimension, len(rows))
	}

	basis := make([][]float64, dimension)
	for i, rawRow := range rows {
		row, err := toFloat64Slice(rawRow)
		if err != nil {
			return nil, fmt.Errorf("basis[%d]: %w", i, err)
		}
		if len(row) != dimension {
			return nil, fmt.Errorf("basis[%d] must contain %d values, got %d", i, dimension, len(row))
		}
		basis[i] = row
	}
	if err := validateOrthonormalBasis(basis); err != nil {
		return nil, err
	}
	return basis, nil
}

func validateOrthonormalBasis(basis [][]float64) error {
	const tol = 1e-6
	for i, row := range basis {
		lengthSquared := 0.0
		for j, value := range row {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("basis[%d][%d] must be finite", i, j)
			}
			lengthSquared += value * value
		}
		if math.Abs(lengthSquared-1) > tol {
			return fmt.Errorf("basis[%d] must be unit length", i)
		}
		for j := i + 1; j < len(basis); j++ {
			dot := 0.0
			for axis, value := range row {
				dot += value * basis[j][axis]
			}
			if math.Abs(dot) > tol {
				return fmt.Errorf("basis[%d] and basis[%d] must be orthogonal", i, j)
			}
		}
	}
	return nil
}

func identityBasis(dimension int) [][]float64 {
	basis := make([][]float64, dimension)
	for i := range basis {
		basis[i] = make([]float64, dimension)
		basis[i][i] = 1
	}
	return basis
}
