package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

const (
	defaultImplicitMaxSteps    = 2048
	defaultImplicitRootTol     = 1e-6
	defaultImplicitValueTol    = 1e-7
	defaultImplicitGradientEps = 1e-5
)

type ImplicitEquation struct {
	BaseShape
	Function func(*mat.VecDense) float64 // implicit scalar field F(p), whose surface is F(p)=0.
	Gradient func(point, res *mat.VecDense) *mat.VecDense
	Range    [2]*mat.VecDense

	Step        float64
	MaxSteps    int
	RootTol     float64
	ValueTol    float64
	GradientEps float64
}

func NewImplicitEquation(Function func(*mat.VecDense) float64, Range [2]*mat.VecDense) *ImplicitEquation { // Index order: 1, x, y, z
	return &ImplicitEquation{
		Function:    Function,
		Range:       Range,
		MaxSteps:    defaultImplicitMaxSteps,
		RootTol:     defaultImplicitRootTol,
		ValueTol:    defaultImplicitValueTol,
		GradientEps: defaultImplicitGradientEps,
	}
}

func NewImplicitEquationWithGradient(
	Function func(*mat.VecDense) float64,
	Gradient func(point, res *mat.VecDense) *mat.VecDense,
	Range [2]*mat.VecDense,
) *ImplicitEquation {
	equation := NewImplicitEquation(Function, Range)
	equation.Gradient = Gradient
	return equation
}

func (f *ImplicitEquation) Name() string {
	return "Implicit Equation"
}

func (f *ImplicitEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := f.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (f *ImplicitEquation) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	if f == nil || f.Function == nil || raySt == nil || rayDir == nil {
		return SurfaceInteraction{}, false
	}
	if raySt.Len() != rayDir.Len() || raySt.Len() < utils.Dimension || tMax < tMin {
		return SurfaceInteraction{}, false
	}

	if f.hasValidRange() {
		var ok bool
		tMin, tMax, ok = NewCuboid(f.Range[0], f.Range[1]).OverlapRange(raySt, rayDir, tMin, tMax)
		if !ok {
			return SurfaceInteraction{}, false
		}
	}

	scanEnd := tMax
	step := f.searchStep(tMin, scanEnd)
	if step <= 0 {
		return SurfaceInteraction{}, false
	}
	if !isFinite(scanEnd) || scanEnd > tMin+step*float64(f.maxSteps()) {
		scanEnd = tMin + step*float64(f.maxSteps())
	}
	if scanEnd < tMin {
		return SurfaceInteraction{}, false
	}

	prevT := tMin
	prevValue := f.evaluateRay(raySt, rayDir, prevT)
	if isFinite(prevValue) && math.Abs(prevValue) <= f.valueTol() {
		return f.interactionAt(raySt, rayDir, prevT), true
	}

	steps := int(math.Ceil((scanEnd - tMin) / step))
	if steps < 1 {
		steps = 1
	}
	if steps > f.maxSteps() {
		steps = f.maxSteps()
	}

	for i := 1; i <= steps; i++ {
		currT := tMin + float64(i)*step
		if currT > scanEnd || i == steps {
			currT = scanEnd
		}
		currValue := f.evaluateRay(raySt, rayDir, currT)
		if !isFinite(currValue) {
			continue
		}

		if math.Abs(currValue) <= f.valueTol() {
			return f.interactionAt(raySt, rayDir, currT), true
		}

		if isFinite(prevValue) && hasSignChange(prevValue, currValue) {
			root, ok := f.findRootBisection(raySt, rayDir, prevT, currT, prevValue, currValue)
			if ok && distanceInRange(root, tMin, tMax) {
				return f.interactionAt(raySt, rayDir, root), true
			}
		}

		prevT = currT
		prevValue = currValue
	}

	return SurfaceInteraction{}, false
}

func (f *ImplicitEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	if intersect == nil {
		return res
	}
	if res == nil || res.Len() != intersect.Len() {
		res = mat.NewVecDense(intersect.Len(), nil)
	} else {
		res.Zero()
	}

	if f != nil && f.Gradient != nil {
		gradient := f.Gradient(intersect, res)
		if gradient != nil {
			return maths.Normalize(gradient)
		}
	}

	f.numericalGradient(intersect, res)
	return maths.Normalize(res)
}

