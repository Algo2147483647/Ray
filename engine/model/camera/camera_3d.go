package camera

import (
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
)

type Camera3D struct {
	CameraBase
	Position    *mat.VecDense // Camera position
	Direction   *mat.VecDense // Viewing direction
	Up          *mat.VecDense // Up vector
	Width       int           // Film width (pixels)
	Height      int           // Film height (pixels)
	FieldOfView float64       // Field of view angle (degrees)
	AspectRatio float64       // Aspect ratio
	Ortho       bool          // Orthographic camera / perspective camera
}

func NewCamera3D() *Camera3D {
	return &Camera3D{}
}

func (c *Camera3D) GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray {
	if res == nil {
		res = &renderray.Ray{}
	}
	res.Init()

	var (
		row, col   = index[0], index[1]
		dir        = c.Direction
		up         = c.Up
		right      = math_lib.Normalize(math_lib.Cross2(dir, up))          // Compute the right vector from direction and up.
		u          = 2*(float64(row)+rand.Float64())/float64(c.Width) - 1  // [-1, 1]
		v          = 2*(float64(col)+rand.Float64())/float64(c.Height) - 1 // [-1, 1]
		fovRad     = c.FieldOfView * math.Pi / 180
		halfHeight = math.Tan(fovRad / 2)
		halfWidth  = c.AspectRatio * halfHeight
	)

	res.Origin.CloneFromVec(c.Position)
	res.Direction.CloneFromVec(dir)
	res.Direction.AddScaledVec(res.Direction, u*halfWidth, right)
	res.Direction.AddScaledVec(res.Direction, -v*halfHeight, up)
	math_lib.Normalize(res.Direction)

	return res
}

// SetLookAt sets the camera target.
func (c *Camera3D) SetLookAt(lookAt *mat.VecDense) *Camera3D {
	c.Direction = mat.NewVecDense(lookAt.Len(), nil)
	c.Direction.SubVec(lookAt, c.Position)
	math_lib.Normalize(c.Direction)
	return c
}
