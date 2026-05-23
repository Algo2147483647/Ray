package camera

import (
	"fmt"
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
)

type Camera3D struct {
	CameraBase
	Position    *mat.VecDense // Camera origin in scene space.
	Direction   *mat.VecDense // Forward viewing direction.
	Up          *mat.VecDense // Up vector defining camera roll.
	Width       int           // Film width in pixels.
	Height      int           // Film height in pixels.
	FieldOfView float64       // Field-of-view angle in degrees.
	AspectRatio float64       // Film width-to-height ratio.
	Ortho       bool          // Uses orthographic projection when true.
	dir         *mat.VecDense // Normalized viewing direction.
	up          *mat.VecDense // Normalized camera up vector.
	right       *mat.VecDense // Normalized camera right vector.
	halfWidth   float64       // Half-width of the view plane.
	halfHeight  float64       // Half-height of the view plane.
	invWidth2   float64       // Reciprocal of twice the film width.
	invHeight2  float64       // Reciprocal of twice the film height.
	prepared    bool          // Indicates cached camera basis is ready.
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
	} else if c.FieldOfView <= 0 {
		return fmt.Errorf("camera field of view must be > 0")
	} else if c.AspectRatio <= 0 {
		return fmt.Errorf("camera aspect ratio must be > 0")
	} else if mat.Norm(c.Direction, 2) == 0 {
		return fmt.Errorf("camera direction must not be zero")
	} else if mat.Norm(c.Up, 2) == 0 {
		return fmt.Errorf("camera up vector must not be zero")
	}

	c.dir = mat.VecDenseCopyOf(c.Direction)
	math_lib.Normalize(c.dir)
	c.up = mat.VecDenseCopyOf(c.Up)
	math_lib.Normalize(c.up)
	right := math_lib.Cross2(c.dir, c.up)
	if mat.Norm(right, 2) == 0 {
		return fmt.Errorf("camera direction and up vector must not be parallel")
	}
	c.right = math_lib.Normalize(right)

	fovRad := c.FieldOfView * math.Pi / 180
	c.halfHeight = math.Tan(fovRad / 2)
	c.halfWidth = c.AspectRatio * c.halfHeight
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
	math_lib.Normalize(res.Direction)

	return res
}

// SetLookAt sets the camera target.
func (c *Camera3D) SetLookAt(lookAt *mat.VecDense) *Camera3D {
	c.Direction = mat.NewVecDense(lookAt.Len(), nil)
	c.Direction.SubVec(lookAt, c.Position)
	math_lib.Normalize(c.Direction)
	c.prepared = false
	return c
}
