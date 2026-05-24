package maths

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"
)

const (
	DefaultTol     = 1e-12
	DefaultMaxIter = 10000
)

var (
	ErrNoSolution        = errors.New("no solution")
	ErrInfiniteSolutions = errors.New("infinite solutions")
	ErrInvalidInput      = errors.New("invalid input")
	ErrNonSquareSystem   = errors.New("linear system must be square")
	ErrSingularMatrix    = errors.New("singular matrix")
	ErrNotConverged      = errors.New("not converged")
	ErrDegreeTooHigh     = errors.New("polynomial degree too high")
	ErrZeroPolynomial    = errors.New("zero polynomial")
)

func isZero(x float64) bool {
	return math.Abs(x) <= DefaultTol
}

func isZeroComplex(z complex128) bool {
	return cmplx.Abs(z) <= DefaultTol
}

// SolveLinearEquation solves:
//
//	a*x + b = 0
//
// Return values:
//   - one root: []float64{x}
//   - no solution: ErrNoSolution
//   - infinite solutions: ErrInfiniteSolutions
func SolveLinearEquation(a, b float64) ([]float64, error) {
	if isZero(a) {
		if isZero(b) {
			return nil, ErrInfiniteSolutions
		}
		return nil, ErrNoSolution
	}

	return []float64{-b / a}, nil
}

// SolveQuadraticEquation solves:
//
//	a*x^2 + b*x + c = 0
//
// It returns all roots as complex numbers.
func SolveQuadraticEquation(a, b, c float64) ([]complex128, error) {
	if isZero(a) {
		roots, err := SolveLinearEquation(b, c)
		if err != nil {
			return nil, err
		}
		return []complex128{complex(roots[0], 0)}, nil
	}

	delta := complex(b*b-4*a*c, 0)
	sqrtDelta := cmplx.Sqrt(delta)
	denom := complex(2*a, 0)

	return []complex128{
		(-complex(b, 0) + sqrtDelta) / denom,
		(-complex(b, 0) - sqrtDelta) / denom,
	}, nil
}

// SolveQuadraticEquationReal solves:
//
//	a*x^2 + b*x + c = 0
//
// It returns only real roots.
//
// Return values:
//   - len(roots) == 0: no real root
//   - len(roots) == 1: one repeated real root or linear root
//   - len(roots) == 2: two distinct real roots
func SolveQuadraticEquationReal(a, b, c float64) ([]float64, error) {
	if isZero(a) {
		return SolveLinearEquation(b, c)
	}

	delta := b*b - 4*a*c

	switch {
	case delta < -DefaultTol:
		return []float64{}, nil

	case math.Abs(delta) <= DefaultTol:
		return []float64{-b / (2 * a)}, nil

	default:
		sqrtDelta := math.Sqrt(delta)

		// Numerically stable quadratic formula.
		q := -0.5 * (b + math.Copysign(sqrtDelta, b))

		if isZero(q) {
			// Fallback. This case is rare but avoids division by zero.
			return []float64{
				(-b + sqrtDelta) / (2 * a),
				(-b - sqrtDelta) / (2 * a),
			}, nil
		}

		x1 := q / a
		x2 := c / q

		return []float64{x1, x2}, nil
	}
}

// SolveCubicEquation solves:
//
//	a*x^3 + b*x^2 + c*x + d = 0
//
// It uses a general numerical polynomial solver instead of Cardano's formula.
// This is usually more robust for library usage.
func SolveCubicEquation(a, b, c, d float64) ([]complex128, error) {
	return SolvePolynomial([]complex128{
		complex(a, 0),
		complex(b, 0),
		complex(c, 0),
		complex(d, 0),
	})
}

// SolveQuarticEquation solves:
//
//	a*x^4 + b*x^3 + c*x^2 + d*x + e = 0
//
// It uses a general numerical polynomial solver instead of Ferrari's formula.
// This avoids many branch and division-by-zero problems in the closed-form formula.
func SolveQuarticEquation(a, b, c, d, e float64) ([]complex128, error) {
	return SolvePolynomial([]complex128{
		complex(a, 0),
		complex(b, 0),
		complex(c, 0),
		complex(d, 0),
		complex(e, 0),
	})
}

