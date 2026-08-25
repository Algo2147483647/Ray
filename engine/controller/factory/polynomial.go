package factory

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
)

func parsePolynomial(spec *parser.PolynomialSpec, bounds *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	if dimension != 3 {
		return nil, fmt.Errorf("polynomial requires scene dimension 3, got %d", dimension)
	}
	if spec.Degree == nil {
		return nil, fmt.Errorf("missing required field %q", "degree")
	}
	degree := *spec.Degree
	if degree <= 0 {
		return nil, fmt.Errorf("field %q must be > 0", "degree")
	}
	if len(spec.Terms) == 0 {
		return nil, fmt.Errorf("field %q must contain at least one term", "terms")
	}

	entries := make([]maths.SparseTensorEntry[float64], 0, len(spec.Terms))
	seen := make(map[[3]int]bool, len(spec.Terms))
	actualDegree := 0
	for index, term := range spec.Terms {
		if len(term.Exponents) != 3 || term.Coefficient == nil {
			return nil, fmt.Errorf("terms[%d] is incomplete", index)
		}
		key := [3]int{term.Exponents[0], term.Exponents[1], term.Exponents[2]}
		if seen[key] {
			return nil, fmt.Errorf("terms[%d] duplicates exponents %v", index, term.Exponents)
		}
		seen[key] = true
		termDegree := key[0] + key[1] + key[2]
		if termDegree > degree {
			return nil, fmt.Errorf("terms[%d] degree %d exceeds declared degree %d", index, termDegree, degree)
		}
		if termDegree > actualDegree {
			actualDegree = termDegree
		}
		if math.IsNaN(*term.Coefficient) || math.IsInf(*term.Coefficient, 0) {
			return nil, fmt.Errorf("terms[%d] coefficient must be finite", index)
		}
		if *term.Coefficient == 0 {
			return nil, fmt.Errorf("terms[%d] coefficient must be non-zero in sparse terms", index)
		}
		entries = append(entries, maths.SparseTensorEntry[float64]{
			Index: append([]int(nil), term.Exponents...), Value: *term.Coefficient,
		})
	}
	if actualDegree != degree {
		return nil, fmt.Errorf("declared degree %d does not match highest term degree %d", degree, actualDegree)
	}

	coefficientShape := []int{degree + 1, degree + 1, degree + 1}
	coefficients, err := maths.NewSparseTensorFromEntries(coefficientShape, maths.SparseTensorHash, entries)
	if err != nil {
		return nil, fmt.Errorf("build polynomial terms: %w", err)
	}
	polynomial := shape.NewPolynomial(coefficients)
	polynomial.Transform, err = parsePolynomialTransform(spec.Transform)
	if err != nil {
		return nil, err
	}
	return wrapSingleShapeWithBounds(polynomial, bounds, dimension)
}

func parsePolynomialTransform(raw json.RawMessage) ([4][4]float64, error) {
	if len(raw) == 0 {
		return maths.IdentityTransform4(), nil
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return [4][4]float64{}, fmt.Errorf("field %q: %w", "transform", err)
	}
	values, err := transformRows(value)
	if err != nil {
		return [4][4]float64{}, err
	}
	if len(values) != 4 {
		return [4][4]float64{}, fmt.Errorf("field %q must contain 4 rows, got %d", "transform", len(values))
	}

	transform := [4][4]float64{}
	for row, values := range values {
		if len(values) != 4 {
			return [4][4]float64{}, fmt.Errorf("transform[%d] must contain 4 values, got %d", row, len(values))
		}
		for col, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return [4][4]float64{}, fmt.Errorf("transform[%d][%d] must be finite", row, col)
			}
			transform[row][col] = value
		}
	}
	return transform, nil
}

func transformRows(raw interface{}) ([][]float64, error) {
	if rows, ok := raw.([]interface{}); ok {
		result := make([][]float64, len(rows))
		for i, row := range rows {
			values, err := utils.ToFloat64Slice(row)
			if err != nil {
				return nil, fmt.Errorf("transform[%d]: %w", i, err)
			}
			result[i] = values
		}
		return result, nil
	}
	values, err := utils.ToFloat64Slice(raw)
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
