package shape

import (
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
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
	interaction, ok := f.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (f *Triangle) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	t := utils.VectorPool.Get().(*mat.VecDense)
	p := utils.VectorPool.Get().(*mat.VecDense)
	q := utils.VectorPool.Get().(*mat.VecDense)
	defer func() {
		utils.VectorPool.Put(t)
		utils.VectorPool.Put(p)
		utils.VectorPool.Put(q)
	}()

	math_lib.Cross(p, rayDir, f.Mem.Edge2)
	a := mat.Dot(f.Mem.Edge1, p)
	if a > 0 {
		t.SubVec(raySt, f.P1)
	} else {
		t.SubVec(f.P1, raySt)
		a = -a
	}
	if a < utils.EPS {
		return SurfaceInteraction{}, false
	}

	math_lib.Cross(q, t, f.Mem.Edge1)
	u := mat.Dot(t, p) / a
	v := mat.Dot(rayDir, q) / a
	if u < 0 || u > 1 {
		return SurfaceInteraction{}, false
	}
	if v < 0 || u+v > 1 {
		return SurfaceInteraction{}, false
	}

	distance := mat.Dot(f.Mem.Edge2, q) / a
	if !distanceInRange(distance, tMin, tMax) {
		return SurfaceInteraction{}, false
	}

	if f.EngravingFunc != nil {
		if f.EngravingFunc(map[string]interface{}{
			"ray_start": raySt,
			"ray_dir":   rayDir,
			"distance":  distance,
			"self":      f,
		}) {
			return SurfaceInteraction{}, false
		}
	}

	interaction := newSurfaceInteraction(raySt, rayDir, distance, f.Mem.Normal)
	interaction.UV = [2]float64{u, v}
	interaction.DPDU = mat.VecDenseCopyOf(f.Mem.Edge1)
	interaction.DPDV = mat.VecDenseCopyOf(f.Mem.Edge2)
	return interaction, true
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
