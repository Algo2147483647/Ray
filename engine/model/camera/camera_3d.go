package camera

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/maths"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
)

type Camera3D struct {
	Camera
	Position               *mat.VecDense   // Camera origin in scene space.
	Coordinates            []*mat.VecDense // Camera basis vectors: forward, right, up.
	FieldOfViews           []float64       // Vertical and horizontal field-of-view angles in degrees.
	Ortho                  bool            // Uses orthographic projection when true.
	orthonormalCoordinates []*mat.VecDense // Normalized camera basis vectors.
	halfWidth              float64         // Half-width of the view plane.
	halfHeight             float64         // Half-height of the view plane.
	prepared               bool            // Indicates cached camera basis is ready.
}

func NewCamera3D() *Camera3D {
	return &Camera3D{}
}

func (c *Camera3D) Prepare() error {
	if c.Position == nil {
		return fmt.Errorf("camera position is not configured")
	} else if len(c.Coordinates) != 3 {
		return fmt.Errorf("camera coordinates must contain forward, right, and up vectors")
	}
	halfHeight := math.Tan(c.FieldOfViews[0] * math.Pi / 180 / 2)
	halfWidth := math.Tan(c.FieldOfViews[1] * math.Pi / 180 / 2)

	c.orthonormalCoordinates = maths.GramSchmidt(c.Coordinates...)
	if len(c.orthonormalCoordinates) != 3 || mat.Norm(c.orthonormalCoordinates[0], 2) == 0 || mat.Norm(c.orthonormalCoordinates[1], 2) == 0 || mat.Norm(c.orthonormalCoordinates[2], 2) == 0 {
		return fmt.Errorf("camera coordinates must be linearly independent")
	}

	c.halfHeight = halfHeight
	c.halfWidth = halfWidth
	c.prepared = true

	return nil
}

func (c *Camera3D) GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray {
	if res == nil {
		res = &renderray.Ray{}
	}
	res.Init()

	if !c.prepared {
		if err := c.Prepare(); err != nil {
			panic(err)
		}
	}
	width, height := c.Film.Shape[0], c.Film.Shape[1]

	var (
		row, col = index[0], index[1]
		u        = 2*(float64(row)+rand.Float64())/float64(width) - 1
		v        = 2*(float64(col)+rand.Float64())/float64(height) - 1
	)

	res.Origin.CloneFromVec(c.Position)
	res.Direction.CloneFromVec(c.orthonormalCoordinates[0])
	res.Direction.AddScaledVec(res.Direction, u*c.halfWidth, c.orthonormalCoordinates[1])
	res.Direction.AddScaledVec(res.Direction, -v*c.halfHeight, c.orthonormalCoordinates[2])
	maths.Normalize(res.Direction)

	return res
}

// ProjectPoint maps a world-space point to the box-filtered pinhole film.
// The returned Jacobian omits the receiving surface cosine because the camera
// does not know that surface's normal.
func (c *Camera3D) ProjectPoint(point *mat.VecDense) (FilmProjection, bool) {
	if point == nil || point.Len() != 3 {
		return FilmProjection{}, false
	}
	if !c.prepared {
		if err := c.Prepare(); err != nil {
			return FilmProjection{}, false
		}
	}
	width, height := c.Film.Shape[0], c.Film.Shape[1]

	fromCamera := mat.NewVecDense(3, nil)
	fromCamera.SubVec(point, c.Position)
	forwardDistance := mat.Dot(fromCamera, c.orthonormalCoordinates[0])
	if forwardDistance <= 0 {
		return FilmProjection{}, false
	}

	distance := mat.Norm(fromCamera, 2)
	if distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return FilmProjection{}, false
	}
	x := mat.Dot(fromCamera, c.orthonormalCoordinates[1]) / forwardDistance
	y := mat.Dot(fromCamera, c.orthonormalCoordinates[2]) / forwardDistance
	u := x / c.halfWidth
	v := -y / c.halfHeight
	// Make the optical-axis mapping deterministic for even-sized films. Dot
	// products of an analytically perpendicular basis can retain a few ULPs of
	// signed error, otherwise placing the center ray one pixel left or above.
	if math.Abs(u) < 1e-14 {
		u = 0
	}
	if math.Abs(v) < 1e-14 {
		v = 0
	}
	if u < -1 || u >= 1 || v < -1 || v >= 1 {
		return FilmProjection{}, false
	}

	raster := RasterPosition{
		X: (u+1)*0.5*float64(width) - 0.5,
		Y: (v+1)*0.5*float64(height) - 0.5,
	}
	if _, ok := raster.PixelIndex(width, height); !ok {
		return FilmProjection{}, false
	}

	cosAxis := forwardDistance / distance
	denominator := 4 * c.halfWidth * c.halfHeight * cosAxis * cosAxis * cosAxis * distance * distance
	if denominator <= 0 || math.IsNaN(denominator) || math.IsInf(denominator, 0) {
		return FilmProjection{}, false
	}

	toCamera := mat.VecDenseCopyOf(fromCamera)
	toCamera.ScaleVec(-1/distance, toCamera)
	return FilmProjection{
		Raster:   raster,
		ToCamera: toCamera,
		Distance: distance,
		Jacobian: float64(width*height) / denominator,
	}, true
}
