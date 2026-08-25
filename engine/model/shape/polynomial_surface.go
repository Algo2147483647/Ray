package shape

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"gonum.org/v1/gonum/mat"
)

type Polynomial struct {
	BaseShape
	InputDim     int                          // Number of local input coordinates used by the implicit polynomial.
	Coefficients *maths.SparseTensor[float64] // Sparse polynomial coefficients indexed by exponents.
	Transform    [4][4]float64                // World-to-local homogeneous transform matrix.
	Mem          PolynomialCalculateStorage
}

type PolynomialCalculateStorage struct {
	Degree    int
	Terms     []polynomialTerm
	Quadratic quadraticPolynomialKernel
}

type polynomialTerm struct {
	Exponents []int
	Value     float64
}

type quadraticPolynomialKernel struct {
	Constant float64
	Linear   [3]float64
	Square   [3]float64
	Cross    [3][3]float64
}

func NewPolynomial(coefficients *maths.SparseTensor[float64]) *Polynomial {
	surface := &Polynomial{
		InputDim:     3,
		Coefficients: coefficients,
		Transform:    identityTransform4(),
	}
	surface.RebuildCalculateStorage()
	return surface
}

func (p *Polynomial) RebuildCalculateStorage() {
	if p == nil {
		return
	}
	p.Mem = buildPolynomialCalculateStorage(p.InputDim, p.Coefficients)
}

func (p *Polynomial) Name() string {
	return "Polynomial"
}

func (p *Polynomial) IntersectAffine(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if !options.valid() {
		return SurfaceInteraction{}, false
	}
	if p == nil || raySt == nil || rayDir == nil || raySt.Len() < 3 || rayDir.Len() < 3 {
		return SurfaceInteraction{}, false
	}

	coeffs, err := p.rayPolynomial(raySt, rayDir)
	if err != nil {
		return SurfaceInteraction{}, false
	}

	roots, err := maths.SolvePolynomialReal(coeffs)
	if err != nil {
		return SurfaceInteraction{}, false
	}

	bestT := math.MaxFloat64
	for _, root := range roots {
		if distanceInRange(root, options.Range.Min, options.Range.Max) && root < bestT {
			bestT = root
		}
	}
	if bestT == math.MaxFloat64 {
		return SurfaceInteraction{}, false
	}

	point := affinePointAt(raySt, rayDir, bestT)
	normal := p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteractionAt(point, bestT, normal), true
}

func (p *Polynomial) IntersectGeodesic(rayStart, rayDir *mat.VecDense, g geometry.Geometry, options IntersectOptions) (SurfaceInteraction, bool) {
	if !supportsSphericalGeodesic(g, options) {
		return SurfaceInteraction{}, false
	}
	if p == nil || p.Coefficients == nil || rayStart.Len() != rayDir.Len() || p.InputDim > rayStart.Len() {
		return SurfaceInteraction{}, false
	}
	sMin, sMax := options.Range.Min, options.Range.Max
	return intersectSphericalScalar(rayStart, rayDir, sMin, sMax, func(point *mat.VecDense) float64 {
		local := p.localPoint(point)
		return p.Evaluate(local[:p.InputDim])
	}, func(point *mat.VecDense) *mat.VecDense {
		return p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	})
}

func (p *Polynomial) Evaluate(input []float64) float64 {
	if p == nil || p.Coefficients == nil || len(input) != p.InputDim {
		return math.NaN()
	}

	mem := p.calculateStorage()
	if mem.Degree <= 2 {
		return mem.Quadratic.evaluate(input)
	}
	powers := precomputePowers(input, mem.Degree)
	result := 0.0

	for _, polynomialTerm := range mem.Terms {
		term := polynomialTerm.Value
		for axis, exponent := range polynomialTerm.Exponents {
			term *= powers[axis][exponent]
		}
		result += term
	}
	return result
}

