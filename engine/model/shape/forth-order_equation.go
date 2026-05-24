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
	Center [3]float64
	Scale  [3]float64
}

func NewFourOrderEquation(A []float64, centerScale ...[]float64) *FourOrderEquation { // Index order: 1, x, y, z
	center, scale := normalizePolynomialCenterScale(centerScale...)
	baked := bakeFourOrderCoefficients(A, center, scale)
	ATensor := maths.NewTensorFromSlice(baked, []int{4, 4, 4, 4})

	return &FourOrderEquation{
		A:      ATensor,
		Center: center,
		Scale:  scale,
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
		coeffs = [5]float64{0, 0, 0, 0, 0} // Initialize coefficients from the constant term to the fourth-degree term.
		stx    = raySt.At(0, 0)            // Get ray origin and direction components.
		sty    = raySt.At(1, 0)
		stz    = raySt.At(2, 0)
		dirx   = rayDir.At(0, 0)
		diry   = rayDir.At(1, 0)
		dirz   = rayDir.At(2, 0)
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

					for d, coef := range poly { // Multiply the current term polynomial coefficients by c and accumulate them.
						coeffs[d] += c * coef
					}
				}
			}
		}
	}

	roots := solveQuarticEquationReal(coeffs[4], coeffs[3], coeffs[2], coeffs[1], coeffs[0])
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

func solveQuarticEquationReal(a4, a3, a2, a1, a0 float64) []float64 {
	if math.Abs(a4) < utils.EPS {
		return solveCubicEquationReal(a3, a2, a1, a0)
	}

	bound := 1 + math.Max(math.Max(math.Abs(a3/a4), math.Abs(a2/a4)), math.Max(math.Abs(a1/a4), math.Abs(a0/a4)))
	points := []float64{-bound}
	points = append(points, solveCubicEquationReal(4*a4, 3*a3, 2*a2, a1)...)
	points = append(points, bound)
	points = uniqueSortedRoots(points)

	const valueEpsilon = 1e-8
	var roots []float64
	for i, point := range points {
		if math.Abs(evalQuartic(a4, a3, a2, a1, a0, point)) < valueEpsilon {
			roots = append(roots, point)
		}
		if i == len(points)-1 {
			continue
		}

		left, right := point, points[i+1]
		fLeft := evalQuartic(a4, a3, a2, a1, a0, left)
		fRight := evalQuartic(a4, a3, a2, a1, a0, right)
		if fLeft == 0 || fRight == 0 || fLeft*fRight > 0 {
			continue
		}
		roots = append(roots, bisectQuarticRoot(a4, a3, a2, a1, a0, left, right))
	}

	return uniqueSortedRoots(roots)
}

func bisectQuarticRoot(a4, a3, a2, a1, a0, left, right float64) float64 {
	fLeft := evalQuartic(a4, a3, a2, a1, a0, left)
	for i := 0; i < 96; i++ {
		mid := 0.5 * (left + right)
		fMid := evalQuartic(a4, a3, a2, a1, a0, mid)
		if math.Abs(fMid) < 1e-12 {
			return mid
		}
		if fLeft*fMid <= 0 {
			right = mid
		} else {
			left = mid
			fLeft = fMid
		}
	}
	return 0.5 * (left + right)
}

func evalQuartic(a4, a3, a2, a1, a0, x float64) float64 {
	return (((a4*x)+a3)*x+a2)*x*x + a1*x + a0
}

func (p *FourOrderEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	var (
		x       = intersect.At(0, 0) // Get intersection coordinates.
		y       = intersect.At(1, 0)
		z       = intersect.At(2, 0)
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

func bakeFourOrderCoefficients(a []float64, center, scale [3]float64) []float64 {
	if polynomialPlacementIsIdentity(center, scale) {
		result := make([]float64, len(a))
		copy(result, a)
		return result
	}

	matrix := polynomialHomogeneousMatrix3D(center, scale)
	result := make([]float64, 256)
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				for l := 0; l < 4; l++ {
					coef := fourOrderCoeff(a, i, j, k, l)
					if coef == 0 {
						continue
					}
					for wi := 0; wi < 4; wi++ {
						for wj := 0; wj < 4; wj++ {
							for wk := 0; wk < 4; wk++ {
								for wl := 0; wl < 4; wl++ {
									result[fourOrderOffset(wi, wj, wk, wl)] += coef * matrix[i][wi] * matrix[j][wj] * matrix[k][wk] * matrix[l][wl]
								}
							}
						}
					}
				}
			}
		}
	}
	return result
}
func fourOrderCoeff(a []float64, i, j, k, l int) float64 {
	return a[fourOrderOffset(i, j, k, l)]
}

func fourOrderOffset(i, j, k, l int) int {
	return ((i*4+j)*4+k)*4 + l
}
