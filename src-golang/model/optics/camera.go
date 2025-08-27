package optics

type Camera interface {
	GenerateRay(res *Ray, index []int64) *Ray
}

type CameraBase struct {
}

func (c *CameraBase) GenerateRay(ray *Ray, index []int64) *Ray {
	return &Ray{}
}
