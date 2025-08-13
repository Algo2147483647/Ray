package model

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"src-golang/math_lib"
	"src-golang/model/ray"
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
	c.Direction = mat.NewVecDense(3, nil)
	c.Direction.SubVec(lookAt, c.Position)
	math_lib.Normalize(c.Direction)
	return c
}

func (c *Camera) GenerateRay(Ray *ray.Ray, row, col int) *ray.Ray {
	if Ray == nil {
		Ray = &ray.Ray{}
	}
	if Ray.Origin == nil {
		Ray.Origin = mat.NewVecDense(3, nil)
	}
	if Ray.Direction == nil {
		Ray.Direction = mat.NewVecDense(3, nil)
	}

	dir := c.Direction
	up := c.Up
	right := math_lib.Normalize(math_lib.Cross(dir, up)) // 计算右向量和上向量

	fovRad := c.FieldOfView * math.Pi / 180
	halfHeight := math.Tan(fovRad / 2)
	halfWidth := c.AspectRatio * halfHeight

	u := 2*(float64(row)+rand.Float64())/float64(c.Width) - 1  // [-1, 1]
	v := 2*(float64(col)+rand.Float64())/float64(c.Height) - 1 // [-1, 1]
	u *= halfWidth
	v *= -halfHeight //（翻转Y轴）

	Ray.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	Ray.Origin.CloneFromVec(c.Position)
	Ray.Direction.CloneFromVec(dir)
	Ray.Direction.AddScaledVec(Ray.Direction, u, right)
	Ray.Direction.AddScaledVec(Ray.Direction, v, up)
	math_lib.Normalize(Ray.Direction)

	return Ray
}
