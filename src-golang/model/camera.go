package model

import (
	"bytes"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"os"
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

	// 归一化方向向量（确保计算基础正确）
	dir := math_lib.Normalize(c.Direction)
	// 计算右向量和上向量
	right := math_lib.Normalize(math_lib.Cross(dir, c.Up))
	cameraUp := math_lib.Normalize(math_lib.Cross(right, dir))

	// 计算成像平面尺寸
	fovRad := c.FieldOfView * math.Pi / 180
	halfHeight := math.Tan(fovRad / 2)
	halfWidth := c.AspectRatio * halfHeight

	// 添加随机偏移（抗锯齿）
	randX := rand.Float64()
	randY := rand.Float64()
	// 计算成像平面坐标（修正冗余缩放）
	u := 2*(float64(row)+randX)/float64(c.Width) - 1  // [-1, 1]
	v := 1 - 2*(float64(col)+randY)/float64(c.Height) // [-1, 1]（翻转Y轴）
	u *= halfWidth
	v *= halfHeight

	if c.Ortho { // 正交相机：起点在成像平面
		Ray.Origin = mat.NewVecDense(3, nil)
		Ray.Origin.AddScaledVec(c.Position, u, right)
		Ray.Origin.AddScaledVec(Ray.Origin, v, cameraUp)
		Ray.Direction = dir // 方向固定为观察方向
	} else { // 透视相机：起点为相机位置
		Ray.Origin = c.Position
		rayDir := mat.NewVecDense(3, nil) // 方向 = 成像平面点 - 原点（即 u*right + v*cameraUp + 前向）
		rayDir.AddScaledVec(rayDir, u, right)
		rayDir.AddScaledVec(rayDir, v, cameraUp)
		rayDir.AddScaledVec(rayDir, 1, dir) // dir已归一化
		Ray.Direction = math_lib.Normalize(rayDir)
	}
	return Ray
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

func (c *Camera) GenerateRaysSVG() string {
	const (
		svgWidth  = 1000  // SVG画布宽度
		svgHeight = 1000  // SVG画布高度
		step      = 50    // 像素采样步长(每10个像素采样一次)
		maxRays   = 10000 // 最大光线数量限制(防止过多)
	)

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">`, svgWidth, svgHeight))
	buf.WriteString(`<rect width="100%" height="100%" fill="black"/>`)

	// 计算相机坐标系基向量
	dir := math_lib.Normalize(c.Direction)
	right := math_lib.Normalize(math_lib.Cross(dir, c.Up))
	up := math_lib.Normalize(math_lib.Cross(right, dir))
	rayCount := 0
	Ray := &ray.Ray{}

	for y := 0; y < c.Height; y += step {
		for x := 0; x < c.Width; x += step {
			if rayCount > maxRays {
				break // 防止生成过多光线
			}
			rayCount++

			// 计算终点位置 (起点 + 缩放后的方向)
			c.GenerateRay(Ray, x, y)
			end := mat.NewVecDense(3, nil)
			end.AddVec(c.Position, Ray.Direction)

			// 转换到SVG坐标系 (简单正交投影)
			startX := svgWidth/2 + 100*mat.Dot(c.Position, right)
			startY := svgHeight/2 - 100*mat.Dot(c.Position, up)
			endX := svgWidth/2 + 100*mat.Dot(end, right)
			endY := svgHeight/2 - 100*mat.Dot(end, up)

			// 添加SVG线段
			color := fmt.Sprintf("hsl(%d, 100%%, 70%%)", x*360/c.Width) // 基于X位置的色相
			buf.WriteString(fmt.Sprintf(
				`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="%s" stroke-width="0.5" opacity="0.7"/>`,
				startX, startY, endX, endY, color,
			))
		}
	}

	// 添加相机位置标记
	buf.WriteString(fmt.Sprintf(
		`<circle cx="%.2f" cy="%.2f" r="5" fill="red"/>`,
		svgWidth/2+100*mat.Dot(c.Position, right),
		svgHeight/2-100*mat.Dot(c.Position, up),
	))

	buf.WriteString("</svg>")

	file, err := os.Create("rays.svg")
	if err != nil {
		buf.String()
	}
	defer file.Close()

	file.WriteString(buf.String())
	return buf.String()
}
