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
	cosTheta := CosTheta(w)
	if cosTheta <= 0 {
		return 0
	}
	return cosTheta / CosineHemisphereIntegral(w.Len())
}

func CosineSampleHemisphereND(sample Sample2D, dim int) Direction {
	if dim <= 3 {
		return CosineSampleHemisphere(sample)
	}

	tangentDim := dim - 1
	uniforms := deterministicUniforms(sample, tangentDim*2+1)
	radius := math.Pow(clamp01(uniforms[0]), 1/float64(tangentDim))
	tangent := uniformSphereDirection(tangentDim, uniforms[1:])

	components := make([]float64, dim)
	tangentLengthSquared := 0.0
	for i := 0; i < tangentDim; i++ {
		components[i] = radius * tangent[i]
		tangentLengthSquared += components[i] * components[i]
	}
	components[dim-1] = math.Sqrt(math.Max(0, 1-tangentLengthSquared))
	return NewDirectionFromComponents(components)
}

func CosineHemisphereIntegral(dim int) float64 {
	if dim <= 1 {
		return 1
	}
	tangentDim := float64(dim - 1)
	return math.Pow(math.Pi, tangentDim/2) / math.Gamma(tangentDim/2+1)
}

func deterministicUniforms(sample Sample2D, count int) []float64 {
	seed := uint64(clamp01(sample.U)*float64(^uint64(0)>>1)) ^
		(bitsRotateLeft64(uint64(clamp01(sample.V)*float64(^uint64(0)>>1)), 32))
	if seed == 0 {
		seed = 0x9e3779b97f4a7c15
	}

	values := make([]float64, count)
	for i := range values {
		seed += 0x9e3779b97f4a7c15
		z := seed
		z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
		z = (z ^ (z >> 27)) * 0x94d049bb133111eb
		z ^= z >> 31
		values[i] = (float64(z>>11) + 0.5) / (1 << 53)
	}
	return values
}

func uniformSphereDirection(dim int, uniforms []float64) []float64 {
	values := make([]float64, dim)
	sum := 0.0
	for i := 0; i < dim; i += 2 {
		u1 := math.Max(1e-12, clamp01(uniforms[i%len(uniforms)]))
		u2 := clamp01(uniforms[(i+1)%len(uniforms)])
		r := math.Sqrt(-2 * math.Log(u1))
		theta := 2 * math.Pi * u2
		values[i] = r * math.Cos(theta)
		sum += values[i] * values[i]
		if i+1 < dim {
			values[i+1] = r * math.Sin(theta)
			sum += values[i+1] * values[i+1]
		}
	}
	if sum == 0 {
		values[0] = 1
		return values
	}
	scale := 1 / math.Sqrt(sum)
	for i := range values {
		values[i] *= scale
	}
	return values
}

func bitsRotateLeft64(x uint64, k int) uint64 {
	const n = 64
	s := uint(k) & (n - 1)
	return x<<s | x>>(n-s)
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
