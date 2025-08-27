package optics

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"src-golang/math_lib"
	"src-golang/utils"
)

// Camera 表示场景中的相机
type Camera3D struct {
	CameraBase
	Position    *mat.VecDense // 相机位置
	Direction   *mat.VecDense // 观察方向
	Up          *mat.VecDense // 上方向向量
	Image       [3]*mat.Dense
	Width       int     // 胶片宽 (像素)
	Height      int     // 胶片高 (像素)
	FieldOfView float64 // 视野角度 (度)
	AspectRatio float64 // 宽高比
	Ortho       bool    // 正交相机 / 透视相机
}

func NewCamera3D() *Camera3D {
	return &Camera3D{}
}

// SetLookAt 设置相机观察目标
func (c *Camera3D) SetLookAt(lookAt *mat.VecDense) *Camera3D {
	c.Direction = mat.NewVecDense(lookAt.Len(), nil)
	c.Direction.SubVec(lookAt, c.Position)
	math_lib.Normalize(c.Direction)
	return c
}

func (c *Camera3D) InitImage(Width, Height int) {
	c.Width = Width
	c.Height = Height
	for i, _ := range c.Image {
		c.Image[i] = mat.NewDense(c.Width, c.Height, nil)
	}
}

func (c *Camera3D) LoadImage(filename string) {
	var err error
	c.Image, err = utils.LoadMatrices(filename)
	if err != nil {
		panic(err)
	}
}

func (c *Camera3D) SaveImage(filename string) {
	err := utils.SaveMatrices(filename, c.Image)
	if err != nil {
		panic(err)
	}
}

func (c *Camera3D) MergeImage(data [3]*mat.Dense, samples, samplesSt int64) {
	totalSamples := samples + samplesSt
	if samplesSt > 0 {
		for i := range c.Image { // 使用加权平均合并采样结果
			for x := 0; x < c.Width; x++ {
				for y := 0; y < c.Height; y++ {
					mergedValue := (c.Image[i].At(x, y)*float64(samplesSt) + data[i].At(x, y)*float64(samples)) / float64(totalSamples) // 加权平均: (oldValue * samplesSt + newValue * samples) / totalSamples
					c.Image[i].Set(x, y, mergedValue)
				}
			}
		}
	} else {
		for i := range c.Image {
			c.Image[i].Copy(data[i])
		}
	}
}

func (c *Camera3D) GenerateRay(res *Ray, index []int64) *Ray {
	if res == nil {
		res = &Ray{}
	}
	res.Init()

	var (
		row, col   = index[0], index[1]
		dir        = c.Direction
		up         = c.Up
		right      = math_lib.Normalize(math_lib.Cross2(dir, up))          // 计算右向量和上向量
		u          = 2*(float64(row)+rand.Float64())/float64(c.Width) - 1  // [-1, 1]
		v          = 2*(float64(col)+rand.Float64())/float64(c.Height) - 1 // [-1, 1]
		fovRad     = c.FieldOfView * math.Pi / 180
		halfHeight = math.Tan(fovRad / 2)
		halfWidth  = c.AspectRatio * halfHeight
	)

	res.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	res.Origin.CloneFromVec(c.Position)
	res.Direction.CloneFromVec(dir)
	res.Direction.AddScaledVec(res.Direction, u*halfWidth, right)
	res.Direction.AddScaledVec(res.Direction, -v*halfHeight, up)
	math_lib.Normalize(res.Direction)

	return res
}
