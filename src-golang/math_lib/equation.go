package math_lib

import (
	"errors"
	"math"
	"math/cmplx"
)

// 解线性方程: ax + b = 0
func SolveLinearEquation(a, b float64) (float64, error) {
	if a == 0 {
		return 0, errors.New("singular equation")
	}
	return -b / a, nil
}

// 解二次方程 (复数根): ax² + bx + c = 0
func SolveQuadraticEquation(a, b, c float64) (complex128, complex128) {
	if a == 0 {
		root, _ := SolveLinearEquation(b, c)
		return complex(root, 0), cmplx.Inf()
	}

	delta := complex(b*b-4*a*c, 0)
	sqrtDelta := cmplx.Sqrt(delta)
	denom := complex(2*a, 0)
	return (-complex(b, 0) + sqrtDelta) / denom, (-complex(b, 0) - sqrtDelta) / denom
}

// SolveQuadraticEquationReal 解二次方程 (实数根): ax² + bx + c = 0// 返回根数量及有效根
func SolveQuadraticEquationReal(a, b, c float64) (root1, root2 float64, count int) {
	if a == 0 {
		if root, err := SolveLinearEquation(b, c); err == nil {
			return root, math.MaxFloat64, 1
		}
		return math.MaxFloat64, math.MaxFloat64, 0
	}

	root1, root2 = math.MaxFloat64, math.MaxFloat64
	delta := b*b - 4*a*c
	switch {
	case delta < 0:
		return math.MaxFloat64, math.MaxFloat64, 0
	case delta == 0:
		return -b / (2 * a), math.MaxFloat64, 1
	default:
		sqrtDelta := math.Sqrt(delta)
		denom := 2 * a
		return (-b + sqrtDelta) / denom, (-b - sqrtDelta) / denom, 2
	}
}

// SolveCubicEquation 解三次方程: ax³ + bx² + cx + d = 0
func SolveCubicEquation(a, b, c, d float64) [3]complex128 {
	if a == 0 {
		// 降次为二次方程
		r1, r2 := SolveQuadraticEquation(b, c, d)
		return [3]complex128{r1, r2, cmplx.Inf()}
	}

	// 正规化系数
	b, c, d = b/a, c/a, d/a

	// 卡丹公式参数
	p := (3*c - b*b) / 3
	q := (2*b*b*b - 9*b*c + 27*d) / 27
	delta := cmplx.Pow(complex(q/2, 0), 2) + cmplx.Pow(complex(p/3, 0), 3)

	// 计算根
	u := cmplx.Pow(-complex(q/2, 0)+cmplx.Sqrt(delta), 1.0/3)
	v := cmplx.Pow(-complex(q/2, 0)-cmplx.Sqrt(delta), 1.0/3)
	w := complex(-0.5, math.Sqrt(3)/2)

	y0 := u + v
	y1 := w*u + cmplx.Conj(w)*v
	y2 := cmplx.Conj(w)*u + w*v

	// 调整根并返回
	offset := complex(-b/3, 0)
	return [3]complex128{y0 + offset, y1 + offset, y2 + offset}
}

// 解四次方程: ax⁴ + bx³ + cx² + dx + e = 0
func SolveQuarticEquation(a_, b_, c_, d_, e_ float64) [4]complex128 {
	if a_ == 0 {
		// 降次为三次方程
		roots := SolveCubicEquation(b_, c_, d_, e_)
		return [4]complex128{
			complex(real(roots[0]), imag(roots[0])),
			complex(real(roots[1]), imag(roots[1])),
			complex(real(roots[2]), imag(roots[2])),
			cmplx.Inf(),
		}
	}

	var (
		b, c, d, e = complex(b_/a_, 0), complex(c_/a_, 0), complex(d_/a_, 0), complex(e_/a_, 0) // 正规化系数
		Q1         = c*c - 3*b*d + 12*e                                                         // 预计算中间变量
		Q2         = 2*c*c*c - 9*b*c*d + 27*d*d + 27*b*b*e - 72*c*e
		Q3         = 8*b*c - 16*d - 2*b*b*b
		Q4         = 3*b*b - 8*c
		inner      = cmplx.Sqrt(Q2*Q2/4 - Q1*Q1*Q1) // 计算中间根
		Q5         = cmplx.Pow(Q2/2+inner, 1.0/3)
		Q6         = (Q1/Q5 + Q5) / 3
		Q7         = 2 * cmplx.Sqrt(Q4/12+Q6)
		term       = cmplx.Sqrt(complex(4, 0)*Q4/complex(6, 0) - complex(4, 0)*Q6 - Q3/Q7) // 计算最终根
	)

	return [4]complex128{
		(-b - Q7 - term) / 4,
		(-b - Q7 + term) / 4,
		(-b + Q7 - cmplx.Sqrt(complex(4, 0)*Q4/complex(6, 0)-complex(4, 0)*Q6+Q3/Q7)) / 4,
		(-b + Q7 + cmplx.Sqrt(complex(4, 0)*Q4/complex(6, 0)-complex(4, 0)*Q6+Q3/Q7)) / 4,
	}
}

