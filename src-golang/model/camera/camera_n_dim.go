package camera

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"src-golang/math_lib"
	"src-golang/model/optics"
)

type CameraNDim struct {
	CameraBase
	Position    *mat.VecDense   // 相机位置
	Coordinates []*mat.VecDense // 坐标系
	Width       []int64         // 像素宽度
	FieldOfView []float64       // 视野角度 (度)
	Ortho       bool            // 正交相机 / 透视相机
}

// NewCamera 创建新相机
func NewCameraNDim() *CameraNDim {
	return &CameraNDim{}
}

func (c *CameraNDim) GenerateRay(res *optics.Ray, x ...int) *optics.Ray {
	if res == nil {
		res = &optics.Ray{}
	}
	res.Init()

	var (
		dim         = len(x)
		u           = make([]float64, dim)
		coordinates = math_lib.GramSchmidt(c.Coordinates...) // 正交化所有向量，确保它们互相垂直
	)

	for i := 0; i < dim; i++ {
		u[i] = 2*(float64(x[i])+rand.Float64())/float64(c.Width[0]) - 1
	}

	res.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	res.Origin.CloneFromVec(c.Position)
	res.Direction = mat.NewVecDense(dim, nil)
	for i := 1; i < dim; i++ {
		res.Direction.AddScaledVec(res.Direction, u[i]*math.Tan(c.FieldOfView[i]*math.Pi/180/2.0), coordinates[i])
	}
	math_lib.Normalize(res.Direction)

	return res
}
