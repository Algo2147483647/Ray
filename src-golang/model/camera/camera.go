package camera

import "src-golang/model/optics"

type Camera interface {
	GenerateRay(res *optics.Ray, index ...int64) *optics.Ray
}

type CameraBase struct {
}

func (c *CameraBase) GenerateRay(ray *optics.Ray, index ...int) *optics.Ray {
	return &optics.Ray{}
}
