package example_lib

import (
	"gonum.org/v1/gonum/mat"
	"math"
)

type EngravingFunc func(data map[string]interface{}) bool

var EngravingFuncMap map[string]EngravingFunc = map[string]EngravingFunc{
	"sphere1": EngravingFuncSphere1,
}

func EngravingFuncSphere1(data map[string]interface{}) bool {
	raySt := data["ray_start"].(*mat.VecDense)
	rayDir := data["ray_dir"].(*mat.VecDense)
	distance := data["distance"].(float64)
	center := data["center"].(*mat.VecDense)
	r := data["r"].(float64)

	// Compute the intersection position.
	intersection := mat.NewVecDense(raySt.Len(), nil)
	intersection.ScaleVec(distance, rayDir)
	intersection.AddVec(intersection, raySt)

	// Convert to coordinates relative to the sphere center.
	relPos := mat.NewVecDense(raySt.Len(), nil)
	relPos.SubVec(intersection, center)

	// Compute spherical coordinates (azimuth and polar angle), normalized to the unit sphere.
	x, y, z := relPos.At(0, 0)/r, relPos.At(1, 0)/r, relPos.At(2, 0)/r
	azimuth := math.Atan2(y, x) // Compute azimuth (0 to 2*pi).
	if azimuth < 0 {
		azimuth += 2 * math.Pi
	}
	polar := math.Acos(z) // Compute polar angle (0 to pi).

	// Spiral stripes, using polar angle and azimuth to create the spiral effect.
	spiralParam := polar*5 + azimuth*3

	// Use a sine function to create a smooth stripe pattern.
	patternValue := math.Sin(spiralParam)

	// If patternValue is greater than 0.3, transmit (true); otherwise intersect (false).
	// Adjust this threshold to change the pattern density.
	return patternValue > 0.3
}
