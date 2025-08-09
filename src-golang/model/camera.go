package model

import (
	"gonum.org/v1/gonum/spatial/r3"
	"math"
	"math/rand"
)

// Camera 表示场景中的相机
type Camera struct {
	Position    r3.Vec  // 相机位置
	Direction   r3.Vec  // 观察方向
	Up          r3.Vec  // 上方向向量
	FieldOfView float64 // 视野角度(度)
	AspectRatio float64 // 宽高比
}

// NewCamera 创建新相机
func NewCamera() *Camera {
	return &Camera{
		Up: r3.Vec{Z: 1}, // 默认Z轴向上
	}
}

// SetLookAt 设置相机观察目标
func (c *Camera) SetLookAt(lookAt r3.Vec) *Camera {
	c.Direction = r3.Unit(r3.Sub(lookAt, c.Position))
	return c
}

// GetRays 生成像素光线
func (c *Camera) GetRays(width, height int, rng *rand.Rand) []*Ray {
	rays := make([]*Ray, 0, width*height)
	right := r3.Unit(r3.Cross(c.Direction, c.Up))
	cameraUp := r3.Unit(r3.Cross(right, c.Direction))

	// 计算成像平面尺寸
	fovRad := c.FieldOfView * math.Pi / 180
	imageHeight := 2 * math.Tan(fovRad/2)
	imageWidth := imageHeight * c.AspectRatio

	for j := 0; j < height; j++ {
		for i := 0; i < width; i++ {
			// 添加随机偏移(抗锯齿)
			randX := rng.Float64()
			randY := rng.Float64()

			// 计算标准化设备坐标
			u := (2*((float64(i)+randX)/float64(width)) - 1) * imageWidth / 2
			v := (1 - 2*((float64(j)+randY)/float64(height))) * imageHeight / 2

			// 计算光线方向
			rayDir := r3.Add(
				c.Direction,
				r3.Add(r3.Scale(u, right),
					r3.Scale(v, cameraUp)),
			)
			rays = append(rays, &Ray{
				Origin:    c.Position,
				Direction: r3.Unit(rayDir),
			})
		}
	}
	return rays
}

// GetRayCoordinates 获取光线对应的像素坐标
func (c *Camera) GetRayCoordinates(ray *Ray, width, height int) (int, int) {
	right := r3.Unit(r3.Cross(c.Direction, c.Up))
	cameraUp := r3.Unit(r3.Cross(right, c.Direction))

	fovRad := c.FieldOfView * math.Pi / 180
	imageHeight := 2 * math.Tan(fovRad/2)
	imageWidth := imageHeight * c.AspectRatio

	dir := r3.Unit(ray.Direction)
	u := r3.Dot(dir, right) / (imageWidth / 2)
	v := r3.Dot(dir, cameraUp) / (imageHeight / 2)

	x := int((u + 1) * float64(width) / 2)
	y := int((1 - v) * float64(height) / 2)

	// 确保坐标在有效范围内
	if x < 0 {
		x = 0
	} else if x >= width {
		x = width - 1
	}
	if y < 0 {
		y = 0
	} else if y >= height {
		y = height - 1
	}
	return x, y
}
