package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"gonum.org/v1/gonum/mat"
	"math"
)

type CubicEquation struct {
	BaseShape
	A   *maths.Tensor[float64] `json:"a"`
	Mem CubicEquationCalculateStorage
}

type CubicEquationCalculateStorage struct {
	Terms []cubicEquationTerm
}

type cubicEquationTerm struct {
	Powers [3]int
	Value  float64
}

func NewCubicEquation(A []float64) *CubicEquation {
	equation := &CubicEquation{
		A: maths.NewTensorFromSlice(A, []int{4, 4, 4}),
	}
	equation.RebuildCalculateStorage()
	return equation
}

func (p *CubicEquation) RebuildCalculateStorage() {
	if p == nil {
		return
	}
	p.Mem = buildCubicEquationCalculateStorage(p.A)
}

func (p *CubicEquation) Name() string {
	return "Cubic Equation"
}

func (p *CubicEquation) IntersectAffine(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if !options.valid() {
		return SurfaceInteraction{}, false
	}
	var (
		coeffs [4]float64
		stx    = raySt.AtVec(0)
		sty    = raySt.AtVec(1)
		stz    = raySt.AtVec(2)
		dirx   = rayDir.AtVec(0)
		diry   = rayDir.AtVec(1)
		dirz   = rayDir.AtVec(2)
	)

	xPowers := linearRayPowerTable(stx, dirx)
	yPowers := linearRayPowerTable(sty, diry)
	zPowers := linearRayPowerTable(stz, dirz)
	for _, term := range p.calculateStorage().Terms {
		xPower, yPower, zPower := term.Powers[0], term.Powers[1], term.Powers[2]
		for xDegree := 0; xDegree <= xPower; xDegree++ {
			xCoefficient := xPowers[xPower][xDegree]
			if xCoefficient == 0 {
				continue
			}
			for yDegree := 0; yDegree <= yPower; yDegree++ {
				xyCoefficient := xCoefficient * yPowers[yPower][yDegree]
				if xyCoefficient == 0 {
					continue
				}
				for zDegree := 0; zDegree <= zPower; zDegree++ {
					degree := xDegree + yDegree + zDegree
					coeffs[len(coeffs)-1-degree] += term.Value * xyCoefficient * zPowers[zPower][zDegree]
				}
			}
		}
	}

	roots, err := maths.SolvePolynomialReal(coeffs[:])
	if err != nil {
		return SurfaceInteraction{}, false
	}

	res := math.MaxFloat64
	for _, root := range roots {
		if distanceInRange(root, options.Range.Min, options.Range.Max) && root < res {
			res = root
		}
	}
	if res == math.MaxFloat64 {
		return SurfaceInteraction{}, false
	}

	point := affinePointAt(raySt, rayDir, res)
	normal := p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteractionAt(point, res, normal), true
}

func (p *CubicEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	var (
		x    = intersect.AtVec(0)
		y    = intersect.AtVec(1)
		z    = intersect.AtVec(2)
		grad = [3]float64{}
	)

	for _, term := range p.calculateStorage().Terms {
		xPower, yPower, zPower := term.Powers[0], term.Powers[1], term.Powers[2]
		if xPower > 0 {
			grad[0] += term.Value * float64(xPower) *
				smallPower(x, xPower-1) *
				smallPower(y, yPower) *
				smallPower(z, zPower)
		}
		if yPower > 0 {
			grad[1] += term.Value * float64(yPower) *
				smallPower(x, xPower) *
				smallPower(y, yPower-1) *
				smallPower(z, zPower)
		}
		if zPower > 0 {
			grad[2] += term.Value * float64(zPower) *
				smallPower(x, xPower) *
				smallPower(y, yPower) *
				smallPower(z, zPower-1)
		}
	}

	res.SetVec(0, grad[0])
	res.SetVec(1, grad[1])
	res.SetVec(2, grad[2])
	return maths.Normalize(res)
}

func (p *CubicEquation) calculateStorage() CubicEquationCalculateStorage {
	if p == nil {
		return CubicEquationCalculateStorage{}
	}
	if p.Mem.Terms == nil && p.A != nil {
		p.RebuildCalculateStorage()
	}
	return p.Mem
}

func buildCubicEquationCalculateStorage(a *maths.Tensor[float64]) CubicEquationCalculateStorage {
	mem := CubicEquationCalculateStorage{
		Terms: []cubicEquationTerm{},
	}
	if a == nil {
		return mem
	}
	terms := map[[3]int]float64{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				value := a.Get(i, j, k)
				if value == 0 {
					continue
				}
				powers := cubicEquationPowers([3]int{i, j, k})
				terms[powers] += value
			}
		}
	}
	for powers, value := range terms {
		if value == 0 {
			continue
		}
		mem.Terms = append(mem.Terms, cubicEquationTerm{
			Powers: powers,
			Value:  value,
		})
	}
	return mem
}

func cubicEquationPowers(indices [3]int) [3]int {
	powers := [3]int{}
	for _, index := range indices {
		if index >= 1 && index <= 3 {
			powers[index-1]++
		}
	}
	return powers
}
