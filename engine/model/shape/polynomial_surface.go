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
	Mode         PolynomialSurfaceMode
	InputDim     int
	OutputDim    int
	Degree       int
	ExplicitAxis int
	Coefficients *maths.SparseTensor[float64]
	Center       []float64
	Scale        []float64
	Bounds       *Cuboid
}

func NewPolynomialSurface(
	mode PolynomialSurfaceMode,
	inputDim, outputDim, degree int,
	coefficients *maths.SparseTensor[float64],
) *PolynomialSurface {
	if outputDim <= 0 {
		outputDim = 1
	}
	return &PolynomialSurface{
		Mode:         mode,
		InputDim:     inputDim,
		OutputDim:    outputDim,
		Degree:       degree,
		ExplicitAxis: 2,
		Coefficients: coefficients,
		Center:       makeFilledFloat64(inputDim, 0),
		Scale:        makeFilledFloat64(inputDim, 1),
	}
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

	if p.Bounds != nil {
		var ok bool
		tMin, tMax, ok = p.Bounds.OverlapRange(raySt, rayDir, tMin, tMax)
		if !ok {
			return SurfaceInteraction{}, false
		}
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

	powers := precomputePowers(input, p.Degree)
	result := 0.0
	hasOutputAxis := p.coefficientsHaveOutputAxis()

	p.Coefficients.IterNonZero(func(index []int, value float64) {
		if hasOutputAxis {
			if index[0] != output {
				return
			}
			index = index[1:]
		}
		if len(index) != p.InputDim {
			return
		}

		term := value
		for axis, exponent := range index {
			if exponent > p.Degree {
				return
			}
			term *= powers[axis][exponent]
		}
		result += term
	})
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

	powers := precomputePowers(input, p.Degree)
	hasOutputAxis := p.coefficientsHaveOutputAxis()

	p.Coefficients.IterNonZero(func(index []int, value float64) {
		if hasOutputAxis {
			if index[0] != output {
				return
			}
			index = index[1:]
		}
		if len(index) != p.InputDim {
			return
		}

		for derivativeAxis, derivativeExponent := range index {
			if derivativeExponent == 0 {
				continue
			}

			term := value * float64(derivativeExponent)
			for axis, exponent := range index {
				switch {
				case axis == derivativeAxis:
					term *= powers[axis][exponent-1]
				default:
					term *= powers[axis][exponent]
				}
			}
			gradient[derivativeAxis] += term
		}
	})
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
	for i := 0; i < len(localGradient) && i < res.Len(); i++ {
		scale := p.scaleAt(i)
		res.SetVec(i, localGradient[i]/scale)
	}
	return maths.Normalize(res)
}

func (p *PolynomialSurface) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	if p != nil && p.Bounds != nil {
		return p.Bounds.BuildBoundingBox()
	}
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
	ascending := make([]float64, p.Degree+1)
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

	ascending := make([]float64, p.Degree+1)
	p.addTermsToRayPolynomial(ascending, inputSt, inputDir, 0)
	ascending[0] -= localSt[p.ExplicitAxis]
	if len(ascending) < 2 {
		ascending = append(ascending, 0)
	}
	ascending[1] -= localDir[p.ExplicitAxis]
	return descendingPolynomial(ascending), nil
}

func (p *PolynomialSurface) addTermsToRayPolynomial(ascending, starts, dirs []float64, output int) {
	hasOutputAxis := p.coefficientsHaveOutputAxis()
	p.Coefficients.IterNonZero(func(index []int, value float64) {
		if hasOutputAxis {
			if index[0] != output {
				return
			}
			index = index[1:]
		}
		if len(index) != p.InputDim {
			return
		}

		termPoly := []float64{value}
		for axis, exponent := range index {
			factor := linearPowerPolynomial(starts[axis], dirs[axis], exponent)
			termPoly = multiplyPolynomialsAscending(termPoly, factor, p.Degree)
		}
		for degree, coefficient := range termPoly {
			ascending[degree] += coefficient
		}
	})
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

func (p *PolynomialSurface) coefficientsHaveOutputAxis() bool {
	if p == nil || p.Coefficients == nil {
		return false
	}
	return p.OutputDim > 1 && len(p.Coefficients.Shape) == p.InputDim+1
}

func (p *PolynomialSurface) localPoint(point *mat.VecDense) []float64 {
	local := make([]float64, point.Len())
	for i := 0; i < point.Len(); i++ {
		local[i] = (point.AtVec(i) - p.centerAt(i)) / p.scaleAt(i)
	}
	return local
}

func (p *PolynomialSurface) localDirection(direction *mat.VecDense) []float64 {
	local := make([]float64, direction.Len())
	for i := 0; i < direction.Len(); i++ {
		local[i] = direction.AtVec(i) / p.scaleAt(i)
	}
	return local
}

func (p *PolynomialSurface) centerAt(axis int) float64 {
	if p != nil && axis >= 0 && axis < len(p.Center) {
		return p.Center[axis]
	}
	return 0
}

func (p *PolynomialSurface) scaleAt(axis int) float64 {
	if p != nil && axis >= 0 && axis < len(p.Scale) && p.Scale[axis] != 0 {
		return p.Scale[axis]
	}
	return 1
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

func makeFilledFloat64(length int, value float64) []float64 {
	result := make([]float64, length)
	for i := range result {
		result[i] = value
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var ErrPolynomialSurfaceDimension = fmt.Errorf("%w: polynomial surface dimension mismatch", maths.ErrInvalidInput)
