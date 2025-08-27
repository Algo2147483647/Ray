package optics

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"src-golang/math_lib"
)

type Camera4D struct {
	CameraBase
	Position    *mat.VecDense   // 相机位置
	Coordinates []*mat.VecDense // 坐标系
	Width       []int64         // 像素宽度
	FieldOfView []float64       // 视野角度 (度)
	Ortho       bool            // 正交相机 / 透视相机
}

// NewCamera 创建新相机
func NewCamera4D() *Camera4D {
	return &Camera4D{}
}

func (c *Camera4D) GenerateRay(res *Ray, index []int64) *Ray {
	if res == nil {
		res = &Ray{}
	}
	res.Init()

	var (
		x, y, z = index[0], index[1], index[2]
		u       = 2*(float64(x)+rand.Float64())/float64(c.Width[0]) - 1 // [-1, 1]
		v       = 2*(float64(y)+rand.Float64())/float64(c.Width[1]) - 1 // [-1, 1]
		w       = 2*(float64(z)+rand.Float64())/float64(c.Width[2]) - 1 // [-1, 1]
	)

	// 四维情况，需要三个基向量来定义视图空间, 创建一个临时的第3个向量（例如 [0,0,0,1]）, 在四维空间中，我们需要两个"向上"的向量来定义视图空间, 第一个向上向量 (Up) 和观察方向 (Direction) 定义了第一个平面, 我们需要计算另一个与这两个向量正交的向量来定义四维空间中的视图方向
	tempVec := mat.NewVecDense(4, []float64{0, 0, 0, 1})
	third := math_lib.Normalize(math_lib.Cross4(c.Coordinates[0], c.Coordinates[2], tempVec)) // 计算第3个基向量，使其与dir和up正交
	right := math_lib.Normalize(math_lib.Cross4(c.Coordinates[0], c.Coordinates[2], third))   // 计算右向量（与dir和up正交）

	// 正交化所有向量，确保它们互相垂直
	vec := math_lib.GramSchmidt(
		mat.VecDenseCopyOf(c.Coordinates[0]),
		mat.VecDenseCopyOf(c.Coordinates[2]),
		mat.VecDenseCopyOf(third),
		mat.VecDenseCopyOf(right))
	_, upOrtho, _, rightOrtho := vec[0], vec[1], vec[2], vec[3]
	right = rightOrtho // 使用正交后的右向量和上向量构建视图方向

	res.Color = mat.NewVecDense(3, []float64{1, 1, 1})
	res.Origin.CloneFromVec(c.Position)
	res.Direction.CloneFromVec(c.Coordinates[0])
	res.Direction.AddScaledVec(res.Direction, u*math.Tan(c.FieldOfView[0]*math.Pi/180/2), right) // 使用正交后的右向量和上向量构建视图方向
	res.Direction.AddScaledVec(res.Direction, -v*math.Tan(c.FieldOfView[1]*math.Pi/180/2), upOrtho)
	res.Direction.AddScaledVec(res.Direction, w*math.Tan(c.FieldOfView[2]*math.Pi/180/2), upOrtho)
	math_lib.Normalize(res.Direction)

	return res
}
