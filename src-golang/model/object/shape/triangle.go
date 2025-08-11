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
	P1, P2, P3 *mat.VecDense
}

func (t *Triangle) GetName() string {
	return "Triangle"
}

func (t *Triangle) Intersect(raySt, rayDir *mat.VecDense) float64 {
	// 从线程池获取资源
	edges := edgePool.Get().([2]*mat.VecDense)
	tmp := tmpPool.Get().(*mat.VecDense)
	defer func() {
		edgePool.Put(edges)
		tmpPool.Put(tmp)
	}()

	// 计算三角形边向量 (E1 = P2-P1, E2 = P3-P1)
	edges[0].SubVec(t.P2, t.P1) // E1 = P2 - P1
	edges[1].SubVec(t.P3, t.P1) // E2 = P3 - P1

	// 计算法向量和行列式 (P = D × E2)
	p := math_lib.Cross(rayDir, edges[1])
	a := mat.Dot(edges[0], p) // a = E1·P

	// 处理背面剔除
	if a > 0 {
		tmp.SubVec(raySt, t.P1) // T = O - P1
	} else {
		tmp.SubVec(t.P1, raySt) // T = P1 - O
		a = -a
	}

	// 检查平行
	if a < math_lib.EPS {
		return math.MaxFloat64
	}

	// 计算重心坐标 u
	u := mat.Dot(tmp, p) / a
	if u < 0 || u > 1 {
		return math.MaxFloat64
	}

	// 计算 Q = T × E1
	q := math_lib.Cross(tmp, edges[0])

	// 计算重心坐标 v
	v := mat.Dot(rayDir, q) / a
	if v < 0 || u+v > 1 {
		return math.MaxFloat64
	}

	// 计算交点参数 t
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

func (t *Triangle) BuildBoundingBox() (min, max *mat.VecDense) {
	minData := make([]float64, 3)
	maxData := make([]float64, 3)

	for i := 0; i < 3; i++ {
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
		minData[i] = minVal
		maxData[i] = maxVal
	}

	return mat.NewVecDense(3, minData), mat.NewVecDense(3, maxData)
}
