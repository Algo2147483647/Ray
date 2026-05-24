package maths

import (
	"math"
	"math/cmplx"
)

// SolveQuarticEquationAnalytical solves:
//
//	a4*x^4 + a3*x^3 + a2*x^2 + a1*x + a0 = 0
//
// using Ferrari's closed-form method.
//
// It returns all complex roots, including repeated roots.
func SolveQuarticEquationAnalytical(a4, a3, a2, a1, a0 float64) ([]complex128, error) {
	if isZero(a4) {
		return SolveCubicEquationAnalytical(a3, a2, a1, a0)
	}

	A := complex(a3/a4, 0)
	B := complex(a2/a4, 0)
	C := complex(a1/a4, 0)
	D := complex(a0/a4, 0)

	// Depressed quartic:
	//
	//	x = y - A/4
	//
	//	y^4 + p*y^2 + q*y + r = 0
	//
	p := B - 3*A*A/8
	q := A*A*A/8 - A*B/2 + C
	r := -3*A*A*A*A/256 + A*A*B/16 - A*C/4 + D

	shift := A / 4

	var yRoots []complex128

	if isZeroComplex(q) {
		// Biquadratic case:
		//
		//	y^4 + p*y^2 + r = 0
		//
		// Let z = y^2:
		//
		//	z^2 + p*z + r = 0
		//
		zRoots := solveQuadraticComplex(1, p, r)

		yRoots = make([]complex128, 0, 4)
		for _, z := range zRoots {
			sqrtZ := cmplx.Sqrt(z)
			yRoots = append(yRoots, sqrtZ, -sqrtZ)
		}
	} else {
		// Ferrari factorization:
		//
		//	y^4 + p*y^2 + q*y + r
		//
		// as:
		//
		//	(y^2 + alpha*y + beta)(y^2 - alpha*y + gamma)
		//
		// Let z = alpha^2. Then z satisfies:
		//
		//	z^3 + 2p*z^2 + (p^2 - 4r)z - q^2 = 0
		//
		zRoots, err := SolveCubicEquationAnalytical(
			1,
			real(2*p),
			real(p*p-4*r),
			real(-q*q),
		)
		if err != nil {
			return nil, err
		}

		// Pick a numerically useful z.
		// In exact arithmetic, any non-zero root works. Numerically, avoid
		// roots whose square root is too close to zero because q/alpha appears.
		z := selectBestFerrariZ(zRoots)

		alpha := cmplx.Sqrt(z)
		if isZeroComplex(alpha) {
			// Degenerate fallback:
			// Try another branch if available.
			found := false
			for _, candidate := range zRoots {
				a := cmplx.Sqrt(candidate)
				if !isZeroComplex(a) {
					z = candidate
					alpha = a
					found = true
					break
				}
			}
			if !found {
				return nil, ErrInvalidInput
			}
		}

		beta := (p + z - q/alpha) / 2
		gamma := (p + z + q/alpha) / 2

		roots1 := solveQuadraticComplex(1, alpha, beta)
		roots2 := solveQuadraticComplex(1, -alpha, gamma)

		yRoots = make([]complex128, 0, 4)
		yRoots = append(yRoots, roots1...)
		yRoots = append(yRoots, roots2...)
	}

	roots := make([]complex128, 0, 4)
	for _, y := range yRoots {
		roots = append(roots, cleanComplex(y-shift))
	}

	return roots, nil
}

// SolveCubicEquationAnalytical solves:
//
//	a*x^3 + b*x^2 + c*x + d = 0
//
// It returns all complex roots, including repeated roots.
func SolveCubicEquationAnalytical(a, b, c, d float64) ([]complex128, error) {
	if isZero(a) {
		return SolveQuadraticEquationAnalytical(b, c, d)
	}

	A := complex(b/a, 0)
	B := complex(c/a, 0)
	C := complex(d/a, 0)

	// Depressed cubic:
	//
	//	x = y - A/3
	//
	//	y^3 + p*y + q = 0
	//
	p := B - A*A/3
	q := 2*A*A*A/27 - A*B/3 + C

	discriminant := q*q/4 + p*p*p/27

	u := cmplx.Pow(-q/2+cmplx.Sqrt(discriminant), 1.0/3.0)
	v := cmplx.Pow(-q/2-cmplx.Sqrt(discriminant), 1.0/3.0)

	omega := complex(-0.5, math.Sqrt(3)/2)
	omega2 := complex(-0.5, -math.Sqrt(3)/2)

	shift := A / 3

	roots := []complex128{
		u + v - shift,
		omega*u + omega2*v - shift,
		omega2*u + omega*v - shift,
	}

	return cleanComplexRoots(roots), nil
}

// SolveQuadraticEquationAnalytical solves:
//
//	a*x^2 + b*x + c = 0
//
// It returns all complex roots, including repeated roots.
func SolveQuadraticEquationAnalytical(a, b, c float64) ([]complex128, error) {
	if isZero(a) {
		return SolveLinearEquationAnalytical(b, c)
	}

	return solveQuadraticComplex(
		complex(a, 0),
		complex(b, 0),
		complex(c, 0),
	), nil
}

// SolveLinearEquationAnalytical solves:
//
//	a*x + b = 0
func SolveLinearEquationAnalytical(a, b float64) ([]complex128, error) {
	if isZero(a) {
		if isZero(b) {
			return nil, ErrInfiniteSolutions
		}
		return nil, ErrNoSolution
	}

	return []complex128{complex(-b/a, 0)}, nil
}

func solveQuadraticComplex(a, b, c complex128) []complex128 {
	if isZeroComplex(a) {
		if isZeroComplex(b) {
			return nil
		}
		return []complex128{-c / b}
	}

	delta := b*b - 4*a*c
	sqrtDelta := cmplx.Sqrt(delta)

	return []complex128{
		(-b + sqrtDelta) / (2 * a),
		(-b - sqrtDelta) / (2 * a),
	}
}
func selectBestFerrariZ(zRoots []complex128) complex128 {
	if len(zRoots) == 0 {
		return 0
	}

	best := zRoots[0]
	bestAbs := cmplx.Abs(best)

	for _, z := range zRoots[1:] {
		absZ := cmplx.Abs(z)
		if absZ > bestAbs {
			best = z
			bestAbs = absZ
		}
	}

	return best
}

func cleanComplex(z complex128) complex128 {
	re := real(z)
	im := imag(z)

	if math.Abs(re) <= DefaultTol {
		re = 0
	}

	if math.Abs(im) <= DefaultTol {
		im = 0
	}

	return complex(re, im)
}

func cleanComplexRoots(roots []complex128) []complex128 {
	for i := range roots {
		roots[i] = cleanComplex(roots[i])
	}
	return roots
}
