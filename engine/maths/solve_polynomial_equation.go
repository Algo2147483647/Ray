package maths

import (
	"math"
	"math/cmplx"
)

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
		return DurandKerner(coeffs, DefaultTol, DefaultMaxIter)
	}
}

func DurandKerner(coeffs []complex128, tol float64, maxIter int) ([]complex128, error) {
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

func evalPolynomial(coeffs []complex128, x complex128) complex128 {
	result := complex(0, 0)

	for _, c := range coeffs {
		result = result*x + c
	}

	return result
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
