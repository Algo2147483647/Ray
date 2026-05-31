package camera

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/maths"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
)

type CameraNDim struct {
	CameraBase
	Position               *mat.VecDense   // Camera origin in N-dimensional space.
	Coordinates            []*mat.VecDense // Camera basis vectors.
	Width                  []int           // Film resolution per dimension.
	FieldOfView            []float64       // Field-of-view angle per dimension.
	Ortho                  bool            // Uses orthographic projection when true.
	orthonormalCoordinates []*mat.VecDense // Orthonormalized camera basis vectors.
	fovTangents            []float64       // Tangents of half field-of-view angles.
}

func NewCameraNDim() *CameraNDim {
	return &CameraNDim{}
}

func (c *CameraNDim) Prepare() error {
	// Avoid double counting
	if len(c.orthonormalCoordinates) == len(c.Coordinates) && len(c.fovTangents) == len(c.FieldOfView) {
		return nil
	}

	// Check configuration
	if len(c.Coordinates) == 0 {
		return fmt.Errorf("camera coordinates are not configured")
	} else if len(c.Width) == 0 {
		return fmt.Errorf("camera width is not configured")
	} else if len(c.Width) != len(c.FieldOfView) {
		return fmt.Errorf("width count %d does not match field of view count %d", len(c.Width), len(c.FieldOfView))
	} else if len(c.Coordinates) != len(c.Width)+1 {
		return fmt.Errorf("coordinate count %d must equal width count + 1 (%d)", len(c.Coordinates), len(c.Width)+1)
	}

	// Compute
	c.orthonormalCoordinates = maths.GramSchmidt(c.Coordinates...)
	c.fovTangents = make([]float64, len(c.FieldOfView))
	for i, fov := range c.FieldOfView {
		c.fovTangents[i] = math.Tan(fov * math.Pi / 180 / 2.0)
	}

	return nil
}

func (c *CameraNDim) GenerateRay(res *renderray.Ray, x ...int) *renderray.Ray {
	if res == nil {
		res = &renderray.Ray{}
	}
	res.Init()

	if err := c.Prepare(); err != nil {
		panic(err)
	}

	u := make([]float64, len(x))
	for i := 0; i < len(x); i++ {
		u[i] = 2*(float64(x[i])+rand.Float64())/float64(c.Width[i]) - 1
	}

	res.Origin.CloneFromVec(c.Position)
	res.Direction.CloneFromVec(c.orthonormalCoordinates[0])
	if c.Ortho {
		for i := 0; i < len(x); i++ {
			res.Origin.AddScaledVec(res.Origin, u[i]*c.fovTangents[i], c.orthonormalCoordinates[i+1])
		}
		maths.Normalize(res.Direction)
		return res
	}

	for i := 0; i < len(x); i++ {
		res.Direction.AddScaledVec(res.Direction, u[i]*c.fovTangents[i], c.orthonormalCoordinates[i+1])
	}
	maths.Normalize(res.Direction)

	return res
}
