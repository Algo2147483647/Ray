package shape

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
	"sync"
)

// 全局临时向量池
var (
	edgePool = sync.Pool{
		New: func() interface{} {
			return [2]*mat.VecDense{
				mat.NewVecDense(3, nil),
				mat.NewVecDense(3, nil),
			}
		},
	}
	tmpPool = sync.Pool{
		New: func() interface{} {
			return mat.NewVecDense(3, nil)
		},
	}
)

type Triangle struct {
	BaseShape
	P1 *mat.VecDense `json:"p1"`
	P2 *mat.VecDense `json:"p2"`
	P3 *mat.VecDense `json:"p3"`
}

func NewTriangle(P1, P2, P3 *mat.VecDense) *Triangle {
	return &Triangle{
		P1: P1,
		P2: P2,
		P3: P3,
	}
}

func (t *Triangle) Name() string {
	return "Triangle"
}

func (t *Triangle) Intersect(raySt, rayDir *mat.VecDense) float64 {
	edges := edgePool.Get().([2]*mat.VecDense)
	tmp := tmpPool.Get().(*mat.VecDense)
	defer func() {
		edgePool.Put(edges)
		tmpPool.Put(tmp)
	}()

	edges[0].SubVec(t.P2, t.P1)           // E1 = P2 - P1
	edges[1].SubVec(t.P3, t.P1)           // E2 = P3 - P1
	p := math_lib.Cross(rayDir, edges[1]) // 计算法向量和行列式 (P = D × E2)
	a := mat.Dot(edges[0], p)             // a = E1·P
	if a > 0 {                            // 处理背面剔除
		tmp.SubVec(raySt, t.P1) // T = O - P1
	} else {
		tmp.SubVec(t.P1, raySt) // T = P1 - O
		a = -a
	}
	if a < math_lib.EPS { // 检查平行
		return math.MaxFloat64
	}

	q := math_lib.Cross(tmp, edges[0]) // Q = T × E1
	u := mat.Dot(tmp, p) / a           // 重心坐标 u
	v := mat.Dot(rayDir, q) / a        // 重心坐标 v
	if u < 0 || u > 1 {
		return math.MaxFloat64
	}
	if v < 0 || u+v > 1 {
		return math.MaxFloat64
	}
	return mat.Dot(edges[1], q) / a
}

func (t *Triangle) GetNormalVector(intersect *mat.VecDense) *mat.VecDense {
	edge1 := mat.NewVecDense(3, nil)
	edge2 := mat.NewVecDense(3, nil)

	edge1.SubVec(t.P2, t.P1)
	edge2.SubVec(t.P3, t.P1)
	normal := math_lib.Cross(edge1, edge2)

	// 归一化法向量
	norm := mat.Norm(normal, 2)
	if norm > 0 {
		normal.ScaleVec(1/norm, normal)
	}
	return normal
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
