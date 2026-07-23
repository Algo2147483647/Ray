package shape

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type PolynomialSurfaceMode string

const (
	PolynomialSurfaceImplicit PolynomialSurfaceMode = "implicit"
	PolynomialSurfaceExplicit PolynomialSurfaceMode = "explicit"
)

type PolynomialSurface struct {
	BaseShape
	Mode         PolynomialSurfaceMode        // Surface equation mode: implicit F(x)=0 or explicit z=P(x).
	InputDim     int                          // Number of local input coordinates used by the polynomial.
	ExplicitAxis int                          // Local axis solved by explicit mode, usually z.
	Coefficients *maths.SparseTensor[float64] // Sparse polynomial coefficients indexed by exponents.
	Transform    [4][4]float64                // World-to-local homogeneous transform matrix.
	Mem          PolynomialSurfaceCalculateStorage
}

type PolynomialSurfaceCalculateStorage struct {
	Degree        int
	HasOutputAxis bool
	Terms         []polynomialSurfaceTerm
}

type polynomialSurfaceTerm struct {
	Output    int
	Exponents []int
	Value     float64
}

func NewPolynomialSurface(
	mode PolynomialSurfaceMode,
	inputDim int,
	coefficients *maths.SparseTensor[float64],
) *PolynomialSurface {
	surface := &PolynomialSurface{
		Mode:         mode,
		InputDim:     inputDim,
		ExplicitAxis: 2,
		Coefficients: coefficients,
		Transform:    identityTransform4(),
	}
	surface.RebuildCalculateStorage()
	return surface
}

func (p *PolynomialSurface) RebuildCalculateStorage() {
	if p == nil {
		return
	}
	p.Mem = buildPolynomialSurfaceCalculateStorage(p.InputDim, p.Coefficients)
}

func (p *PolynomialSurface) Name() string {
	return "Polynomial Surface"
}

func (p *PolynomialSurface) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := p.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (p *PolynomialSurface) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
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
		if distanceInRange(root, tMin, tMax) && root < bestT {
			bestT = root
		}
	}
	if bestT == math.MaxFloat64 {
		return SurfaceInteraction{}, false
	}

	point := pointAt(raySt, rayDir, bestT)
	normal := p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteractionAt(point, bestT, normal), true
}

func (p *PolynomialSurface) Evaluate(input []float64) float64 {
	return p.EvaluateOutput(input, 0)
}

func (p *PolynomialSurface) EvaluateOutput(input []float64, output int) float64 {
	if p == nil || p.Coefficients == nil || len(input) != p.InputDim {
		return math.NaN()
	}

	mem := p.calculateStorage()
	powers := precomputePowers(input, mem.Degree)
	result := 0.0

	for _, polynomialTerm := range mem.Terms {
		if mem.HasOutputAxis && polynomialTerm.Output != output {
			continue
		}
		term := polynomialTerm.Value
		for axis, exponent := range polynomialTerm.Exponents {
			term *= powers[axis][exponent]
		}
		result += term
	}
	return result
}

func (p *PolynomialSurface) Gradient(input []float64) []float64 {
	return p.GradientOutput(input, 0)
}