func (p *Polynomial) Gradient(input []float64) []float64 {
	gradient := make([]float64, p.InputDim)
	if p == nil || p.Coefficients == nil || len(input) != p.InputDim {
		return gradient
	}

	mem := p.calculateStorage()
	if mem.Degree <= 2 {
		return mem.Quadratic.gradient(input)
	}
	powers := precomputePowers(input, mem.Degree)

	for _, polynomialTerm := range mem.Terms {
		for derivativeAxis, derivativeExponent := range polynomialTerm.Exponents {
			if derivativeExponent == 0 {
				continue
			}

			term := polynomialTerm.Value * float64(derivativeExponent)
			for axis, exponent := range polynomialTerm.Exponents {
				switch {
				case axis == derivativeAxis:
					term *= powers[axis][exponent-1]
				default:
					term *= powers[axis][exponent]
				}
			}
			gradient[derivativeAxis] += term
		}
	}
	return gradient
}

func (p *Polynomial) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	if res == nil || res.Len() != intersect.Len() {
		res = mat.NewVecDense(intersect.Len(), nil)
	} else {
		res.Zero()
	}

	local := p.localPoint(intersect)
	localGradient := p.localSurfaceGradient(local)
	for localAxis := 0; localAxis < len(localGradient); localAxis++ {
		if localAxis >= 3 {
			continue
		}
		for worldAxis := 0; worldAxis < res.Len() && worldAxis < 3; worldAxis++ {
			res.SetVec(worldAxis, res.AtVec(worldAxis)+localGradient[localAxis]*p.Transform[localAxis+1][worldAxis+1])
		}
	}
	return maths.Normalize(res)
}

func (p *Polynomial) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return p.BaseShape.BuildBoundingBox()
}

func (p *Polynomial) rayPolynomial(raySt, rayDir *mat.VecDense) ([]float64, error) {
	localSt := p.localPoint(raySt)
	localDir := p.localDirection(rayDir)
	if p.InputDim > len(localSt) {
		return nil, fmt.Errorf("%w: polynomial dimension mismatch", maths.ErrInvalidInput)
	}
	mem := p.calculateStorage()
	if mem.Degree <= 2 {
		return mem.Quadratic.rayPolynomial(localSt[:p.InputDim], localDir[:p.InputDim]), nil
	}
	ascending := make([]float64, mem.Degree+1)
	p.addTermsToRayPolynomial(ascending, localSt[:p.InputDim], localDir[:p.InputDim])
	return descendingPolynomial(ascending), nil
}

func (p *Polynomial) addTermsToRayPolynomial(ascending, starts, dirs []float64) {
	mem := p.calculateStorage()
	maxDegree := len(ascending) - 1
	for _, polynomialTerm := range mem.Terms {
		termPoly := []float64{polynomialTerm.Value}
		for axis, exponent := range polynomialTerm.Exponents {
			factor := linearPowerPolynomial(starts[axis], dirs[axis], exponent)
			termPoly = multiplyPolynomialsAscending(termPoly, factor, maxDegree)
		}
		for degree, coefficient := range termPoly {
			ascending[degree] += coefficient
		}
	}
}

func (p *Polynomial) localSurfaceGradient(local []float64) []float64 {
	return p.Gradient(local[:p.InputDim])
}

func (p *Polynomial) calculateStorage() PolynomialCalculateStorage {
	if p == nil {
		return PolynomialCalculateStorage{}
	}
	if p.Mem.Terms == nil && p.Coefficients != nil && p.Coefficients.NNZ() > 0 {
		p.RebuildCalculateStorage()
	}
	return p.Mem
}

func buildPolynomialCalculateStorage(inputDim int, coefficients *maths.SparseTensor[float64]) PolynomialCalculateStorage {
	mem := PolynomialCalculateStorage{}
	if coefficients == nil {
		return mem
	}

	coefficients.IterNonZero(func(index []int, value float64) {
		if len(index) != inputDim {
			return
		}

		exponents := append([]int(nil), index...)
		totalDegree := 0
		for _, exponent := range exponents {
			totalDegree += exponent
		}
		if totalDegree > mem.Degree {
			mem.Degree = totalDegree
		}
		mem.Terms = append(mem.Terms, polynomialTerm{
			Exponents: exponents,
			Value:     value,
		})
		if totalDegree <= 2 {
			mem.Quadratic.add(exponents, value)
		}
	})
	return mem
}

