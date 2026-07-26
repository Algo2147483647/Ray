package factory

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type sphericalHarmonicSurface struct {
	terms []sphericalHarmonicTerm
}

type sphericalHarmonicTerm struct {
	L      int
	M      int
	MFloat float64
	Weight float64
	Basis  string
	Norm   float64
}

func parseParametricSphericalHarmonicSurface(surfaceDef map[string]interface{}) (shape.ParametricFunction, shape.ParametricDerivative, error) {
	terms, err := parseSphericalHarmonicTerms(surfaceDef)
	if err != nil {
		return nil, nil, err
	}

	surface := sphericalHarmonicSurface{
		terms: terms,
	}
	return surface.evaluate, surface.derivative, nil
}

func parseSphericalHarmonicTerms(surfaceDef map[string]interface{}) ([]sphericalHarmonicTerm, error) {
	raw, ok := surfaceDef["terms"]
	if !ok {
		return nil, fmt.Errorf(`spherical_harmonic surface requires "terms"`)
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("field %q: expected array, got %T", "terms", raw)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf(`spherical_harmonic surface "terms" must not be empty`)
	}

	terms := make([]sphericalHarmonicTerm, 0, len(items))
	for index, item := range items {
		def, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("terms[%d]: expected object, got %T", index, item)
		}

		l, err := requiredIntegerField(def, "l")
		if err != nil {
			return nil, fmt.Errorf("terms[%d].l: %w", index, err)
		}
		m, err := requiredIntegerField(def, "m")
		if err != nil {
			return nil, fmt.Errorf("terms[%d].m: %w", index, err)
		}
		if l < 0 {
			return nil, fmt.Errorf("terms[%d].l must be >= 0", index)
		}
		if m < 0 || m > l {
			return nil, fmt.Errorf("terms[%d].m must be within [0, l]", index)
		}

		weight := 1.0
		if value, ok, err := utils.OptionalFloat64Field(def, "weight"); err != nil {
			return nil, fmt.Errorf("terms[%d].weight: %w", index, err)
		} else if ok {
			weight = value
		}

		basis := "cos"
		if value, ok, err := utils.OptionalStringField(def, "basis"); err != nil {
			return nil, fmt.Errorf("terms[%d].basis: %w", index, err)
		} else if ok {
			switch value {
			case "cos", "sin":
				basis = value
			default:
				return nil, fmt.Errorf(`terms[%d].basis must be "cos" or "sin"`, index)
			}
		}

		terms = append(terms, sphericalHarmonicTerm{
			L:      l,
			M:      m,
			MFloat: float64(m),
			Weight: weight,
			Basis:  basis,
			Norm:   sphericalHarmonicNorm(l, m),
		})
	}
	return terms, nil
}

func (s sphericalHarmonicSurface) evaluate(u, v float64) *mat.VecDense {
	psi, _, _ := s.valueAndDerivatives(u, v)
	r := math.Abs(psi)
	su, cu := math.Sincos(u)
	sv, cv := math.Sincos(v)
	return mat.NewVecDense(3, []float64{
		r * su * cv,
		r * su * sv,
		r * cu,
	})
}

func (s sphericalHarmonicSurface) derivative(u, v float64, du, dv *mat.VecDense) (*mat.VecDense, *mat.VecDense) {
	if du == nil || du.Len() != 3 {
		du = mat.NewVecDense(3, nil)
	} else {
		du.Zero()
	}
	if dv == nil || dv.Len() != 3 {
		dv = mat.NewVecDense(3, nil)
	} else {
		dv.Zero()
	}

	psi, psiTheta, psiPhi := s.valueAndDerivatives(u, v)
	r := math.Abs(psi)
	rTheta := 0.0
	rPhi := 0.0
	if psi > 0 {
		rTheta = psiTheta
		rPhi = psiPhi
	} else if psi < 0 {
		rTheta = -psiTheta
		rPhi = -psiPhi
	}

	su, cu := math.Sincos(u)
	sv, cv := math.Sincos(v)
	dir := [3]float64{su * cv, su * sv, cu}
	dirTheta := [3]float64{cu * cv, cu * sv, -su}
	dirPhi := [3]float64{-su * sv, su * cv, 0}
	for axis := 0; axis < 3; axis++ {
		du.SetVec(axis, rTheta*dir[axis]+r*dirTheta[axis])
		dv.SetVec(axis, rPhi*dir[axis]+r*dirPhi[axis])
	}
	return du, dv
}

