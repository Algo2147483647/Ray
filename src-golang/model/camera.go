package model

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
)

// Camera 表示场景中的相机
type Camera struct {
	Position    *mat.VecDense // 相机位置
	Direction   *mat.VecDense // 观察方向
	Up          *mat.VecDense // 上方向向量
	FieldOfView float64       // 视野角度(度)
	AspectRatio float64       // 宽高比
}

// NewCamera 创建新相机
func NewCamera() *Camera {
	return &Camera{
		Position:  mat.NewVecDense(3, []float64{0, 0, 0}),
		Direction: mat.NewVecDense(3, []float64{1, 0, 0}),
		Up:        mat.NewVecDense(3, []float64{0, 0, 1}),
	}
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

// 向量叉乘函数
func cross(a, b *mat.VecDense) *mat.VecDense {
	if a.Len() != 3 || b.Len() != 3 {
		panic("叉乘要求向量必须是三维")
	}
	result := mat.NewVecDense(3, []float64{
		a.AtVec(1)*b.AtVec(2) - a.AtVec(2)*b.AtVec(1),
		a.AtVec(2)*b.AtVec(0) - a.AtVec(0)*b.AtVec(2),
		a.AtVec(0)*b.AtVec(1) - a.AtVec(1)*b.AtVec(0),
	})
	return result
}

// 向量归一化函数
func normalize(v *mat.VecDense) *mat.VecDense {
	norm := mat.Norm(v, 2)
	if norm == 0 {
		return v
	}
	result := mat.NewVecDense(v.Len(), nil)
	result.ScaleVec(1/norm, v)
	return result
}

// GetRays 生成像素光线
func (c *Camera) GetRays(width, height int, rng *rand.Rand) []*Ray {
	rays := make([]*Ray, 0, width*height)

	// 计算右向量: 方向 × 上向量
	right := cross(c.Direction, c.Up)
	right = normalize(right)

	// 计算实际上向量: 右向量 × 方向
	cameraUp := cross(right, c.Direction)
	cameraUp = normalize(cameraUp)

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
			randX := rng.Float64()
			randY := rng.Float64()

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
			rayDir = normalize(rayDir)

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
	right := cross(c.Direction, c.Up)
	right = normalize(right)
	cameraUp := cross(right, c.Direction)
	cameraUp = normalize(cameraUp)

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
