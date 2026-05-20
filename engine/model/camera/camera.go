package camera

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
)

type Camera interface {
	GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray // Generates a ray for a given pixel index.
}

type CameraBase struct {
}

func (c *CameraBase) GenerateRay(ray *renderray.Ray, index ...int) *renderray.Ray {
	return &renderray.Ray{}
}