func (s sphericalHarmonicSurface) valueAndDerivatives(theta, phi float64) (float64, float64, float64) {
	psi := 0.0
	psiTheta := 0.0
	psiPhi := 0.0
	for _, term := range s.terms {
		value, thetaDerivative, phiDerivative := term.valueAndDerivatives(theta, phi)
		psi += term.Weight * value
		psiTheta += term.Weight * thetaDerivative
		psiPhi += term.Weight * phiDerivative
	}
	return psi, psiTheta, psiPhi
}

func (t sphericalHarmonicTerm) valueAndDerivatives(theta, phi float64) (float64, float64, float64) {
	x := math.Cos(theta)
	p := associatedLegendre(t.L, t.M, x)
	pTheta := associatedLegendreThetaDerivative(t.L, t.M, theta, x, p)
	scale := t.Norm
	if t.M > 0 {
		scale *= math.Sqrt2
	}

	base := scale * p
	baseTheta := scale * pTheta
	if t.M == 0 {
		return base, baseTheta, 0
	}

	angle := t.MFloat * phi
	sinAngle, cosAngle := math.Sincos(angle)
	switch t.Basis {
	case "sin":
		return base * sinAngle, baseTheta * sinAngle, base * t.MFloat * cosAngle
	default:
		return base * cosAngle, baseTheta * cosAngle, -base * t.MFloat * sinAngle
	}
}

func realSphericalHarmonic(l, m int, basis string, theta, phi float64) float64 {
	x := math.Cos(theta)
	p := associatedLegendre(l, m, x)
	norm := sphericalHarmonicNorm(l, m)
	if m == 0 {
		return norm * p
	}
	value := math.Sqrt2 * norm * p
	switch basis {
	case "sin":
		return value * math.Sin(float64(m)*phi)
	default:
		return value * math.Cos(float64(m)*phi)
	}
}

func sphericalHarmonicNorm(l, m int) float64 {
	logRatio, _ := math.Lgamma(float64(l - m + 1))
	logDenom, _ := math.Lgamma(float64(l + m + 1))
	return math.Sqrt((float64(2*l+1) / (4 * math.Pi)) * math.Exp(logRatio-logDenom))
}

func associatedLegendreThetaDerivative(l, m int, theta, x, p float64) float64 {
	sinTheta := math.Sin(theta)
	if math.Abs(sinTheta) > 1e-8 {
		prev := associatedLegendre(l-1, m, x)
		return (float64(l)*x*p - float64(l+m)*prev) / sinTheta
	}
	if m != 1 {
		return 0
	}
	derivative := 0.5 * float64(l) * float64(l+1)
	if x >= 0 {
		return -derivative
	}
	if l%2 == 0 {
		return -derivative
	}
	return derivative
}

func associatedLegendre(l, m int, x float64) float64 {
	if m < 0 || m > l {
		return 0
	}
	if x > 1 {
		x = 1
	} else if x < -1 {
		x = -1
	}

	pmm := 1.0
	if m > 0 {
		somx2 := math.Sqrt(math.Max(0, (1-x)*(1+x)))
		fact := 1.0
		for i := 1; i <= m; i++ {
			pmm *= -fact * somx2
			fact += 2
		}
	}
	if l == m {
		return pmm
	}

	pmmp1 := x * float64(2*m+1) * pmm
	if l == m+1 {
		return pmmp1
	}

	prev := pmm
	curr := pmmp1
	for ll := m + 2; ll <= l; ll++ {
		next := (float64(2*ll-1)*x*curr - float64(ll+m-1)*prev) / float64(ll-m)
		prev, curr = curr, next
	}
	return curr
}

func requiredIntegerField(data map[string]interface{}, key string) (int, error) {
	value, err := utils.RequiredFloat64Field(data, key)
	if err != nil {
		return 0, err
	}
	integer := int(value)
	if float64(integer) != value {
		return 0, fmt.Errorf("must be an integer")
	}
	return integer, nil
}
