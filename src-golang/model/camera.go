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
	Ray.Direction.Zero()
	Ray.Direction.AddScaledVec(Ray.Direction, u, right)
	Ray.Direction.AddScaledVec(Ray.Direction, v, up)
	Ray.Direction.AddScaledVec(Ray.Direction, 1, dir) // dir已归一化
	math_lib.Normalize(Ray.Direction)

	return Ray
}

func (c *Camera) DebugGenerateRaysSVG() string {
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
	dir := c.Direction
	up := c.Up
	right := math_lib.Normalize(math_lib.Cross(dir, c.Up))
	zero := mat.NewVecDense(3, nil)
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
			end.AddVec(zero, Ray.Direction)

			// 转换到SVG坐标系 (简单正交投影)
			startX := svgWidth/2 + 100*mat.Dot(zero, right)
			startY := svgHeight/2 - 100*mat.Dot(zero, up)
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
