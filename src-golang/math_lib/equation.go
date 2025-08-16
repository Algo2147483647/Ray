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

// 解二次方程 (实数根): ax² + bx + c = 0// 返回根数量及有效根
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

// 解三次方程: ax³ + bx² + cx + d = 0
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
func SolveQuarticEquation(a, b, c, d, e complex128) [4]complex128 {
	if a == 0 {
		// 降次为三次方程
		roots := SolveCubicEquation(real(b), real(c), real(d), real(e))
		return [4]complex128{
			complex(real(roots[0]), imag(roots[0])),
			complex(real(roots[1]), imag(roots[1])),
			complex(real(roots[2]), imag(roots[2])),
			cmplx.Inf(),
		}
	}

	// 正规化系数
	b, c, d, e = b/a, c/a, d/a, e/a

	// 预计算中间变量
	Q1 := c*c - 3*b*d + 12*e
	Q2 := 2*c*c*c - 9*b*c*d + 27*d*d + 27*b*b*e - 72*c*e
	Q3 := 8*b*c - 16*d - 2*b*b*b
	Q4 := 3*b*b - 8*c

	// 计算中间根
	inner := cmplx.Sqrt(Q2*Q2/4 - Q1*Q1*Q1)
	Q5 := cmplx.Pow(Q2/2+inner, 1.0/3)
	Q6 := (Q1/Q5 + Q5) / 3
	Q7 := 2 * cmplx.Sqrt(Q4/12+Q6)

	// 计算最终根
	term := cmplx.Sqrt(complex(4, 0)*Q4/complex(6, 0) - complex(4, 0)*Q6 - Q3/Q7)
	return [4]complex128{
		(-b - Q7 - term) / 4,
		(-b - Q7 + term) / 4,
		(-b + Q7 - cmplx.Sqrt(complex(4, 0)*Q4/complex(6, 0)-complex(4, 0)*Q6+Q3/Q7)) / 4,
		(-b + Q7 + cmplx.Sqrt(complex(4, 0)*Q4/complex(6, 0)-complex(4, 0)*Q6+Q3/Q7)) / 4,
	}
}
