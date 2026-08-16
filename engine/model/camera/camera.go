package camera

import (
	"math"

	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
)

type Camera struct {
	Film *Film
}

type RayCamera interface {
	GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray
	SetFilm(*Film)
}

func (c *Camera) SetFilm(film *Film) {
	c.Film = film
}

// FilmProjection describes the pinhole-camera mapping of a scene point.
// Jacobian converts an area density at the point into a box-filtered pixel
// density; callers multiply it by abs(dot(surfaceNormal, ToCamera)).
type FilmProjection struct {
	Position []float64
	ToCamera *mat.VecDense
	Distance float64
	Jacobian float64
}

func PixelIndex(x, y float64, width, height int) (int, bool) {
	if width <= 0 || height <= 0 || math.IsNaN(x) || math.IsNaN(y) ||
		math.IsInf(x, 0) || math.IsInf(y, 0) {
		return 0, false
	}
	x_ := int(math.Floor(x + 0.5))
	y_ := int(math.Floor(y + 0.5))
	if x_ < 0 || x_ >= width || y_ < 0 || y_ >= height {
		return 0, false
	}
	return y_*width + x_, true
}

// ProjectiveCamera is implemented by cameras that can receive light-tracing
// splats. ProjectPoint is the inverse of GenerateRay's raster mapping.
type ProjectiveCamera interface {
	RayCamera
	ProjectPoint(point *mat.VecDense) (FilmProjection, bool)
}

type CameraType string

const (
	CameraType3D         CameraType = "3d"
	CameraTypeNDim       CameraType = "n_dim"
	CameraTypeHyperbolic CameraType = "hyperbolic"
	CameraTypeSpherical  CameraType = "spherical"
)
