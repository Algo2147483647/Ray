package maths

import (
	"math"
	"sort"
)

// SolvePolynomialReal solves a real-coefficient polynomial and returns its real roots.
//
// Coefficients must be ordered from highest degree to constant term:
//
//	coeffs = []float64{a_n, a_{n-1}, ..., a_1, a_0}
//
// Supported degree:
//
//	0: no roots
//	1: linear
//	2: quadratic
//	3: cubic
//	4: quartic
//
// For degree > 4, this function returns ErrDegreeTooHigh.
func SolvePolynomialReal(coeffs []float64) ([]float64, error) {
	coeffs = trimLeadingZerosReal(coeffs)
	if len(coeffs) == 0 {
		return nil, ErrZeroPolynomial
	}

	degree := len(coeffs) - 1

	switch degree {
	case 0:
		return nil, nil

	case 1:
		return SolveLinearEquation(coeffs[0], coeffs[1])

	case 2:
		return SolveQuadraticEquationReal(coeffs[0], coeffs[1], coeffs[2])

	case 3:
		return SolveCubicEquationReal(coeffs[0], coeffs[1], coeffs[2], coeffs[3])

	case 4:
		return SolveQuarticEquationReal(coeffs[0], coeffs[1], coeffs[2], coeffs[3], coeffs[4])

	default:
		return SolvePolynomialRealNumeric(coeffs)
	}
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

// SolveQuadraticEquationReal solves:
//
//	a*x^2 + b*x + c = 0
//
// It returns only real roots.
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
			return uniqueSortedRoots([]float64{
				(-b + sqrtDelta) / (2 * a),
				(-b - sqrtDelta) / (2 * a),
			}), nil
		}

		x1 := q / a
		x2 := c / q

		return uniqueSortedRoots([]float64{x1, x2}), nil
	}
}

// SolveCubicEquationReal solves:
//
//	a*x^3 + b*x^2 + c*x + d = 0
//
// It returns only real roots.
func SolveCubicEquationReal(a, b, c, d float64) ([]float64, error) {
	if isZero(a) {
		roots, err := SolveQuadraticEquationReal(b, c, d)
		if err != nil {
			return nil, err
		}
		return uniqueSortedRoots(roots), nil
	}

	// Normalize:
	//
	//	x^3 + aa*x^2 + bb*x + cc = 0
	//
	aa := b / a
	bb := c / a
	cc := d / a

	// Depressed cubic:
	//
	//	y^3 + p*y + q = 0
	//	x = y - aa/3
	//
	p := bb - aa*aa/3
	q := 2*aa*aa*aa/27 - aa*bb/3 + cc
	shift := aa / 3

	discriminant := q*q/4 + p*p*p/27

	roots := make([]float64, 0, 3)

	switch {
	case discriminant > DefaultTol:
		// One real root.
		sqrtDiscriminant := math.Sqrt(discriminant)

		u := math.Cbrt(-q/2 + sqrtDiscriminant)
		v := math.Cbrt(-q/2 - sqrtDiscriminant)

		roots = append(roots, u+v-shift)

	case discriminant < -DefaultTol:
		// Three distinct real roots.
		//
		// This branch theoretically requires p < 0.
		if p >= 0 {
			return []float64{}, nil
		}

		r := math.Sqrt(-p / 3)
		if isZero(r) {
			return []float64{}, nil
		}

		denom := 2 * r * r * r
		if isZero(denom) {
			return []float64{}, nil
		}

		arg := clamp(-q/denom, -1, 1)
		phi := math.Acos(arg)

		for k := 0; k < 3; k++ {
			angle := (phi + 2*math.Pi*float64(k)) / 3
			roots = append(roots, 2*r*math.Cos(angle)-shift)
		}

	default:
		// discriminant ~= 0.
		if math.Abs(q) <= DefaultTol && math.Abs(p) <= DefaultTol {
			// Triple root.
			roots = append(roots, -shift)
		} else {
			u := math.Cbrt(-q / 2)

			// Double root case:
			//
			//	y1 = 2u
			//	y2 = -u
			//
			roots = append(roots, 2*u-shift, -u-shift)
		}
	}

	return uniqueSortedRoots(roots), nil
}