func (f *ImplicitEquation) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	if f == nil {
		return (&BaseShape{}).BuildBoundingBox()
	}
	if !f.hasValidRange() {
		return f.BaseShape.BuildBoundingBox()
	}
	return f.Range[0], f.Range[1]
}

func (f *ImplicitEquation) evaluateRay(raySt, rayDir *mat.VecDense, t float64) float64 {
	point := pointAt(raySt, rayDir, t)
	return f.Function(point)
}

func (f *ImplicitEquation) interactionAt(raySt, rayDir *mat.VecDense, distance float64) SurfaceInteraction {
	point := pointAt(raySt, rayDir, distance)
	normal := f.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteractionAt(point, distance, normal)
}

func (f *ImplicitEquation) findRootBisection(
	raySt, rayDir *mat.VecDense,
	left, right, fLeft, fRight float64,
) (float64, bool) {
	if math.Abs(fLeft) <= f.valueTol() {
		return left, true
	}
	if math.Abs(fRight) <= f.valueTol() {
		return right, true
	}
	if !hasSignChange(fLeft, fRight) {
		return 0, false
	}

	for i := 0; i < f.maxSteps(); i++ {
		mid := 0.5 * (left + right)
		fMid := f.evaluateRay(raySt, rayDir, mid)
		if !isFinite(fMid) {
			return 0, false
		}
		if math.Abs(fMid) <= f.valueTol() || math.Abs(right-left) <= f.rootTol() {
			return mid, true
		}

		if hasSignChange(fLeft, fMid) {
			right = mid
			fRight = fMid
		} else {
			left = mid
			fLeft = fMid
		}
	}

	return 0.5 * (left + right), true
}

func (f *ImplicitEquation) numericalGradient(point, res *mat.VecDense) {
	if f == nil || f.Function == nil || point == nil || res == nil {
		return
	}

	eps := f.gradientEps()
	work := mat.VecDenseCopyOf(point)
	dim := minInt(point.Len(), utils.Dimension)
	for axis := 0; axis < dim; axis++ {
		original := point.AtVec(axis)

		work.SetVec(axis, original+eps)
		plus := f.Function(work)

		work.SetVec(axis, original-eps)
		minus := f.Function(work)

		work.SetVec(axis, original)
		if isFinite(plus) && isFinite(minus) {
			res.SetVec(axis, (plus-minus)/(2*eps))
		}
	}
}

func (f *ImplicitEquation) hasValidRange() bool {
	if f == nil || f.Range[0] == nil || f.Range[1] == nil {
		return false
	}
	if f.Range[0].Len() < utils.Dimension || f.Range[1].Len() < utils.Dimension {
		return false
	}
	for i := 0; i < utils.Dimension; i++ {
		if f.Range[0].AtVec(i) >= f.Range[1].AtVec(i) {
			return false
		}
	}
	return true
}

func (f *ImplicitEquation) searchStep(tMin, tMax float64) float64 {
	if f != nil && f.Step > 0 {
		return f.Step
	}
	if f != nil && f.hasValidRange() {
		diagonalSquared := 0.0
		for i := 0; i < utils.Dimension; i++ {
			side := f.Range[1].AtVec(i) - f.Range[0].AtVec(i)
			diagonalSquared += side * side
		}
		if diagonalSquared > 0 {
			samples := f.maxSteps() / 4
			if samples < 1 {
				samples = 1
			}
			return math.Sqrt(diagonalSquared) / float64(samples)
		}
	}
	if isFinite(tMax) && tMax > tMin {
		return (tMax - tMin) / float64(f.maxSteps())
	}
	return 0.02
}

func (f *ImplicitEquation) maxSteps() int {
	if f != nil && f.MaxSteps > 0 {
		return f.MaxSteps
	}
	return defaultImplicitMaxSteps
}

func (f *ImplicitEquation) rootTol() float64 {
	if f != nil && f.RootTol > 0 {
		return f.RootTol
	}
	return defaultImplicitRootTol
}

func (f *ImplicitEquation) valueTol() float64 {
	if f != nil && f.ValueTol > 0 {
		return f.ValueTol
	}
	return defaultImplicitValueTol
}

func (f *ImplicitEquation) gradientEps() float64 {
	if f != nil && f.GradientEps > 0 {
		return f.GradientEps
	}
	return defaultImplicitGradientEps
}

func hasSignChange(a, b float64) bool {
	return (a < 0 && b > 0) || (a > 0 && b < 0)
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
