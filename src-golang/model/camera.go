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

func (c *Camera) GenerateRay(Ray *ray.Ray, row, col int) *ray.Ray {
	if Ray == nil {
		Ray = &ray.Ray{}
	}

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

	randX := rand.Float64() // 添加随机偏移(抗锯齿)
	randY := rand.Float64()
	u := (2*((float64(row)+randX)/float64(c.Width)) - 1) * imageWidth / 2 // 计算标准化设备坐标
	v := 1 - 2*((float64(col)+randY)/float64(c.Height))*imageHeight/2
	u *= imageWidth / 2 // 缩放坐标到成像平面
	v *= imageHeight / 2

	// 计算光线方向
	rayDir := mat.NewVecDense(3, nil)
	rayDir.AddScaledVec(rayDir, u, right)
	rayDir.AddScaledVec(rayDir, v, cameraUp)
	rayDir.AddScaledVec(rayDir, dirScale, c.Direction)
	rayDir = math_lib.Normalize(rayDir)

	Ray.Origin = c.Position
	Ray.Direction = rayDir

	// println(utils.FormatVec(Ray.Origin), utils.FormatVec(Ray.Direction))
	return Ray
}

// GenerateRays 生成像素光线
func (c *Camera) GenerateRays(width, height int) []*ray.Ray {
	rays := make([]*ray.Ray, 0, width*height)

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
			randX := rand.Float64()
			randY := rand.Float64()

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

			rays = append(rays, &ray.Ray{
				Origin:    c.Position,
				Direction: rayDir,
			})
		}
	}
	return rays
}

// GetRayCoordinates 获取光线对应的像素坐标
func (c *Camera) GetRayCoordinates(ray *ray.Ray, width, height int) (int, int) {
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