func (q *quadraticPolynomialKernel) add(exponents []int, value float64) {
	total := exponents[0] + exponents[1] + exponents[2]
	switch total {
	case 0:
		q.Constant += value
	case 1:
		for axis, exponent := range exponents {
			if exponent == 1 {
				q.Linear[axis] += value
				return
			}
		}
	case 2:
		for axis, exponent := range exponents {
			if exponent == 2 {
				q.Square[axis] += value
				return
			}
		}
		for first := 0; first < 3; first++ {
			for second := first + 1; second < 3; second++ {
				if exponents[first] == 1 && exponents[second] == 1 {
					q.Cross[first][second] += value
					return
				}
			}
		}
	}
}

func (q quadraticPolynomialKernel) evaluate(point []float64) float64 {
	value := q.Constant
	for axis := 0; axis < 3; axis++ {
		value += q.Linear[axis]*point[axis] + q.Square[axis]*point[axis]*point[axis]
		for other := axis + 1; other < 3; other++ {
			value += q.Cross[axis][other] * point[axis] * point[other]
		}
	}
	return value
}

func (q quadraticPolynomialKernel) gradient(point []float64) []float64 {
	gradient := make([]float64, 3)
	for axis := 0; axis < 3; axis++ {
		gradient[axis] = q.Linear[axis] + 2*q.Square[axis]*point[axis]
		for other := 0; other < 3; other++ {
			if other < axis {
				gradient[axis] += q.Cross[other][axis] * point[other]
			} else if other > axis {
				gradient[axis] += q.Cross[axis][other] * point[other]
			}
		}
	}
	return gradient
}

func (q quadraticPolynomialKernel) rayPolynomial(start, direction []float64) []float64 {
	a, b, c := 0.0, 0.0, q.Constant
	for axis := 0; axis < 3; axis++ {
		s, d := start[axis], direction[axis]
		c += q.Linear[axis]*s + q.Square[axis]*s*s
		b += q.Linear[axis]*d + 2*q.Square[axis]*s*d
		a += q.Square[axis] * d * d
		for other := axis + 1; other < 3; other++ {
			coefficient := q.Cross[axis][other]
			so, do := start[other], direction[other]
			c += coefficient * s * so
			b += coefficient * (s*do + so*d)
			a += coefficient * d * do
		}
	}
	return []float64{a, b, c}
}

func (p *Polynomial) localPoint(point *mat.VecDense) []float64 {
	local := make([]float64, minInt(point.Len(), 3))
	for localAxis := range local {
		local[localAxis] = p.Transform[localAxis+1][0]
		for worldAxis := 0; worldAxis < point.Len() && worldAxis < 3; worldAxis++ {
			local[localAxis] += p.Transform[localAxis+1][worldAxis+1] * point.AtVec(worldAxis)
		}
	}
	return local
}

func (p *Polynomial) localDirection(direction *mat.VecDense) []float64 {
	local := make([]float64, minInt(direction.Len(), 3))
	for localAxis := range local {
		for worldAxis := 0; worldAxis < direction.Len() && worldAxis < 3; worldAxis++ {
			local[localAxis] += p.Transform[localAxis+1][worldAxis+1] * direction.AtVec(worldAxis)
		}
	}
	return local
}

func precomputePowers(input []float64, degree int) [][]float64 {
	powers := make([][]float64, len(input))
	for axis, value := range input {
		powers[axis] = make([]float64, degree+1)
		powers[axis][0] = 1
		for exponent := 1; exponent <= degree; exponent++ {
			powers[axis][exponent] = powers[axis][exponent-1] * value
		}
	}
	return powers
}

func linearPowerPolynomial(start, direction float64, exponent int) []float64 {
	result := []float64{1}
	factor := []float64{start, direction}
	for i := 0; i < exponent; i++ {
		result = multiplyPolynomialsAscending(result, factor, exponent)
	}
	return result
}

func multiplyPolynomialsAscending(a, b []float64, maxDegree int) []float64 {
	result := make([]float64, minInt(len(a)+len(b)-1, maxDegree+1))
	for i, av := range a {
		for j, bv := range b {
			degree := i + j
			if degree >= len(result) {
				continue
			}
			result[degree] += av * bv
		}
	}
	return result
}

func descendingPolynomial(ascending []float64) []float64 {
	descending := make([]float64, len(ascending))
	for i, coefficient := range ascending {
		descending[len(ascending)-1-i] = coefficient
	}
	return descending
}

func identityTransform4() [4][4]float64 {
	transform := [4][4]float64{}
	for i := 0; i < 4; i++ {
		transform[i][i] = 1
	}
	return transform
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
