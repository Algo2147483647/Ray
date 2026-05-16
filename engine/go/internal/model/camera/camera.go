package camera

import "github.com/Algo2147483647/ray/engine/go/internal/model/optics"

type Camera interface {
	GenerateRay(res *optics.Ray, index ...int) *optics.Ray
}

type CameraBase struct {
}

func (c *CameraBase) GenerateRay(ray *optics.Ray, index ...int) *optics.Ray {
	return &optics.Ray{}
}
