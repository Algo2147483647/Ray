package maths

import "math"

type Sample2D struct {
	U float64
	V float64
}

func CosineSampleHemisphere(sample Sample2D) Direction {
	r := math.Sqrt(clamp01(sample.U))
	theta := 2 * math.Pi * clamp01(sample.V)
	x := r * math.Cos(theta)
	y := r * math.Sin(theta)
	z := math.Sqrt(math.Max(0, 1-x*x-y*y))
	return Direction{X: x, Y: y, Z: z}
}

func CosineHemispherePDF(w Direction) float64 {
	if w.Z <= 0 {
		return 0
	}
	return w.Z / math.Pi
}

func UniformHemisphereDirection(i, n int) Direction {
	if n <= 0 {
		return Direction{Z: 1}
	}
	u := (float64(i) + 0.5) / float64(n)
	z := u
	r := math.Sqrt(math.Max(0, 1-z*z))
	phi := math.Pi * (3 - math.Sqrt(5)) * float64(i)
	return Direction{
		X: r * math.Cos(phi),
		Y: r * math.Sin(phi),
		Z: z,
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
