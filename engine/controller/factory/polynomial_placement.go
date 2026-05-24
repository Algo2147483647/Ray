package factory

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

func normalizePolynomialCenterScale(center, scale []float64) ([3]float64, [3]float64, error) {
	normalizedCenter := [3]float64{}
	normalizedScale := [3]float64{1, 1, 1}

	if center != nil {
		if err := utils.RequireSliceLength("center", center, utils.Dimension); err != nil {
			return normalizedCenter, normalizedScale, err
		}
		copy(normalizedCenter[:], center)
	}
	if scale != nil {
		if err := utils.RequireSliceLength("scale", scale, utils.Dimension); err != nil {
			return normalizedCenter, normalizedScale, err
		}
		for i, value := range scale {
			if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				return normalizedCenter, normalizedScale, errInvalidScale(i)
			}
			normalizedScale[i] = value
		}
	}
	return normalizedCenter, normalizedScale, nil
}

func errInvalidScale(index int) error {
	return fmt.Errorf("scale index %d must be a finite positive number", index)
}

func polynomialPlacementIsIdentity(center, scale [3]float64) bool {
	return center == [3]float64{} && scale == [3]float64{1, 1, 1}
}

func polynomialHomogeneousMatrix3D(center, scale [3]float64) [4][4]float64 {
	return [4][4]float64{
		{1, 0, 0, 0},
		{-center[0] / scale[0], 1 / scale[0], 0, 0},
		{-center[1] / scale[1], 0, 1 / scale[1], 0},
		{-center[2] / scale[2], 0, 0, 1 / scale[2]},
	}
}

func bakeQuadraticCoefficients(a *mat.Dense, b *mat.VecDense, c float64, center, scale [3]float64) (*mat.Dense, *mat.VecDense, float64) {
	if polynomialPlacementIsIdentity(center, scale) {
		return mat.DenseCopyOf(a), mat.VecDenseCopyOf(b), c
	}

	d := mat.NewDense(3, 3, []float64{
		1 / scale[0], 0, 0,
		0, 1 / scale[1], 0,
		0, 0, 1 / scale[2],
	})
	e := mat.NewVecDense(3, []float64{
		-center[0] / scale[0],
		-center[1] / scale[1],
		-center[2] / scale[2],
	})

	var aD mat.Dense
	aD.Mul(a, d)

	var worldA mat.Dense
	worldA.Mul(d.T(), &aD)

	var aPlusAT mat.Dense
	aPlusAT.Add(a, a.T())

	tmp := mat.NewVecDense(3, nil)
	tmp.MulVec(&aPlusAT, e)
	worldB := mat.NewVecDense(3, nil)
	worldB.MulVec(d.T(), tmp)
	worldB.AddVec(worldB, scaledByDiagonal(b, d))

	aE := mat.NewVecDense(3, nil)
	aE.MulVec(a, e)
	worldC := mat.Dot(e, aE) + mat.Dot(b, e) + c

	return &worldA, worldB, worldC
}

func bakeCubicCoefficients(a []float64, center, scale [3]float64) []float64 {
	if polynomialPlacementIsIdentity(center, scale) {
		result := make([]float64, len(a))
		copy(result, a)
		return result
	}

	matrix := polynomialHomogeneousMatrix3D(center, scale)
	result := make([]float64, 64)
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				coef := cubicCoeff(a, i, j, k)
				if coef == 0 {
					continue
				}
				for wi := 0; wi < 4; wi++ {
					for wj := 0; wj < 4; wj++ {
						for wk := 0; wk < 4; wk++ {
							result[cubicOffset(wi, wj, wk)] += coef * matrix[i][wi] * matrix[j][wj] * matrix[k][wk]
						}
					}
				}
			}
		}
	}
	return result
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

func scaledByDiagonal(v *mat.VecDense, d *mat.Dense) *mat.VecDense {
	result := mat.NewVecDense(3, nil)
	for i := 0; i < 3; i++ {
		result.SetVec(i, v.AtVec(i)*d.At(i, i))
	}
	return result
}

func cubicCoeff(a []float64, i, j, k int) float64 {
	return a[cubicOffset(i, j, k)]
}

func cubicOffset(i, j, k int) int {
	return (i*4+j)*4 + k
}

func fourOrderCoeff(a []float64, i, j, k, l int) float64 {
	return a[fourOrderOffset(i, j, k, l)]
}

func fourOrderOffset(i, j, k, l int) int {
	return ((i*4+j)*4+k)*4 + l
}
