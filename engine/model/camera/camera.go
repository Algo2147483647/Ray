package camera

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
)

type Camera interface {
	GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray // Generates a ray for a given pixel index.
}

type CameraType string

const (
	CameraType3D         CameraType = "3d"
	CameraTypeNDim       CameraType = "n_dim"
	CameraTypeHyperbolic CameraType = "hyperbolic"
	CameraTypeSpherical  CameraType = "spherical"
)

type CameraBase struct {
}

func (c *CameraBase) GenerateRay(ray *renderray.Ray, index ...int) *renderray.Ray {
	return &renderray.Ray{}
}
