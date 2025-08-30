package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
)

type FourOrderEquation struct {
	BaseShape
	A *math_lib.Tensor[float64] `json:"a"`
}

func NewFourOrderEquation(A []float64) *FourOrderEquation {
	ATensor := math_lib.NewTensorFromSlice(A, []int{4, 4, 4, 4})

	return &FourOrderEquation{
		A: ATensor,
	}
}

func (p *FourOrderEquation) Name() string {
	return "Four-Order Equation"
}

func (p *FourOrderEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	var (
		coeffs = [5]float64{0, 0, 0, 0, 0} // 初始化系数数组，索引0到4分别对应常数项到4次项
		stx    = raySt.At(0, 0)            // 获取射线起点和方向的分量
		sty    = raySt.At(1, 0)
		stz    = raySt.At(2, 0)
		dirx   = rayDir.At(0, 0)
		diry   = rayDir.At(1, 0)
		dirz   = rayDir.At(2, 0)
	)

	for i := 0; i < 4; i++ { // 遍历张量A的索引（0到3）
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				for l := 0; l < 4; l++ {
					c := p.A.Get(i, j, k, l)
					if c == 0 {
						continue
					}

					poly := [5]float64{1, 0, 0, 0, 0} // 初始化当前项的多项式系数（常数项为1）
					indices := [4]int{i, j, k, l}     // 处理每个索引对应的因子
					for _, idx := range indices {
						var polyFactor [2]float64
						switch idx {
						case 0:
							polyFactor = [2]float64{1, 0} // 常数因子1
						case 1:
							polyFactor = [2]float64{stx, dirx} // x因子
						case 2:
							polyFactor = [2]float64{sty, diry} // y因子
						case 3:
							polyFactor = [2]float64{stz, dirz} // z因子
						default:
							polyFactor = [2]float64{0, 0} // 无效索引，默认为0
						}

						newPoly := [5]float64{} // 将当前多项式与因子多项式相乘
						for d1, coef1 := range poly {
							for d2, coef2 := range polyFactor {
								if d1+d2 < 5 {
									newPoly[d1+d2] += coef1 * coef2
								}
							}
						}
						poly = newPoly
					}

					for d, coef := range poly { // 将当前项的多项式系数乘以c并累加到总系数中
						coeffs[d] += c * coef
					}
				}
			}
		}
	}

	roots := math_lib.SolveQuarticEquation(coeffs[4], coeffs[3], coeffs[2], coeffs[1], coeffs[0]) // 解四次方程：a*t^4 + b*t^3 + c*t^2 + d*t + e = 0
	res := math.MaxFloat64                                                                        // 寻找最小的正实数根
	for _, root := range roots {
		if math.Abs(imag(root)) < math_lib.EPS && real(root) > math_lib.EPS && real(root) < res {
			res = real(root)
		}
	}
	return res
}

func (p *FourOrderEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	var (
		x       = intersect.At(0, 0) // 获取交点坐标
		y       = intersect.At(1, 0)
		z       = intersect.At(2, 0)
		factors = [4]float64{1, x, y, z} // factors[0]=1, factors[1]=x, factors[2]=y, factors[3]=z
		grad    = [3]float64{0, 0, 0}    // dx, dy, dz	// 初始化梯度向量
	)

	// 遍历张量A的索引（0到3）
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				for l := 0; l < 4; l++ {
					c := p.A.Get(i, j, k, l)
					if c == 0 {
						continue
					}

					// 计算对x的偏导贡献
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

					// 计算对y的偏导贡献
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

					// 计算对z的偏导贡献
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
	return math_lib.Normalize(res)
}
