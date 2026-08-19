package emission

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
)

type Sidedness uint8

const (
	FrontSide Sidedness = iota
	BackSide
	TwoSided
)

func (s Sidedness) String() string {
	switch s {
	case FrontSide:
		return "front"
	case BackSide:
		return "back"
	case TwoSided:
		return "two_sided"
	default:
		return "unknown"
	}
}

type AngularSample struct {
	Wo    maths.Direction
	PDF   float64
	Flags DirectionFlags
}

// AngularDistribution is dimensionless. Eval describes relative radiance,
// while PDF samples Eval(wo)*|cos(theta)|, the emitted-flux integrand.
type AngularDistribution interface {
	Eval(wo maths.Direction) float64
	Sample(u maths.Sample2D, dimension int) AngularSample
	PDF(wo maths.Direction) float64
	ProjectedIntegral(dimension int) float64
	Flags() DirectionFlags
}

// CosinePower defines D(wo)=cos(theta)^Exponent on each enabled side.
type CosinePower struct {
	Exponent  float64
	Sidedness Sidedness
}

func NewCosinePower(exponent float64, sidedness Sidedness) (CosinePower, error) {
	if exponent < 0 || math.IsNaN(exponent) || math.IsInf(exponent, 0) {
		return CosinePower{}, fmt.Errorf("cosine-power exponent must be finite and >= 0")
	}
	if sidedness > TwoSided {
		return CosinePower{}, fmt.Errorf("invalid emission sidedness %d", sidedness)
	}
	return CosinePower{Exponent: exponent, Sidedness: sidedness}, nil
}

func NewUniform(sidedness Sidedness) CosinePower {
	return CosinePower{Sidedness: sidedness}
}

func (d CosinePower) Eval(wo maths.Direction) float64 {
	c, ok := d.enabledCosine(wo)
	if !ok {
		return 0
	}
	if d.Exponent == 0 {
		return 1
	}
	return math.Pow(c, d.Exponent)
}

func (d CosinePower) PDF(wo maths.Direction) float64 {
	c, ok := d.enabledCosine(wo)
	if !ok {
		return 0
	}
	z := d.ProjectedIntegral(wo.Len())
	if z <= 0 {
		return 0
	}
	return math.Pow(c, d.Exponent+1) / z
}

func (d CosinePower) Sample(u maths.Sample2D, dimension int) AngularSample {
	if dimension < 2 {
		return AngularSample{}
	}
	// The existing N-D cosine sampler exactly covers the exponent-zero case.
	// General cosine-power sampling is currently needed only by the 3D
	// Euclidean bidirectional integrators.
	if dimension != 3 && d.Exponent != 0 {
		return AngularSample{}
	}

	chooseBack := d.Sidedness == BackSide
	uSide := clampUnit(u.U)
	if d.Sidedness == TwoSided {
		chooseBack = uSide < 0.5
		if chooseBack {
			uSide *= 2
		} else {
			uSide = (uSide - 0.5) * 2
		}
	}
	uSide = clampUnit(uSide)
	if dimension != 3 {
		wo := maths.CosineSampleHemisphereND(maths.Sample2D{U: uSide, V: u.V}, dimension)
		if chooseBack {
			wo = wo.MulScalar(-1)
		}
		return AngularSample{Wo: wo, PDF: d.PDF(wo), Flags: DirectionContinuous}
	}

	cosTheta := math.Pow(uSide, 1/(d.Exponent+2))
	sinTheta := math.Sqrt(math.Max(0, 1-cosTheta*cosTheta))
	phi := 2 * math.Pi * clampUnit(u.V)
	z := cosTheta
	if chooseBack {
		z = -z
	}
	wo := maths.NewDirection(sinTheta*math.Cos(phi), sinTheta*math.Sin(phi), z)
	return AngularSample{Wo: wo, PDF: d.PDF(wo), Flags: DirectionContinuous}
}

func (d CosinePower) ProjectedIntegral(dimension int) float64 {
	if dimension < 2 || d.Sidedness > TwoSided {
		return 0
	}
	// Integral over one hemisphere in R^dimension:
	// pi^((d-1)/2) Gamma((k+2)/2) / Gamma((k+d+1)/2).
	dim := float64(dimension)
	logZ := ((dim - 1) / 2 * math.Log(math.Pi)) +
		lgamma((d.Exponent+2)/2) - lgamma((d.Exponent+dim+1)/2)
	z := math.Exp(logZ)
	if d.Sidedness == TwoSided {
		z *= 2
	}
	return z
}

func (CosinePower) Flags() DirectionFlags { return DirectionContinuous }

func (d CosinePower) enabledCosine(wo maths.Direction) (float64, bool) {
	if wo.Len() < 2 || d.Sidedness > TwoSided {
		return 0, false
	}
	z := wo.Component(wo.Len() - 1)
	switch d.Sidedness {
	case FrontSide:
		return z, z > 0
	case BackSide:
		return -z, z < 0
	case TwoSided:
		return math.Abs(z), z != 0
	default:
		return 0, false
	}
}

func lgamma(x float64) float64 {
	value, _ := math.Lgamma(x)
	return value
}

func clampUnit(v float64) float64 {
	if v <= 0 {
		return math.SmallestNonzeroFloat64
	}
	if v >= 1 {
		return math.Nextafter(1, 0)
	}
	return v
}
