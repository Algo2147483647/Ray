package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type CubicEquation struct {
	BaseShape
	A *maths.Tensor[float64] `json:"a"`
}

func NewCubicEquation(A []float64) *CubicEquation {
	return &CubicEquation{
		A: maths.NewTensorFromSlice(A, []int{4, 4, 4}),
	}
}

func (p *CubicEquation) Name() string {
	return "Cubic Equation"
}

func (p *CubicEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := p.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (p *CubicEquation) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	var (
		coeffs = []float64{}
		stx    = raySt.AtVec(0)
		sty    = raySt.AtVec(1)
		stz    = raySt.AtVec(2)
		dirx   = rayDir.AtVec(0)
		diry   = rayDir.AtVec(1)
		dirz   = rayDir.AtVec(2)
	)

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				c := p.A.Get(i, j, k)
				if c == 0 {
					continue
				}

				poly := [4]float64{1, 0, 0, 0}
				for _, idx := range [3]int{i, j, k} {
					var factor [2]float64
					switch idx {
					case 0:
						factor = [2]float64{1, 0}
					case 1:
						factor = [2]float64{stx, dirx}
					case 2:
						factor = [2]float64{sty, diry}
					case 3:
						factor = [2]float64{stz, dirz}
					}

					next := [4]float64{}
					for d1, coef1 := range poly {
						for d2, coef2 := range factor {
							if d1+d2 < len(next) {
								next[d1+d2] += coef1 * coef2
							}
						}
					}
					poly = next
				}

				for d, coef := range poly {
					coeffs[d] += c * coef
				}
			}
		}
	}

	roots, err := maths.SolvePolynomialReal(coeffs)
	if err != nil {
		return SurfaceInteraction{}, false
	}

	res := math.MaxFloat64
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

func (p *CubicEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	var (
		x       = intersect.AtVec(0)
		y       = intersect.AtVec(1)
		z       = intersect.AtVec(2)
		factors = [4]float64{1, x, y, z}
		grad    = [3]float64{}
	)

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				c := p.A.Get(i, j, k)
				if c == 0 {
					continue
				}

				if i == 1 {
					grad[0] += c * factors[j] * factors[k]
				}
				if j == 1 {
					grad[0] += c * factors[i] * factors[k]
				}
				if k == 1 {
					grad[0] += c * factors[i] * factors[j]
				}

				if i == 2 {
					grad[1] += c * factors[j] * factors[k]
				}
				if j == 2 {
					grad[1] += c * factors[i] * factors[k]
				}
				if k == 2 {
					grad[1] += c * factors[i] * factors[j]
				}

				if i == 3 {
					grad[2] += c * factors[j] * factors[k]
				}
				if j == 3 {
					grad[2] += c * factors[i] * factors[k]
				}
				if k == 3 {
					grad[2] += c * factors[i] * factors[j]
				}
			}
		}
	}

	res.SetVec(0, grad[0])
	res.SetVec(1, grad[1])
	res.SetVec(2, grad[2])
	return maths.Normalize(res)
}