// SolveQuarticEquationReal solves:
//
//	a4*x^4 + a3*x^3 + a2*x^2 + a1*x + a0 = 0
//
// It returns only real roots.
//
// This implementation uses derivative critical points to split the real line
// into monotonic intervals, then applies bisection on intervals with sign changes.
// It also checks critical points directly so even-multiplicity roots are not missed.
func SolveQuarticEquationReal(a4, a3, a2, a1, a0 float64) ([]float64, error) {
	if isZero(a4) {
		return SolveCubicEquationReal(a3, a2, a1, a0)
	}

	// Cauchy root bound:
	//
	//	Every real root satisfies |x| <= 1 + max(|a_i/a4|)
	//
	bound := 1 + max4(
		math.Abs(a3/a4),
		math.Abs(a2/a4),
		math.Abs(a1/a4),
		math.Abs(a0/a4),
	)

	// Derivative:
	//
	//	4*a4*x^3 + 3*a3*x^2 + 2*a2*x + a1
	//
	criticalPoints, err := SolveCubicEquationReal(4*a4, 3*a3, 2*a2, a1)
	if err != nil {
		return nil, err
	}

	points := make([]float64, 0, len(criticalPoints)+2)
	points = append(points, -bound)

	for _, p := range criticalPoints {
		if p > -bound && p < bound {
			points = append(points, p)
		}
	}

	points = append(points, bound)
	points = uniqueSortedRoots(points)

	const valueTol = 1e-9

	roots := make([]float64, 0, 4)

	// Check every split point directly.
	// This catches boundary roots and even-multiplicity roots at critical points.
	for _, p := range points {
		if math.Abs(evalQuartic(a4, a3, a2, a1, a0, p)) <= valueTol {
			roots = append(roots, p)
		}
	}

	// Check each monotonic interval.
	for i := 0; i+1 < len(points); i++ {
		left := points[i]
		right := points[i+1]

		if right-left <= DefaultTol {
			continue
		}

		fLeft := evalQuartic(a4, a3, a2, a1, a0, left)
		fRight := evalQuartic(a4, a3, a2, a1, a0, right)

		// Already captured exact or near-exact endpoint roots above.
		if math.Abs(fLeft) <= valueTol || math.Abs(fRight) <= valueTol {
			continue
		}

		// No sign change means no odd-multiplicity root in this monotonic interval.
		if sameSign(fLeft, fRight) {
			continue
		}

		root := bisectQuarticRoot(a4, a3, a2, a1, a0, left, right)
		roots = append(roots, root)
	}

	return uniqueSortedRoots(roots), nil
}

// SolvePolynomialRealNumeric solves real roots for arbitrary-degree real polynomials.
//
// Coefficients must be ordered from highest degree to constant term:
//
//	coeffs = []float64{a_n, a_{n-1}, ..., a_1, a_0}
//
// This function returns only real roots.
//
// Method:
//   - recursively solve roots of the derivative
//   - split the real line into monotonic intervals
//   - use bisection where sign changes occur
//   - check derivative critical points to catch even-multiplicity roots
func SolvePolynomialRealNumeric(coeffs []float64) ([]float64, error) {
	coeffs = trimLeadingZerosReal(coeffs)
	if len(coeffs) == 0 {
		return nil, ErrZeroPolynomial
	}

	degree := len(coeffs) - 1
	if degree == 0 {
		return nil, nil
	}

	if degree == 1 {
		return SolveLinearEquation(coeffs[0], coeffs[1])
	}

	// For low degree, use existing specialized solvers.
	switch degree {
	case 2:
		return SolveQuadraticEquationReal(coeffs[0], coeffs[1], coeffs[2])
	case 3:
		return SolveCubicEquationReal(coeffs[0], coeffs[1], coeffs[2], coeffs[3])
	case 4:
		return SolveQuarticEquationReal(coeffs[0], coeffs[1], coeffs[2], coeffs[3], coeffs[4])
	}

	bound := polynomialRootBound(coeffs)

	derivative := derivativePolynomial(coeffs)

	criticalPoints, err := SolvePolynomialRealNumeric(derivative)
	if err != nil {
		return nil, err
	}

	points := make([]float64, 0, len(criticalPoints)+2)
	points = append(points, -bound)

	for _, p := range criticalPoints {
		if p > -bound && p < bound {
			points = append(points, p)
		}
	}

	points = append(points, bound)
	points = uniqueSortedRoots(points)

	roots := make([]float64, 0, degree)

	valueTol := polynomialValueTol(coeffs)

	// 1. Check split points directly.
	//
	// This catches:
	//   - boundary roots
	//   - even-multiplicity roots
	//   - roots located exactly at derivative critical points
	for _, p := range points {
		if math.Abs(evalPolynomialReal(coeffs, p)) <= valueTol {
			roots = append(roots, p)
		}
	}

	// 2. Search monotonic intervals.
	for i := 0; i+1 < len(points); i++ {
		left := points[i]
		right := points[i+1]

		if right-left <= DefaultTol {
			continue
		}

		fLeft := evalPolynomialReal(coeffs, left)
		fRight := evalPolynomialReal(coeffs, right)

		// Endpoint roots were already captured.
		if math.Abs(fLeft) <= valueTol || math.Abs(fRight) <= valueTol {
			continue
		}

		if sameSign(fLeft, fRight) {
			continue
		}

		root := bisectPolynomialRoot(coeffs, left, right)
		roots = append(roots, root)
	}

	return uniqueSortedRoots(roots), nil
}

