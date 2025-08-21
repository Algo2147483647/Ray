package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
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
	edge1 := new(mat.VecDense)
	edge2 := new(mat.VecDense)
	edge1.SubVec(P2, P1)
	edge2.SubVec(P3, P1)
	normal := TriangleGetNormalVector(edge1, edge2)

	return &Triangle{
		P1: P1,
		P2: P2,
		P3: P3,
		Mem: TriangleCalculateStorage{
			Edge1:  edge1,
			Edge2:  edge2,
			Normal: normal,
		},
	}
}

func (t *Triangle) Name() string {
	return "Triangle"
}

func (t *Triangle) Intersect(raySt, rayDir *mat.VecDense) float64 {
	tmp := utils.VectorPool.Get().(*mat.VecDense)
	p := utils.VectorPool.Get().(*mat.VecDense)
	q := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(tmp)
		utils.VectorPool.Put(p)
		utils.VectorPool.Put(q)
	}()

	math_lib.Cross(p, rayDir, t.Mem.Edge2) // 计算法向量和行列式 (P = D × E2)
	a := mat.Dot(t.Mem.Edge1, p)           // a = E1·P
	if a > 0 {                             // 处理背面剔除
		tmp.SubVec(raySt, t.P1) // T = O - P1
	} else {
		tmp.SubVec(t.P1, raySt) // T = P1 - O
		a = -a
	}
	if a < math_lib.EPS { // 检查平行
		return math.MaxFloat64
	}

	math_lib.Cross(q, tmp, t.Mem.Edge1) // Q = T × E1
	u := mat.Dot(tmp, p) / a            // 重心坐标 u
	v := mat.Dot(rayDir, q) / a         // 重心坐标 v
	if u < 0 || u > 1 {
		return math.MaxFloat64
	}
	if v < 0 || u+v > 1 {
		return math.MaxFloat64
	}
	return mat.Dot(t.Mem.Edge2, q) / a
}

func (t *Triangle) GetNormalVector(_, res *mat.VecDense) *mat.VecDense {
	res = t.Mem.Normal
	return res
}

func TriangleGetNormalVector(Edge1, Edge2 *mat.VecDense) *mat.VecDense {
	return math_lib.Normalize(math_lib.Cross2(Edge1, Edge2))
}

func (t *Triangle) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	pmin = mat.NewVecDense(3, make([]float64, 3))
	pmax = mat.NewVecDense(3, make([]float64, 3))

	for i := 0; i < t.P1.Len(); i++ {
		vals := []float64{t.P1.AtVec(i), t.P2.AtVec(i), t.P3.AtVec(i)}
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
