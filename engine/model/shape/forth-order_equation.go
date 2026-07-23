package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type FourOrderEquation struct {
	BaseShape
	A   *maths.Tensor[float64] `json:"a"`
	Mem FourOrderEquationCalculateStorage
}

type FourOrderEquationCalculateStorage struct {
	Terms []fourOrderEquationTerm
}

type fourOrderEquationTerm struct {
	Powers [3]int
	Value  float64
}

func NewFourOrderEquation(A []float64) *FourOrderEquation { // Index order: 1, x, y, z
	ATensor := maths.NewTensorFromSlice(A, []int{4, 4, 4, 4})

	equation := &FourOrderEquation{
		A: ATensor,
	}
	equation.RebuildCalculateStorage()
	return equation
}

func (p *FourOrderEquation) RebuildCalculateStorage() {
	if p == nil {
		return
	}
	p.Mem = buildFourOrderEquationCalculateStorage(p.A)
}

func (p *FourOrderEquation) Name() string {
	return "Four-Order Equation"
}

func (p *FourOrderEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := p.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (p *FourOrderEquation) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	var (
		coeffs [5]float64       // Coefficients from the fourth-degree term to the constant term.
		stx    = raySt.AtVec(0) // Get ray origin and direction components.
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

	res := math.MaxFloat64 // Find the smallest positive real root.
	for _, root := range roots {
		if distanceInRange(root, tMin, tMax) && root < res {
			res = root
		}
	}
	if res == math.MaxFloat64 {
		return SurfaceInteraction{}, false
	}

	point := pointAt(raySt, rayDir, res)
	normal := p.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	return newSurfaceInteractionAt(point, res, normal), true
}

func (p *FourOrderEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	var (
		x       = intersect.AtVec(0) // Get intersection coordinates.
		y       = intersect.AtVec(1)
		z       = intersect.AtVec(2)
		factors = [4]float64{1, x, y, z} // factors[0]=1, factors[1]=x, factors[2]=y, factors[3]=z
		grad    = [3]float64{0, 0, 0}    // dx, dy, dz	// Initialize the gradient vector.
	)

	for _, term := range p.calculateStorage().Terms {
		xPower, yPower, zPower := term.Powers[0], term.Powers[1], term.Powers[2]
		if xPower > 0 {
			grad[0] += term.Value * float64(xPower) *
				smallPower(factors[1], xPower-1) *
				smallPower(factors[2], yPower) *
				smallPower(factors[3], zPower)
		}
		if yPower > 0 {
			grad[1] += term.Value * float64(yPower) *
				smallPower(factors[1], xPower) *
				smallPower(factors[2], yPower-1) *
				smallPower(factors[3], zPower)
		}
		if zPower > 0 {
			grad[2] += term.Value * float64(zPower) *
				smallPower(factors[1], xPower) *
				smallPower(factors[2], yPower) *
				smallPower(factors[3], zPower-1)
		}
	}

	res.SetVec(0, grad[0])
	res.SetVec(1, grad[1])
	res.SetVec(2, grad[2])
	return maths.Normalize(res)
}

func (p *FourOrderEquation) calculateStorage() FourOrderEquationCalculateStorage {
	if p == nil {
		return FourOrderEquationCalculateStorage{}
	}
	if p.Mem.Terms == nil && p.A != nil {
		p.RebuildCalculateStorage()
	}
	return p.Mem
}

func buildFourOrderEquationCalculateStorage(a *maths.Tensor[float64]) FourOrderEquationCalculateStorage {
	mem := FourOrderEquationCalculateStorage{
		Terms: []fourOrderEquationTerm{},
	}
	if a == nil {
		return mem
	}
	terms := map[[3]int]float64{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				for l := 0; l < 4; l++ {
					value := a.Get(i, j, k, l)
					if value == 0 {
						continue
					}
					powers := fourOrderEquationPowers([4]int{i, j, k, l})
					terms[powers] += value
				}
			}
		}
	}
	for powers, value := range terms {
		if value == 0 {
			continue
		}
		mem.Terms = append(mem.Terms, fourOrderEquationTerm{
			Powers: powers,
			Value:  value,
		})
	}
	return mem
}

func fourOrderEquationPowers(indices [4]int) [3]int {
	powers := [3]int{}
	for _, index := range indices {
		if index >= 1 && index <= 3 {
			powers[index-1]++
		}
	}
	return powers
}

func linearRayPowerTable(start, direction float64) [5][5]float64 {
	powers := [5][5]float64{}
	powers[0][0] = 1
	for exponent := 1; exponent <= 4; exponent++ {
		for degree := 0; degree <= exponent-1; degree++ {
			coefficient := powers[exponent-1][degree]
			powers[exponent][degree] += coefficient * start
			powers[exponent][degree+1] += coefficient * direction
		}
	}
	return powers
}

func smallPower(value float64, exponent int) float64 {
	switch exponent {
	case 0:
		return 1
	case 1:
		return value
	case 2:
		return value * value
	case 3:
		return value * value * value
	case 4:
		squared := value * value
		return squared * squared
	default:
		return math.Pow(value, float64(exponent))
	}
}
