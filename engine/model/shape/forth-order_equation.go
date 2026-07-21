package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type FourOrderEquation struct {
	BaseShape
	A *maths.Tensor[float64] `json:"a"`
}

func NewFourOrderEquation(A []float64) *FourOrderEquation { // Index order: 1, x, y, z
	ATensor := maths.NewTensorFromSlice(A, []int{4, 4, 4, 4})

	return &FourOrderEquation{
		A: ATensor,
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
	var (
		coeffs = []float64{0, 0, 0, 0, 0} // Coefficients from the fourth-degree term to the constant term.
		stx    = raySt.AtVec(0)           // Get ray origin and direction components.
		sty    = raySt.AtVec(1)
		stz    = raySt.AtVec(2)
		dirx   = rayDir.AtVec(0)
		diry   = rayDir.AtVec(1)
		dirz   = rayDir.AtVec(2)
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
	var (
		x       = intersect.AtVec(0) // Get intersection coordinates.
		y       = intersect.AtVec(1)
		z       = intersect.AtVec(2)
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

	res.SetVec(0, grad[0])
	res.SetVec(1, grad[1])
	res.SetVec(2, grad[2])
	return maths.Normalize(res)
}
