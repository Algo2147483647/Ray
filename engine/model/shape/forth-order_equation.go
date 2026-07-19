package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type FourOrderEquation struct {
	BaseShape
	A      *maths.Tensor[float64] `json:"a"`
	Center []float64
	Scale  []float64
	Basis  [][]float64
}

func NewFourOrderEquation(A []float64) *FourOrderEquation { // Index order: 1, x, y, z
	ATensor := maths.NewTensorFromSlice(A, []int{4, 4, 4, 4})

	return &FourOrderEquation{
		A:      ATensor,
		Center: []float64{0, 0, 0},
		Scale:  []float64{1, 1, 1},
		Basis:  identityFourOrderBasis(),
	}
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
	localSt := p.localPoint(raySt)
	localDir := p.localDirection(rayDir)
	var (
		coeffs = []float64{0, 0, 0, 0, 0} // Coefficients from the fourth-degree term to the constant term.
		stx    = localSt[0]               // Get local ray origin and direction components.
		sty    = localSt[1]
		stz    = localSt[2]
		dirx   = localDir[0]
		diry   = localDir[1]
		dirz   = localDir[2]
	)

	for i := 0; i < 4; i++ { // Iterate over tensor A indices (0 to 3).
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				for l := 0; l < 4; l++ {
					c := p.A.Get(i, j, k, l)
					if c == 0 {
						continue
					}

					poly := [5]float64{1, 0, 0, 0, 0} // Initialize the current term polynomial coefficients (constant term is 1).
					indices := [4]int{i, j, k, l}     // Process the factor for each index.
					for _, idx := range indices {
						var polyFactor [2]float64
						switch idx {
						case 0:
							polyFactor = [2]float64{1, 0} // Constant factor 1
						case 1:
							polyFactor = [2]float64{stx, dirx} // x factor
						case 2:
							polyFactor = [2]float64{sty, diry} // y factor
						case 3:
							polyFactor = [2]float64{stz, dirz} // z factor
						default:
							polyFactor = [2]float64{0, 0} // Invalid index, default to 0.
						}

						newPoly := [5]float64{} // Multiply the current polynomial by the factor polynomial.
						for d1, coef1 := range poly {
							for d2, coef2 := range polyFactor {
								if d1+d2 < 5 {
									newPoly[d1+d2] += coef1 * coef2
								}
							}
						}
						poly = newPoly
					}

					for degree, coef := range poly { // Multiply the current term polynomial coefficients by c and accumulate them.
						coeffs[len(coeffs)-1-degree] += c * coef
					}
				}
			}
		}
	}

	roots, err := maths.SolvePolynomialReal(coeffs)
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
	local := p.localPoint(intersect)
	var (
		x       = local[0] // Get local intersection coordinates.
		y       = local[1]
		z       = local[2]
		factors = [4]float64{1, x, y, z} // factors[0]=1, factors[1]=x, factors[2]=y, factors[3]=z
		grad    = [3]float64{0, 0, 0}    // dx, dy, dz	// Initialize the gradient vector.
	)

	// Iterate over tensor A indices (0 to 3).
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				for l := 0; l < 4; l++ {
					c := p.A.Get(i, j, k, l)
					if c == 0 {
						continue
					}

					// Compute the partial derivative contribution for x.
					dx := 0.0
					if i == 1 {
						dx += factors[j] * factors[k] * factors[l]
					}
					if j == 1 {
						dx += factors[i] * factors[k] * factors[l]
					}
					if k == 1 {
						dx += factors[i] * factors[j] * factors[l]
					}
					if l == 1 {
						dx += factors[i] * factors[j] * factors[k]
					}
					grad[0] += c * dx

					// Compute the partial derivative contribution for y.
					dy := 0.0
					if i == 2 {
						dy += factors[j] * factors[k] * factors[l]
					}
					if j == 2 {
						dy += factors[i] * factors[k] * factors[l]
					}
					if k == 2 {
						dy += factors[i] * factors[j] * factors[l]
					}
					if l == 2 {
						dy += factors[i] * factors[j] * factors[k]
					}
					grad[1] += c * dy

					// Compute the partial derivative contribution for z.
					dz := 0.0
					if i == 3 {
						dz += factors[j] * factors[k] * factors[l]
					}
					if j == 3 {
						dz += factors[i] * factors[k] * factors[l]
					}
					if k == 3 {
						dz += factors[i] * factors[j] * factors[l]
					}
					if l == 3 {
						dz += factors[i] * factors[j] * factors[k]
					}
					grad[2] += c * dz
				}
			}
		}
	}

	if res == nil || res.Len() != intersect.Len() {
		res = mat.NewVecDense(intersect.Len(), nil)
	} else {
		res.Zero()
	}
	for localAxis, gradient := range grad {
		scale := p.scaleAt(localAxis)
		for worldAxis := 0; worldAxis < res.Len(); worldAxis++ {
			res.SetVec(worldAxis, res.AtVec(worldAxis)+gradient*p.basisAt(localAxis, worldAxis)/scale)
		}
	}
	return maths.Normalize(res)
}

func (p *FourOrderEquation) localPoint(point *mat.VecDense) []float64 {
	local := make([]float64, 3)
	for localAxis := 0; localAxis < 3; localAxis++ {
		for worldAxis := 0; worldAxis < 3 && worldAxis < point.Len(); worldAxis++ {
			local[localAxis] += (point.AtVec(worldAxis) - p.centerAt(worldAxis)) * p.basisAt(localAxis, worldAxis)
		}
		local[localAxis] /= p.scaleAt(localAxis)
	}
	return local
}

func (p *FourOrderEquation) localDirection(direction *mat.VecDense) []float64 {
	local := make([]float64, 3)
	for localAxis := 0; localAxis < 3; localAxis++ {
		for worldAxis := 0; worldAxis < 3 && worldAxis < direction.Len(); worldAxis++ {
			local[localAxis] += direction.AtVec(worldAxis) * p.basisAt(localAxis, worldAxis)
		}
		local[localAxis] /= p.scaleAt(localAxis)
	}
	return local
}

func (p *FourOrderEquation) centerAt(axis int) float64 {
	if p != nil && axis >= 0 && axis < len(p.Center) {
		return p.Center[axis]
	}
	return 0
}

func (p *FourOrderEquation) scaleAt(axis int) float64 {
	if p != nil && axis >= 0 && axis < len(p.Scale) && p.Scale[axis] != 0 {
		return p.Scale[axis]
	}
	return 1
}

func (p *FourOrderEquation) basisAt(localAxis, worldAxis int) float64 {
	if p != nil && localAxis >= 0 && localAxis < len(p.Basis) && worldAxis >= 0 && worldAxis < len(p.Basis[localAxis]) {
		return p.Basis[localAxis][worldAxis]
	}
	if localAxis == worldAxis {
		return 1
	}
	return 0
}

func identityFourOrderBasis() [][]float64 {
	return [][]float64{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
}