// SolvePolynomial solves a polynomial with complex coefficients.
//
// Coefficients must be ordered from highest degree to constant term:
//
//	a_n*x^n + a_{n-1}*x^{n-1} + ... + a_1*x + a_0
//
// Example:
//
//	x^3 - 1 = 0
//
// should be passed as:
//
//	[]complex128{1, 0, 0, -1}
//
// For degree 1 and 2, it uses direct formulas.
// For degree 3 and above, it uses the Durand-Kerner method.
func SolvePolynomial(coeffs []complex128) ([]complex128, error) {
	coeffs = trimLeadingZeros(coeffs)

	if len(coeffs) == 0 {
		return nil, ErrZeroPolynomial
	}

	degree := len(coeffs) - 1

	switch degree {
	case 0:
		if isZeroComplex(coeffs[0]) {
			return nil, ErrZeroPolynomial
		}
		return nil, ErrNoSolution

	case 1:
		a := coeffs[0]
		b := coeffs[1]
		if isZeroComplex(a) {
			return nil, ErrInvalidInput
		}
		return []complex128{-b / a}, nil

	case 2:
		a := coeffs[0]
		b := coeffs[1]
		c := coeffs[2]

		if isZeroComplex(a) {
			return SolvePolynomial(coeffs[1:])
		}

		delta := b*b - 4*a*c
		sqrtDelta := cmplx.Sqrt(delta)
		denom := 2 * a

		return []complex128{
			(-b + sqrtDelta) / denom,
			(-b - sqrtDelta) / denom,
		}, nil

	default:
		return durandKerner(coeffs, DefaultTol, DefaultMaxIter)
	}
}

func trimLeadingZeros(coeffs []complex128) []complex128 {
	i := 0
	for i < len(coeffs) && isZeroComplex(coeffs[i]) {
		i++
	}
	return coeffs[i:]
}

func trimTrailingZeros(coeffs []complex128) ([]complex128, int) {
	zeroRoots := 0

	for len(coeffs) > 0 && isZeroComplex(coeffs[len(coeffs)-1]) {
		coeffs = coeffs[:len(coeffs)-1]
		zeroRoots++
	}

	return coeffs, zeroRoots
}

func evalPolynomial(coeffs []complex128, x complex128) complex128 {
	result := complex(0, 0)

	for _, c := range coeffs {
		result = result*x + c
	}

	return result
}

func durandKerner(coeffs []complex128, tol float64, maxIter int) ([]complex128, error) {
	coeffs = trimLeadingZeros(coeffs)
	if len(coeffs) == 0 {
		return nil, ErrZeroPolynomial
	}

	// Factor out zero roots first.
	var zeroRoots int
	coeffs, zeroRoots = trimTrailingZeros(coeffs)

	if len(coeffs) == 0 {
		roots := make([]complex128, zeroRoots)
		return roots, nil
	}

	degree := len(coeffs) - 1

	if degree <= 2 {
		roots, err := SolvePolynomial(coeffs)
		if err != nil {
			return nil, err
		}

		for i := 0; i < zeroRoots; i++ {
			roots = append(roots, 0)
		}

		return roots, nil
	}

	// Normalize to monic polynomial.
	lead := coeffs[0]
	if isZeroComplex(lead) {
		return nil, ErrInvalidInput
	}

	normalized := make([]complex128, len(coeffs))
	for i := range coeffs {
		normalized[i] = coeffs[i] / lead
	}

	// Cauchy bound for initial radius.
	radius := 1.0
	for i := 1; i < len(normalized); i++ {
		radius = math.Max(radius, 1+cmplx.Abs(normalized[i]))
	}

	roots := make([]complex128, degree)

	// Initial roots placed on a circle.
	// Slight radius perturbation helps avoid symmetry traps.
	for i := 0; i < degree; i++ {
		angle := 2 * math.Pi * float64(i) / float64(degree)
		r := radius * (1 + 0.01*float64(i))
		roots[i] = cmplx.Rect(r, angle)
	}

	for iter := 0; iter < maxIter; iter++ {
		maxDelta := 0.0

		for i := 0; i < degree; i++ {
			denom := complex(1, 0)

			for j := 0; j < degree; j++ {
				if i == j {
					continue
				}

				diff := roots[i] - roots[j]

				if cmplx.Abs(diff) < tol {
					// Perturb nearly identical roots to avoid division by zero.
					diff += complex(tol*float64(i+1), tol*float64(j+1))
				}

				denom *= diff
			}

			if isZeroComplex(denom) {
				return nil, ErrNotConverged
			}

			delta := evalPolynomial(normalized, roots[i]) / denom
			roots[i] -= delta

			if cmplx.Abs(delta) > maxDelta {
				maxDelta = cmplx.Abs(delta)
			}
		}

		if maxDelta < tol {
			// Residual validation.
			for _, r := range roots {
				if cmplx.Abs(evalPolynomial(normalized, r)) > 1e-7 {
					return nil, ErrNotConverged
				}
			}

			for i := 0; i < zeroRoots; i++ {
				roots = append(roots, 0)
			}

			return roots, nil
		}
	}

	return nil, ErrNotConverged
}

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

// PolynomialResidual returns |P(x)|.
func PolynomialResidual(coeffs []complex128, x complex128) float64 {
	return cmplx.Abs(evalPolynomial(coeffs, x))
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
