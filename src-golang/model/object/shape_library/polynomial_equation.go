package shape_library

import (
	"gonum.org/v1/gonum/spatial/r3"
	"math"
)

type PolynomialEquation struct {
	Coeffs    [4]float64 // 系数 [a, b, c, d]
	Engraving func(r3.Vec) bool
}

func (p *PolynomialEquation) GetName() string {
	return "PolynomialEquation"
}

func (p *PolynomialEquation) Intersect(raySt, rayDir r3.Vec) float64 {
	a, b, c, d := p.Coeffs[0], p.Coeffs[1], p.Coeffs[2], p.Coeffs[3]

	// 计算二次方程系数
	A := a*rayDir.X*rayDir.X + b*rayDir.Y*rayDir.Y + c*rayDir.Z*rayDir.Z
	B := 2 * (a*raySt.X*rayDir.X + b*raySt.Y*rayDir.Y + c*raySt.Z*rayDir.Z)
	C := a*raySt.X*raySt.X + b*raySt.Y*raySt.Y + c*raySt.Z*raySt.Z - d

	// 求解二次方程
	discriminant := B*B - 4*A*C
	if discriminant < 0 {
		return math.MaxFloat64
	}

	sqrtD := math.Sqrt(discriminant)
	t1 := (-B - sqrtD) / (2 * A)
	t2 := (-B + sqrtD) / (2 * A)

	if t1 > 0 && t1 < t2 {
		return t1
	} else if t2 > 0 {
		return t2
	}
	return math.MaxFloat64
}

func (p *PolynomialEquation) GetNormalVector(intersect r3.Vec) r3.Vec {
	a, b, c := p.Coeffs[0], p.Coeffs[1], p.Coeffs[2]
	return r3.Norm(r3.Vec{
		X: 2 * a * intersect.X,
		Y: 2 * b * intersect.Y,
		Z: 2 * c * intersect.Z,
	})
}

func (p *PolynomialEquation) BuildBoundingBox() (r3.Vec, r3.Vec) {
	return r3.Vec{X: -math.MaxFloat64, Y: -math.MaxFloat64, Z: -math.MaxFloat64},
		r3.Vec{X: math.MaxFloat64, Y: math.MaxFloat64, Z: math.MaxFloat64}
}
