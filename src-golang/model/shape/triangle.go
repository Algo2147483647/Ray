package shape

import (
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/utils"
)

type Triangle struct {
	BaseShape
	P1  *mat.VecDense `json:"p1"`
	P2  *mat.VecDense `json:"p2"`
	P3  *mat.VecDense `json:"p3"`
	Mem TriangleCalculateStorage
}

type TriangleCalculateStorage struct {
	Edge1  *mat.VecDense
	Edge2  *mat.VecDense
	Normal *mat.VecDense
}

func NewTriangle(P1, P2, P3 *mat.VecDense) *Triangle {
	edge1 := mat.NewVecDense(P1.Len(), nil)
	edge2 := mat.NewVecDense(P1.Len(), nil)
	res := &Triangle{
		P1: P1,
		P2: P2,
		P3: P3,
		Mem: TriangleCalculateStorage{
			Edge1: math_lib.SubVec(edge1, P2, P1),
			Edge2: math_lib.SubVec(edge2, P3, P1),
		},
	}
	res.Mem.Normal = res.GetNormalVectorPure()

	return res
}

func (f *Triangle) Name() string {
	return "Triangle"
}

func (f *Triangle) Intersect(raySt, rayDir *mat.VecDense) float64 {
	distance := f.IntersectPure(raySt, rayDir)

	if f.EngravingFunc != nil && distance != math.MaxFloat64 {
		if f.EngravingFunc(map[string]interface{}{
			"ray_start": raySt,
			"ray_dir":   rayDir,
			"distance":  distance,
			"self":      f,
		}) {
			return math.MaxFloat64
		}
	}
	return distance
}

func (f *Triangle) IntersectPure(raySt, rayDir *mat.VecDense) float64 {
	t := utils.VectorPool.Get().(*mat.VecDense)
	p := utils.VectorPool.Get().(*mat.VecDense)
	q := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(t)
		utils.VectorPool.Put(p)
		utils.VectorPool.Put(q)
	}()

	math_lib.Cross(p, rayDir, f.Mem.Edge2) // 计算法向量和行列式 (P = D × E2)
	a := mat.Dot(f.Mem.Edge1, p)           // a = E1·P
	if a > 0 {                             // 处理背面剔除
		t.SubVec(raySt, f.P1) // T = O - P1
	} else {
		t.SubVec(f.P1, raySt) // T = P1 - O
		a = -a
	}
	if a < utils.EPS { // 检查平行
		return math.MaxFloat64
	}

	math_lib.Cross(q, t, f.Mem.Edge1) // Q = T × E1
	u := mat.Dot(t, p) / a            // 重心坐标 u
	v := mat.Dot(rayDir, q) / a       // 重心坐标 v
	if u < 0 || u > 1 {
		return math.MaxFloat64
	}
	if v < 0 || u+v > 1 {
		return math.MaxFloat64
	}
	return mat.Dot(f.Mem.Edge2, q) / a
}

func (f *Triangle) GetNormalVector(_, res *mat.VecDense) *mat.VecDense {
	res.CloneFromVec(f.Mem.Normal)
	return res
}

func (f *Triangle) GetNormalVectorPure() *mat.VecDense {
	edge1 := mat.NewVecDense(f.P1.Len(), nil)
	edge2 := mat.NewVecDense(f.P1.Len(), nil)
	return math_lib.Normalize(math_lib.Cross2(math_lib.SubVec(edge1, f.P2, f.P1), math_lib.SubVec(edge2, f.P3, f.P1)))
}

func (f *Triangle) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	pmin = mat.NewVecDense(f.P1.Len(), nil)
	pmax = mat.NewVecDense(f.P1.Len(), nil)

	for i := 0; i < f.P1.Len(); i++ {
		vals := []float64{f.P1.AtVec(i), f.P2.AtVec(i), f.P3.AtVec(i)}
		minVal, maxVal := vals[0], vals[0]
		for _, v := range vals[1:] {
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
		pmin.SetVec(i, minVal)
		pmax.SetVec(i, maxVal)
	}

	return
}
