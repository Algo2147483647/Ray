package camera

import (
	"fmt"
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand/v2"
	"src-golang/model/optics"
)

type CameraNDim struct {
	CameraBase
	Position    *mat.VecDense
	Coordinates []*mat.VecDense
	Width       []int
	FieldOfView []float64
	Ortho       bool

	orthonormalCoordinates []*mat.VecDense
	fovTangents            []float64
}

func NewCameraNDim() *CameraNDim {
	return &CameraNDim{}
}

func (c *CameraNDim) Prepare() error {
	if len(c.Coordinates) == 0 {
		return fmt.Errorf("camera coordinates are not configured")
	}
	if len(c.Width) == 0 {
		return fmt.Errorf("camera width is not configured")
	}
	if len(c.Width) != len(c.FieldOfView) {
		return fmt.Errorf("width count %d does not match field of view count %d", len(c.Width), len(c.FieldOfView))
	}
	if len(c.Coordinates) != len(c.Width)+1 {
		return fmt.Errorf("coordinate count %d must equal width count + 1 (%d)", len(c.Coordinates), len(c.Width)+1)
	}

	c.orthonormalCoordinates = math_lib.GramSchmidt(c.Coordinates...)
	c.fovTangents = make([]float64, len(c.FieldOfView))
	for i, fov := range c.FieldOfView {
		c.fovTangents[i] = math.Tan(fov * math.Pi / 180 / 2.0)
	}

	return nil
}

func (c *CameraNDim) ensurePrepared() error {
	if len(c.orthonormalCoordinates) == len(c.Coordinates) && len(c.fovTangents) == len(c.FieldOfView) {
		return nil
	}

	return c.Prepare()
}

func (c *CameraNDim) GenerateRay(res *optics.Ray, x ...int) *optics.Ray {
	if res == nil {
		res = &optics.Ray{}
	}
	res.Init()

	if err := c.ensurePrepared(); err != nil {
		panic(err)
	}

	u := make([]float64, len(x))
	for i := 0; i < len(x); i++ {
		u[i] = 2*(float64(x[i])+rand.Float64())/float64(c.Width[i]) - 1
	}

	res.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	res.Origin.CloneFromVec(c.Position)
	res.Direction.CloneFromVec(c.orthonormalCoordinates[0])
	for i := 0; i < len(x); i++ {
		res.Direction.AddScaledVec(res.Direction, u[i]*c.fovTangents[i], c.orthonormalCoordinates[i+1])
	}
	math_lib.Normalize(res.Direction)

	return res
}
