package model

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
)

// Camera 表示场景中的相机
type Camera struct {
	Position    *mat.VecDense // 相机位置
	Direction   *mat.VecDense // 观察方向
	Up          *mat.VecDense // 上方向向量
	FieldOfView float64       // 视野角度(度)
	AspectRatio float64       // 宽高比
	Ortho       bool
	Width       int
	Height      int
	Aspect      float64
}

// NewCamera 创建新相机
func NewCamera() *Camera {
	return &Camera{}
}

// SetLookAt 设置相机观察目标
func (c *Camera) SetLookAt(lookAt *mat.VecDense) *Camera {
	// 计算方向向量: 目标位置 - 相机位置
	direction := mat.NewVecDense(3, nil)
	direction.SubVec(lookAt, c.Position)

	// 归一化方向向量
	norm := mat.Norm(direction, 2)
	if norm > 0 {
		direction.ScaleVec(1/norm, direction)
	}
	c.Direction = direction
	return c
}

func (c *Camera) GenerateRay(ray *Ray, row, col, rows, cols int) {
	// 计算相机坐标系基向量
	forward := new(mat.VecDense)
	forward.SubVec(c.Direction, c.Position)
	forward.ScaleVec(1/mat.Norm(forward, 2), forward)

	right := math_lib.Cross(forward, c.Up)
	right.ScaleVec(1/mat.Norm(right, 2), right)

	up := math_lib.Cross(right, forward)

	// 抗锯齿：在像素内随机采样
	var u, v float64
	if math_lib.Rnd != nil {
		u = (float64(col) + math_lib.Rnd.Float64()) / float64(cols)
		v = (float64(row) + math_lib.Rnd.Float64()) / float64(rows)
	} else {
		u = (float64(col) + 0.5) / float64(cols)
		v = (float64(row) + 0.5) / float64(rows)
	}

	// 将UV坐标映射到[-1,1]范围
	u = 2*u - 1
	v = 2*v - 1

	if c.Ortho {
		// 正交投影
		ray.Origin.Reset()
		ray.Origin.ScaleVec(u*float64(c.Width)/2, right)
		temp := new(mat.VecDense)
		temp.ScaleVec(v*float64(c.Width)/(2*c.Aspect), up)
		ray.Origin.AddVec(ray.Origin, temp)
		ray.Origin.AddVec(ray.Origin, c.Position)

		ray.Direction.CopyVec(forward)
	} else {
		//// 透视投影
		//tanFov := math.Tan(c.FOV * math.Pi / 360) // FOV/2 in radians
		//ray.Origin.CopyVec(c.Position)
		//
		//// 计算光线方向
		//ray.Direction.Reset()
		//ray.Direction.ScaleVec(u*tanFov, right)
		//temp := new(mat.VecDense)
		//temp.ScaleVec(v*tanFov/c.Aspect, up)
		//ray.Direction.AddVec(ray.Direction, temp)
		//ray.Direction.AddVec(ray.Direction, forward)
		//ray.Direction.ScaleVec(1/mat.Norm(ray.Direction, 2), ray.Direction)
	}

	// 重置光线颜色
	ray.Color.Reset()
	ray.Color.AddScaledVec(ray.Color, 1, mat.NewVecDense(3, []float64{1, 1, 1}))

}

// GenerateRays 生成像素光线
func (c *Camera) GenerateRays(width, height int) []*Ray {
	rays := make([]*Ray, 0, width*height)

	// 计算右向量: 方向 × 上向量, 实际上向量: 右向量 × 方向
	right := math_lib.Normalize(math_lib.Cross(c.Direction, c.Up))
	cameraUp := math_lib.Normalize(math_lib.Cross(right, c.Direction))

	// 计算成像平面尺寸
	fovRad := c.FieldOfView * math.Pi / 180
	imageHeight := 2 * math.Tan(fovRad/2)
	imageWidth := imageHeight * c.AspectRatio

	// 预计算方向分量
	dirScale := 1.0
	if c.Direction.Len() == 3 {
		dirScale = 1 / mat.Norm(c.Direction, 2)
	}

	for j := 0; j < height; j++ {
		for i := 0; i < width; i++ {
			// 添加随机偏移(抗锯齿)
			randX := math_lib.Rnd.Float64()
			randY := math_lib.Rnd.Float64()

			// 计算标准化设备坐标
			u := (2*((float64(i)+randX)/float64(width)) - 1) * imageWidth / 2
			v := 1 - 2*((float64(j)+randY)/float64(height))*imageHeight/2

			// 缩放坐标到成像平面
			u *= imageWidth / 2
			v *= imageHeight / 2

			// 计算光线方向
			rayDir := mat.NewVecDense(3, nil)
			rayDir.AddScaledVec(rayDir, u, right)
			rayDir.AddScaledVec(rayDir, v, cameraUp)
			rayDir.AddScaledVec(rayDir, dirScale, c.Direction)
			rayDir = math_lib.Normalize(rayDir)

			rays = append(rays, &Ray{
				Origin:    c.Position,
				Direction: rayDir,
			})
		}
	}
	return rays
}

// GetRayCoordinates 获取光线对应的像素坐标
func (c *Camera) GetRayCoordinates(ray *Ray, width, height int) (int, int) {
	// 计算右向量和上向量
	right := math_lib.Normalize(math_lib.Cross(c.Direction, c.Up))
	cameraUp := math_lib.Normalize(math_lib.Cross(right, c.Direction))

	// 计算成像平面尺寸
	fovRad := c.FieldOfView * math.Pi / 180
	imageHeight := 2 * math.Tan(fovRad/2)
	imageWidth := imageHeight * c.AspectRatio

	// 投影光线方向到成像平面
	dir := ray.Direction
	u := mat.Dot(dir, right) * (2 / imageWidth)
	v := mat.Dot(dir, cameraUp) * (2 / imageHeight)

	// 转换为像素坐标
	x := int((u + 1) * 0.5 * float64(width))
	y := int((1 - (v+1)*0.5) * float64(height))

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