func evalPolynomialReal(coeffs []float64, x float64) float64 {
	if len(coeffs) == 0 {
		return 0
	}

	result := coeffs[0]
	for i := 1; i < len(coeffs); i++ {
		result = result*x + coeffs[i]
	}

	return result
}

func derivativePolynomial(coeffs []float64) []float64 {
	degree := len(coeffs) - 1
	if degree <= 0 {
		return nil
	}

	derivative := make([]float64, degree)

	for i := 0; i < degree; i++ {
		power := float64(degree - i)
		derivative[i] = coeffs[i] * power
	}

	return trimLeadingZerosReal(derivative)
}

func polynomialRootBound(coeffs []float64) float64 {
	coeffs = trimLeadingZerosReal(coeffs)
	if len(coeffs) == 0 {
		return 0
	}

	leading := math.Abs(coeffs[0])
	if leading <= DefaultTol {
		return 0
	}

	maxRatio := 0.0
	for i := 1; i < len(coeffs); i++ {
		maxRatio = math.Max(maxRatio, math.Abs(coeffs[i])/leading)
	}

	// Cauchy bound:
	//
	// Every real root satisfies:
	//
	//	|x| <= 1 + max(|a_i / a_n|)
	return 1 + maxRatio
}

func bisectPolynomialRoot(coeffs []float64, left, right float64) float64 {
	fLeft := evalPolynomialReal(coeffs, left)

	for i := 0; i < DefaultMaxIter; i++ {
		mid := 0.5 * (left + right)
		fMid := evalPolynomialReal(coeffs, mid)

		if math.Abs(fMid) <= DefaultTol || math.Abs(right-left) <= DefaultTol {
			return mid
		}

		if !sameSign(fLeft, fMid) {
			right = mid
		} else {
			left = mid
			fLeft = fMid
		}
	}

	return 0.5 * (left + right)
}

func polynomialValueTol(coeffs []float64) float64 {
	scale := 0.0
	for _, c := range coeffs {
		scale = math.Max(scale, math.Abs(c))
	}

	if scale == 0 {
		return DefaultTol
	}

	// Slightly looser than DefaultTol because high-degree polynomial evaluation
	// can accumulate more floating-point error.
	return 1e-9 * scale
}

func evalQuartic(a4, a3, a2, a1, a0, x float64) float64 {
	return (((a4*x+a3)*x+a2)*x+a1)*x + a0
}

func bisectQuarticRoot(a4, a3, a2, a1, a0, left, right float64) float64 {
	fLeft := evalQuartic(a4, a3, a2, a1, a0, left)

	for i := 0; i < DefaultMaxIter; i++ {
		mid := 0.5 * (left + right)
		fMid := evalQuartic(a4, a3, a2, a1, a0, mid)

		if math.Abs(fMid) <= DefaultTol || math.Abs(right-left) <= DefaultTol {
			return mid
		}

		if !sameSign(fLeft, fMid) {
			right = mid
		} else {
			left = mid
			fLeft = fMid
		}
	}

	return 0.5 * (left + right)
}

func uniqueSortedRoots(roots []float64) []float64 {
	sort.Float64s(roots)

	result := roots[:0]
	for _, root := range roots {
		if math.IsNaN(root) || math.IsInf(root, 0) {
			continue
		}

		if len(result) == 0 || math.Abs(root-result[len(result)-1]) > 1e-8 {
			result = append(result, root)
		}
	}

	return result
}

func sameSign(a, b float64) bool {
	return a*b > 0
}

func clamp(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func max4(a, b, c, d float64) float64 {
	return math.Max(math.Max(a, b), math.Max(c, d))
}

func trimLeadingZerosReal(coeffs []float64) []float64 {
	scale := 0.0
	for _, c := range coeffs {
		scale = math.Max(scale, math.Abs(c))
	}

	if scale == 0 {
		return nil
	}

	i := 0
	for i < len(coeffs) && math.Abs(coeffs[i]) <= DefaultTol*scale {
		i++
	}

	return coeffs[i:]
}
