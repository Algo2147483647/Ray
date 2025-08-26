package optics

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"src-golang/math_lib"
	"src-golang/utils"
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

func (c *Camera) GenerateRay4D(ray *Ray, row, col int) *Ray {
	if ray == nil {
		ray = &Ray{}
	}
	ray.Init()

	var (
		dir = c.Direction
		up  = c.Up
	)

	switch utils.Dimension { // 根据空间维度选择不同的处理方式
	case 3:
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

	case 4:
		// 四维情况，需要三个基向量来定义视图空间, 创建一个临时的第3个向量（例如 [0,0,0,1]）
		// 在四维空间中，我们需要两个"向上"的向量来定义视图空间, 第一个向上向量 (Up) 和观察方向 (Direction) 定义了第一个平面, 我们需要计算另一个与这两个向量正交的向量来定义四维空间中的视图方向
		tempVec := mat.NewVecDense(4, []float64{0, 0, 0, 1})

		// 计算第3个基向量，使其与dir和up正交
		third := math_lib.Cross4(dir, up, tempVec)
		math_lib.Normalize(third)

		// 计算右向量（与dir和up正交）
		right := math_lib.Cross4(dir, up, third)
		math_lib.Normalize(right)

		// 正交化所有向量，确保它们互相垂直
		_, upOrtho, _, rightOrtho := math_lib.GramSchmidt4(
			mat.VecDenseCopyOf(dir),
			mat.VecDenseCopyOf(up),
			mat.VecDenseCopyOf(third),
			mat.VecDenseCopyOf(right))

		// 使用正交后的右向量和上向量构建视图方向
		right = rightOrtho

		u := 2*(float64(row)+rand.Float64())/float64(c.Width) - 1  // [-1, 1]
		v := 2*(float64(col)+rand.Float64())/float64(c.Height) - 1 // [-1, 1]
		fovRad := c.FieldOfView * math.Pi / 180
		halfHeight := math.Tan(fovRad / 2)
		halfWidth := c.AspectRatio * halfHeight

		ray.Color = mat.NewVecDense(3, []float64{1, 1, 1})
		ray.Origin.CloneFromVec(c.Position)
		ray.Direction.CloneFromVec(dir)
		// 使用正交后的右向量和上向量构建视图方向
		ray.Direction.AddScaledVec(ray.Direction, u*halfWidth, right)
		ray.Direction.AddScaledVec(ray.Direction, -v*halfHeight, upOrtho)
		math_lib.Normalize(ray.Direction)
	}

	return ray
}