func (p *PolynomialSurface) GradientOutput(input []float64, output int) []float64 {
	gradient := make([]float64, p.InputDim)
	if p == nil || p.Coefficients == nil || len(input) != p.InputDim {
		return gradient
	}

	mem := p.calculateStorage()
	powers := precomputePowers(input, mem.Degree)

	for _, polynomialTerm := range mem.Terms {
		if mem.HasOutputAxis && polynomialTerm.Output != output {
			continue
		}
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

func (p *PolynomialSurface) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
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

func (p *PolynomialSurface) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	return p.BaseShape.BuildBoundingBox()
}

func (p *PolynomialSurface) rayPolynomial(raySt, rayDir *mat.VecDense) ([]float64, error) {
	localSt := p.localPoint(raySt)
	localDir := p.localDirection(rayDir)

	switch p.Mode {
	case PolynomialSurfaceExplicit:
		return p.explicitRayPolynomial(localSt, localDir)
	case PolynomialSurfaceImplicit, "":
		return p.implicitRayPolynomial(localSt, localDir)
	default:
		return nil, fmt.Errorf("unsupported polynomial surface mode %q", p.Mode)
	}
}

func (p *PolynomialSurface) implicitRayPolynomial(localSt, localDir []float64) ([]float64, error) {
	if p.InputDim > len(localSt) {
		return nil, ErrPolynomialSurfaceDimension
	}
	ascending := make([]float64, p.calculateStorage().Degree+1)
	p.addTermsToRayPolynomial(ascending, localSt[:p.InputDim], localDir[:p.InputDim], 0)
	return descendingPolynomial(ascending), nil
}

func (p *PolynomialSurface) explicitRayPolynomial(localSt, localDir []float64) ([]float64, error) {
	axes := p.explicitInputAxes()
	if len(axes) != p.InputDim || p.ExplicitAxis < 0 || p.ExplicitAxis >= len(localSt) {
		return nil, ErrPolynomialSurfaceDimension
	}

	inputSt := make([]float64, p.InputDim)
	inputDir := make([]float64, p.InputDim)
	for i, axis := range axes {
		inputSt[i] = localSt[axis]
		inputDir[i] = localDir[axis]
	}

	ascending := make([]float64, p.calculateStorage().Degree+1)
	p.addTermsToRayPolynomial(ascending, inputSt, inputDir, 0)
	ascending[0] -= localSt[p.ExplicitAxis]
	if len(ascending) < 2 {
		ascending = append(ascending, 0)
	}
	ascending[1] -= localDir[p.ExplicitAxis]
	return descendingPolynomial(ascending), nil
}

func (p *PolynomialSurface) addTermsToRayPolynomial(ascending, starts, dirs []float64, output int) {
	mem := p.calculateStorage()
	maxDegree := len(ascending) - 1
	for _, polynomialTerm := range mem.Terms {
		if mem.HasOutputAxis && polynomialTerm.Output != output {
			continue
		}
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

func (p *PolynomialSurface) localSurfaceGradient(local []float64) []float64 {
	switch p.Mode {
	case PolynomialSurfaceExplicit:
		result := make([]float64, len(local))
		axes := p.explicitInputAxes()
		input := make([]float64, len(axes))
		for i, axis := range axes {
			input[i] = local[axis]
		}
		gradient := p.Gradient(input)
		for i, axis := range axes {
			result[axis] = gradient[i]
		}
		if p.ExplicitAxis >= 0 && p.ExplicitAxis < len(result) {
			result[p.ExplicitAxis] = -1
		}
		return result
	default:
		return p.Gradient(local[:p.InputDim])
	}
}

func (p *PolynomialSurface) explicitInputAxes() []int {
	axes := make([]int, 0, p.InputDim)
	for axis := 0; len(axes) < p.InputDim && axis < 3; axis++ {
		if axis != p.ExplicitAxis {
			axes = append(axes, axis)
		}
	}
	return axes
}

func (p *PolynomialSurface) calculateStorage() PolynomialSurfaceCalculateStorage {
	if p == nil {
		return PolynomialSurfaceCalculateStorage{}
	}
	if p.Mem.Terms == nil && p.Coefficients != nil && p.Coefficients.NNZ() > 0 {
		p.RebuildCalculateStorage()
	}
	return p.Mem
}

func coefficientsHavePolynomialOutputAxis(inputDim int, coefficients *maths.SparseTensor[float64]) bool {
	if coefficients == nil {
		return false
	}
	return len(coefficients.Shape) == inputDim+1
}

func buildPolynomialSurfaceCalculateStorage(inputDim int, coefficients *maths.SparseTensor[float64]) PolynomialSurfaceCalculateStorage {
	mem := PolynomialSurfaceCalculateStorage{
		HasOutputAxis: coefficientsHavePolynomialOutputAxis(inputDim, coefficients),
	}
	if coefficients == nil {
		return mem
	}

	coefficients.IterNonZero(func(index []int, value float64) {
		output := 0
		if mem.HasOutputAxis {
			output = index[0]
			index = index[1:]
		}
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
		mem.Terms = append(mem.Terms, polynomialSurfaceTerm{
			Output:    output,
			Exponents: exponents,
			Value:     value,
		})
	})
	return mem
}

func (p *PolynomialSurface) coefficientsHaveOutputAxis() bool {
	if p == nil || p.Coefficients == nil {
		return false
	}
	return p.calculateStorage().HasOutputAxis
}

func (p *PolynomialSurface) polynomialDegree() int {
	if p == nil || p.Coefficients == nil {
		return 0
	}
	return p.calculateStorage().Degree
}

func (p *PolynomialSurface) localPoint(point *mat.VecDense) []float64 {
	local := make([]float64, minInt(point.Len(), 3))
	for localAxis := range local {
		local[localAxis] = p.Transform[localAxis+1][0]
		for worldAxis := 0; worldAxis < point.Len() && worldAxis < 3; worldAxis++ {
			local[localAxis] += p.Transform[localAxis+1][worldAxis+1] * point.AtVec(worldAxis)
		}
	}
	return local
}

func (p *PolynomialSurface) localDirection(direction *mat.VecDense) []float64 {
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

var ErrPolynomialSurfaceDimension = fmt.Errorf("%w: polynomial surface dimension mismatch", maths.ErrInvalidInput)
