package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
)

// PolynomialEquation 表示多项式方程定义的曲面
type PolynomialEquation struct {
	BaseShape

	Coeffs [4]float64 // 系数 [a, b, c, d] (a*x² + b*y² + c*z² = d)
}

func NewPolynomialEquation(a, b, c, d float64) *PolynomialEquation {
	return &PolynomialEquation{
		Coeffs: [4]float64{a, b, c, d},
	}
}

func (p *PolynomialEquation) GetName() string {
	return "PolynomialEquation"
}

func (p *PolynomialEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	a, b, c, d := p.Coeffs[0], p.Coeffs[1], p.Coeffs[2], p.Coeffs[3]

	// 提取光线起点和方向分量
	x0 := raySt.AtVec(0)
	y0 := raySt.AtVec(1)
	z0 := raySt.AtVec(2)
	dx := rayDir.AtVec(0)
	dy := rayDir.AtVec(1)
	dz := rayDir.AtVec(2)

	// 计算二次方程系数
	A := a*dx*dx + b*dy*dy + c*dz*dz
	B := 2 * (a*x0*dx + b*y0*dy + c*z0*dz)
	C := a*x0*x0 + b*y0*y0 + c*z0*z0 - d

	// 处理退化情况（A接近0）
	if math.Abs(A) < math_lib.EPS {
		// 退化为线性方程
		if math.Abs(B) < math_lib.EPS {
			return math.MaxFloat64 // 无解
		}
		t := -C / B
		if t > 0 {
			return t
		}
		return math.MaxFloat64
	}

	// 计算判别式
	discriminant := B*B - 4*A*C
	if discriminant < 0 {
		return math.MaxFloat64 // 无实根
	}

	// 计算根
	sqrtD := math.Sqrt(discriminant)
	t1 := (-B - sqrtD) / (2 * A)
	t2 := (-B + sqrtD) / (2 * A)

	// 寻找最小正根
	minT := math.MaxFloat64
	if t1 > 0 && t1 < minT {
		minT = t1
	}
	if t2 > 0 && t2 < minT {
		minT = t2
	}

	if minT < math.MaxFloat64 {
		return minT
	}
	return math.MaxFloat64
}

func (p *PolynomialEquation) GetNormalVector(intersect *mat.VecDense) *mat.VecDense {
	a, b, c := p.Coeffs[0], p.Coeffs[1], p.Coeffs[2]
	// 计算梯度（法向量）
	grad := mat.NewVecDense(3, []float64{
		2 * a * intersect.AtVec(0),
		2 * b * intersect.AtVec(1),
		2 * c * intersect.AtVec(2),
	})

	// 归一化
	length := mat.Norm(grad, 2)
	if length < math_lib.EPS {
		return mat.NewVecDense(3, nil) // 零向量
	}
	grad.ScaleVec(1/length, grad)
	return grad
}

func (p *PolynomialEquation) BuildBoundingBox() (*mat.VecDense, *mat.VecDense) {

	maxVal := math.MaxFloat64 / 2 // 多项式曲面通常无界, 避免后续计算溢出
	minVec := mat.NewVecDense(3, []float64{-maxVal, -maxVal, -maxVal})
	maxVec := mat.NewVecDense(3, []float64{maxVal, maxVal, maxVal})
	return minVec, maxVec
}