// NewtonRaphson 实现牛顿-拉弗森方法求解非线性方程组
func NewtonRaphson(f func([]float64) []float64, x0 []float64, tol float64, maxIter int) ([]float64, bool) {
	n := len(x0)
	x := make([]float64, n)
	copy(x, x0)

	for iter := 0; iter < maxIter; iter++ {
		// 计算函数值
		fx := f(x)

		// 检查收敛
		maxError := 0.0
		for i := 0; i < len(fx); i++ {
			if math.Abs(fx[i]) > maxError {
				maxError = math.Abs(fx[i])
			}
		}
		if maxError < tol {
			return x, true
		}

		// 数值计算雅可比矩阵
		jacobian := NumericalJacobian(f, x, 1e-6)

		// 求解线性方程组 J * Δx = -f(x)
		rhs := make([]float64, len(fx))
		for i := range rhs {
			rhs[i] = -fx[i]
		}

		// 使用高斯消元法求解
		delta, success := SolveLinearSystem(jacobian, rhs)
		if !success {
			return nil, false
		}

		// 更新解
		for i := 0; i < n; i++ {
			x[i] += delta[i]
		}
	}

	return nil, false
}

// NumericalJacobian 数值计算雅可比矩阵
func NumericalJacobian(f func([]float64) []float64, x []float64, eps float64) [][]float64 {
	n := len(x)
	fx := f(x)
	m := len(fx)

	jacobian := make([][]float64, m)
	for i := range jacobian {
		jacobian[i] = make([]float64, n)
	}

	for j := 0; j < n; j++ {
		// 向前差分
		xPlus := make([]float64, n)
		copy(xPlus, x)
		xPlus[j] += eps

		fxPlus := f(xPlus)

		for i := 0; i < m; i++ {
			jacobian[i][j] = (fxPlus[i] - fx[i]) / eps
		}
	}

	return jacobian
}

// SolveLinearSystem 使用高斯消元法求解线性方程组
func SolveLinearSystem(A [][]float64, b []float64) ([]float64, bool) {
	n := len(b)

	// 创建增广矩阵
	augmented := make([][]float64, n)
	for i := range augmented {
		augmented[i] = make([]float64, n+1)
		copy(augmented[i], A[i])
		augmented[i][n] = b[i]
	}

	// 前向消元
	for i := 0; i < n; i++ {
		// 寻找主元
		maxRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(augmented[k][i]) > math.Abs(augmented[maxRow][i]) {
				maxRow = k
			}
		}

		// 交换行
		augmented[i], augmented[maxRow] = augmented[maxRow], augmented[i]

		// 检查主元是否为零
		if math.Abs(augmented[i][i]) < 1e-12 {
			return nil, false
		}

		// 消元
		for k := i + 1; k < n; k++ {
			factor := augmented[k][i] / augmented[i][i]
			for j := i; j < n+1; j++ {
				augmented[k][j] -= factor * augmented[i][j]
			}
		}
	}

	// 回代
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		x[i] = augmented[i][n]
		for j := i + 1; j < n; j++ {
			x[i] -= augmented[i][j] * x[j]
		}
		x[i] /= augmented[i][i]
	}

	return x, true
}
