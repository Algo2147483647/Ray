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
	Raster   RasterPosition
	ToCamera *mat.VecDense
	Distance float64
	Jacobian float64
}

// RasterPosition is the single source of truth for a 2D Film projection.
// Integer coordinates are pixel centers; half-integers are pixel boundaries.
type RasterPosition struct {
	X float64
	Y float64
}

func (p RasterPosition) PixelIndex(width, height int) (int, bool) {
	if width <= 0 || height <= 0 || math.IsNaN(p.X) || math.IsNaN(p.Y) ||
		math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) {
		return 0, false
	}
	x := int(math.Floor(p.X + 0.5))
	y := int(math.Floor(p.Y + 0.5))
	if x < 0 || x >= width || y < 0 || y >= height {
		return 0, false
	}
	return y*width + x, true
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
