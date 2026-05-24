package maths

import (
	"fmt"
	"math"
)

// RealRoots filters approximately real roots from complex roots.
func RealRoots(roots []complex128, tol float64) []float64 {
	if tol <= 0 {
		tol = DefaultTol
	}

	result := make([]float64, 0, len(roots))

	for _, r := range roots {
		if math.Abs(imag(r)) <= tol {
			result = append(result, real(r))
		}
	}

	return result
}

// NewtonOptions controls Newton-Raphson behavior.
type NewtonOptions struct {
	Tol         float64
	MaxIter     int
	JacobianEps float64
	Damping     bool
}

func defaultNewtonOptions(options *NewtonOptions) NewtonOptions {
	if options == nil {
		return NewtonOptions{
			Tol:         1e-10,
			MaxIter:     100,
			JacobianEps: 1e-6,
			Damping:     true,
		}
	}

	opt := *options

	if opt.Tol <= 0 {
		opt.Tol = 1e-10
	}
	if opt.MaxIter <= 0 {
		opt.MaxIter = 100
	}
	if opt.JacobianEps <= 0 {
		opt.JacobianEps = 1e-6
	}

	return opt
}

// NewtonRaphson solves a nonlinear square system:
//
//	f(x) = 0
//
// f must return the same number of equations as len(x0).
func NewtonRaphson(
	f func([]float64) []float64,
	x0 []float64,
	options *NewtonOptions,
) ([]float64, error) {
	if f == nil {
		return nil, ErrInvalidInput
	}
	if len(x0) == 0 {
		return nil, ErrInvalidInput
	}

	opt := defaultNewtonOptions(options)

	x := make([]float64, len(x0))
	copy(x, x0)

	for iter := 0; iter < opt.MaxIter; iter++ {
		fx := f(x)

		if len(fx) != len(x) {
			return nil, fmt.Errorf("%w: len(f(x))=%d, len(x)=%d", ErrInvalidInput, len(fx), len(x))
		}

		if infNorm(fx) < opt.Tol {
			return x, nil
		}

		jacobian, err := NumericalJacobian(f, x, opt.JacobianEps)
		if err != nil {
			return nil, err
		}

		rhs := make([]float64, len(fx))
		for i := range rhs {
			rhs[i] = -fx[i]
		}

		delta, err := SolveLinearSystem(jacobian, rhs)
		if err != nil {
			return nil, err
		}

		if infNorm(delta) < opt.Tol {
			for i := range x {
				x[i] += delta[i]
			}
			return x, nil
		}

		if opt.Damping {
			alpha := 1.0
			currentNorm := infNorm(fx)
			accepted := false

			for alpha >= 1e-8 {
				candidate := make([]float64, len(x))
				for i := range x {
					candidate[i] = x[i] + alpha*delta[i]
				}

				fCandidate := f(candidate)
				if len(fCandidate) != len(x) {
					return nil, ErrInvalidInput
				}

				if infNorm(fCandidate) < currentNorm {
					x = candidate
					accepted = true
					break
				}

				alpha *= 0.5
			}

			if !accepted {
				// Fall back to the full Newton step.
				for i := range x {
					x[i] += delta[i]
				}
			}
		} else {
			for i := range x {
				x[i] += delta[i]
			}
		}
	}

	return nil, ErrNotConverged
}

// NumericalJacobian computes the central-difference numerical Jacobian.
//
// It returns an m*n matrix, where:
//
//	m = len(f(x))
//	n = len(x)
func NumericalJacobian(
	f func([]float64) []float64,
	x []float64,
	eps float64,
) ([][]float64, error) {
	if f == nil {
		return nil, ErrInvalidInput
	}
	if len(x) == 0 {
		return nil, ErrInvalidInput
	}
	if eps <= 0 {
		eps = 1e-6
	}

	fx := f(x)
	m := len(fx)
	n := len(x)

	if m == 0 {
		return nil, ErrInvalidInput
	}

	jacobian := make([][]float64, m)
	for i := range jacobian {
		jacobian[i] = make([]float64, n)
	}

	for j := 0; j < n; j++ {
		h := eps * math.Max(1, math.Abs(x[j]))

		xPlus := make([]float64, n)
		xMinus := make([]float64, n)

		copy(xPlus, x)
		copy(xMinus, x)

		xPlus[j] += h
		xMinus[j] -= h

		fPlus := f(xPlus)
		fMinus := f(xMinus)

		if len(fPlus) != m || len(fMinus) != m {
			return nil, ErrInvalidInput
		}

		for i := 0; i < m; i++ {
			jacobian[i][j] = (fPlus[i] - fMinus[i]) / (2 * h)
		}
	}

	return jacobian, nil
}

// SolveLinearSystem solves:
//
//	A*x = b
//
// using Gaussian elimination with partial pivoting.
func SolveLinearSystem(A [][]float64, b []float64) ([]float64, error) {
	n := len(b)

	if n == 0 {
		return nil, ErrInvalidInput
	}
	if len(A) != n {
		return nil, ErrNonSquareSystem
	}

	for i := range A {
		if len(A[i]) != n {
			return nil, ErrNonSquareSystem
		}
	}

	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, n+1)
		copy(aug[i], A[i])
		aug[i][n] = b[i]
	}

	for i := 0; i < n; i++ {
		pivotRow := i
		pivotAbs := math.Abs(aug[i][i])

		for r := i + 1; r < n; r++ {
			v := math.Abs(aug[r][i])
			if v > pivotAbs {
				pivotAbs = v
				pivotRow = r
			}
		}

		if pivotAbs <= DefaultTol {
			return nil, ErrSingularMatrix
		}

		if pivotRow != i {
			aug[i], aug[pivotRow] = aug[pivotRow], aug[i]
		}

		for r := i + 1; r < n; r++ {
			factor := aug[r][i] / aug[i][i]
			aug[r][i] = 0

			for c := i + 1; c <= n; c++ {
				aug[r][c] -= factor * aug[i][c]
			}
		}
	}

	x := make([]float64, n)

	for i := n - 1; i >= 0; i-- {
		sum := aug[i][n]

		for j := i + 1; j < n; j++ {
			sum -= aug[i][j] * x[j]
		}

		if math.Abs(aug[i][i]) <= DefaultTol {
			return nil, ErrSingularMatrix
		}

		x[i] = sum / aug[i][i]
	}

	return x, nil
}

func infNorm(v []float64) float64 {
	maxVal := 0.0

	for _, x := range v {
		ax := math.Abs(x)
		if ax > maxVal {
			maxVal = ax
		}
	}

	return maxVal
}
