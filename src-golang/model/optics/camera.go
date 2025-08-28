package optics

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"src-golang/math_lib"
)

// Camera 表示场景中的相机
type Camera struct {
	Position    *mat.VecDense // 相机位置
	Direction   *mat.VecDense // 观察方向
	Up          *mat.VecDense // 上方向向量
	Width       int           // 胶片宽 (像素)
	Height      int           // 胶片高 (像素)
	FieldOfView float64       // 视野角度 (度)
	AspectRatio float64       // 宽高比
	Ortho       bool          // 正交相机 / 透视相机
}

// NewCamera 创建新相机
func NewCamera() *Camera {
	return &Camera{}
}

// SetLookAt 设置相机观察目标
func (c *Camera) SetLookAt(lookAt *mat.VecDense) *Camera {
	c.Direction = mat.NewVecDense(lookAt.Len(), nil)
	c.Direction.SubVec(lookAt, c.Position)
	math_lib.Normalize(c.Direction)
	return c
}

func (c *Camera) GenerateRay(ray *Ray, row, col int) *Ray {
	if ray == nil {
		ray = &Ray{}
	}
	ray.Init()

	var (
		dir        = c.Direction
		up         = c.Up
		right      = math_lib.Normalize(math_lib.Cross2(dir, up))          // 计算右向量和上向量
		u          = 2*(float64(row)+rand.Float64())/float64(c.Width) - 1  // [-1, 1]
		v          = 2*(float64(col)+rand.Float64())/float64(c.Height) - 1 // [-1, 1]
		fovRad     = c.FieldOfView * math.Pi / 180
		halfHeight = math.Tan(fovRad / 2)
		halfWidth  = c.AspectRatio * halfHeight
	)

	ray.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	ray.Origin.CloneFromVec(c.Position)
	ray.Direction.CloneFromVec(dir)
	ray.Direction.AddScaledVec(ray.Direction, u*halfWidth, right)
	ray.Direction.AddScaledVec(ray.Direction, -v*halfHeight, up)
	math_lib.Normalize(ray.Direction)
	return ray
}
