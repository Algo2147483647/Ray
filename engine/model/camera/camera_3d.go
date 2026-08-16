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
	CameraBase
	Position     *mat.VecDense // Camera origin in scene space.
	Direction    *mat.VecDense // Forward viewing direction.
	Up           *mat.VecDense // Up vector defining camera roll.
	Width        int           // Film width in pixels.
	Height       int           // Film height in pixels.
	FieldOfViews []float64     // Vertical and horizontal field-of-view angles in degrees.
	Ortho        bool          // Uses orthographic projection when true.
	dir          *mat.VecDense // Normalized viewing direction.
	up           *mat.VecDense // Normalized camera up vector.
	right        *mat.VecDense // Normalized camera right vector.
	halfWidth    float64       // Half-width of the view plane.
	halfHeight   float64       // Half-height of the view plane.
	invWidth2    float64       // Reciprocal of twice the film width.
	invHeight2   float64       // Reciprocal of twice the film height.
	prepared     bool          // Indicates cached camera basis is ready.
}

func NewCamera3D() *Camera3D {
	return &Camera3D{}
}

func (c *Camera3D) Prepare() error {
	if c.Position == nil {
		return fmt.Errorf("camera position is not configured")
	} else if c.Direction == nil {
		return fmt.Errorf("camera direction is not configured")
	} else if c.Up == nil {
		return fmt.Errorf("camera up vector is not configured")
	} else if c.Width <= 0 {
		return fmt.Errorf("camera width must be > 0")
	} else if c.Height <= 0 {
		return fmt.Errorf("camera height must be > 0")
	} else if mat.Norm(c.Direction, 2) == 0 {
		return fmt.Errorf("camera direction must not be zero")
	} else if mat.Norm(c.Up, 2) == 0 {
		return fmt.Errorf("camera up vector must not be zero")
	}
	halfHeight, halfWidth, err := frameHalfExtents(c.FieldOfViews)
	if err != nil {
		return err
	}

	c.dir = mat.VecDenseCopyOf(c.Direction)
	maths.Normalize(c.dir)
	right := maths.Cross2(c.dir, c.Up)
	if mat.Norm(right, 2) == 0 {
		return fmt.Errorf("camera direction and up vector must not be parallel")
	}
	c.right = maths.Normalize(right)
	// Rebuild the cached up vector from the other two axes. The configured Up
	// vector is an orientation hint and need not already be perpendicular to
	// Direction (look-at cameras commonly use the world-up axis). GenerateRay
	// and ProjectPoint must share an orthonormal basis to be exact inverses.
	c.up = maths.Normalize(maths.Cross2(c.right, c.dir))

	c.halfHeight = halfHeight
	c.halfWidth = halfWidth
	c.invWidth2 = 2 / float64(c.Width)
	c.invHeight2 = 2 / float64(c.Height)
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

	var (
		row, col = index[0], index[1]
		u        = (float64(row)+rand.Float64())*c.invWidth2 - 1
		v        = (float64(col)+rand.Float64())*c.invHeight2 - 1
	)

	res.Origin.CloneFromVec(c.Position)
	res.Direction.CloneFromVec(c.dir)
	res.Direction.AddScaledVec(res.Direction, u*c.halfWidth, c.right)
	res.Direction.AddScaledVec(res.Direction, -v*c.halfHeight, c.up)
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

	fromCamera := mat.NewVecDense(3, nil)
	fromCamera.SubVec(point, c.Position)
	forwardDistance := mat.Dot(fromCamera, c.dir)
	if forwardDistance <= 0 {
		return FilmProjection{}, false
	}

	distance := mat.Norm(fromCamera, 2)
	if distance <= 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return FilmProjection{}, false
	}
	x := mat.Dot(fromCamera, c.right) / forwardDistance
	y := mat.Dot(fromCamera, c.up) / forwardDistance
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
		X: (u+1)*0.5*float64(c.Width) - 0.5,
		Y: (v+1)*0.5*float64(c.Height) - 0.5,
	}
	if _, ok := raster.PixelIndex(c.Width, c.Height); !ok {
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
		Jacobian: float64(c.Width*c.Height) / denominator,
	}, true
}
