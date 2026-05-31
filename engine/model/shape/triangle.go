package shape

import (
	"github.com/Algo2147483647/ray/engine/maths"
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
	Edge1    *mat.VecDense // First triangle edge vector.
	Edge2    *mat.VecDense // Second triangle edge vector.
	Normal   *mat.VecDense // Triangle surface normal.
	P1XYZ    [3]float64    // First vertex position in 3D.
	Edge1XYZ [3]float64    // First edge vector in 3D.
	Edge2XYZ [3]float64    // Second edge vector in 3D.
}

func NewTriangle(P1, P2, P3 *mat.VecDense) *Triangle {
	edge1 := mat.NewVecDense(P1.Len(), nil)
	edge2 := mat.NewVecDense(P1.Len(), nil)
	res := &Triangle{
		P1: P1,
		P2: P2,
		P3: P3,
		Mem: TriangleCalculateStorage{
			Edge1: maths.SubVec(edge1, P2, P1),
			Edge2: maths.SubVec(edge2, P3, P1),
		},
	}
	res.Mem.Normal = res.GetNormalVectorPure()
	if P1.Len() == 3 {
		res.Mem.P1XYZ = vecDenseXYZ(P1)
		res.Mem.Edge1XYZ = vecDenseXYZ(res.Mem.Edge1)
		res.Mem.Edge2XYZ = vecDenseXYZ(res.Mem.Edge2)
	}

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
	candidate, ok := f.IntersectCandidate(raySt, rayDir, tMin, tMax)
	if !ok {
		return SurfaceInteraction{}, false
	}
	return SurfaceInteractionFromCandidate(raySt, rayDir, candidate), true
}

func (f *Triangle) IntersectCandidate(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceCandidate, bool) {
	if raySt.Len() == 3 && rayDir.Len() == 3 {
		return f.intersectCandidate3D(raySt, rayDir, tMin, tMax)
	}

	t := mat.NewVecDense(raySt.Len(), nil)
	p := mat.NewVecDense(raySt.Len(), nil)
	q := mat.NewVecDense(raySt.Len(), nil)

	maths.Cross(p, rayDir, f.Mem.Edge2)
	a := mat.Dot(f.Mem.Edge1, p)
	if a > 0 {
		t.SubVec(raySt, f.P1)
	} else {
		t.SubVec(f.P1, raySt)
		a = -a
	}
	if a < utils.EPS {
		return SurfaceCandidate{}, false
	}

	maths.Cross(q, t, f.Mem.Edge1)
	u := mat.Dot(t, p) / a
	v := mat.Dot(rayDir, q) / a
	if u < 0 || u > 1 {
		return SurfaceCandidate{}, false
	}
	if v < 0 || u+v > 1 {
		return SurfaceCandidate{}, false
	}

	distance := mat.Dot(f.Mem.Edge2, q) / a
	if !distanceInRange(distance, tMin, tMax) {
		return SurfaceCandidate{}, false
	}

	if f.EngravingFunc != nil {
		if f.EngravingFunc(map[string]interface{}{
			"ray_start": raySt,
			"ray_dir":   rayDir,
			"distance":  distance,
			"self":      f,
		}) {
			return SurfaceCandidate{}, false
		}
	}

	return SurfaceCandidate{
		Distance:        distance,
		GeometricNormal: f.Mem.Normal,
		ShadingNormal:   f.Mem.Normal,
		UV:              [2]float64{u, v},
		DPDU:            f.Mem.Edge1,
		DPDV:            f.Mem.Edge2,
		PrimitiveID:     -1,
	}, true
}

func (f *Triangle) intersectCandidate3D(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceCandidate, bool) {
	ox, oy, oz := raySt.AtVec(0), raySt.AtVec(1), raySt.AtVec(2)
	dx, dy, dz := rayDir.AtVec(0), rayDir.AtVec(1), rayDir.AtVec(2)

	p1x, p1y, p1z := f.Mem.P1XYZ[0], f.Mem.P1XYZ[1], f.Mem.P1XYZ[2]
	e1x, e1y, e1z := f.Mem.Edge1XYZ[0], f.Mem.Edge1XYZ[1], f.Mem.Edge1XYZ[2]
	e2x, e2y, e2z := f.Mem.Edge2XYZ[0], f.Mem.Edge2XYZ[1], f.Mem.Edge2XYZ[2]

	px := dy*e2z - dz*e2y
	py := dz*e2x - dx*e2z
	pz := dx*e2y - dy*e2x
	det := e1x*px + e1y*py + e1z*pz
	if math.Abs(det) < utils.EPS {
		return SurfaceCandidate{}, false
	}

	invDet := 1 / det
	tx := ox - p1x
	ty := oy - p1y
	tz := oz - p1z
	u := (tx*px + ty*py + tz*pz) * invDet
	if u < 0 || u > 1 {
		return SurfaceCandidate{}, false
	}

	qx := ty*e1z - tz*e1y
	qy := tz*e1x - tx*e1z
	qz := tx*e1y - ty*e1x
	v := (dx*qx + dy*qy + dz*qz) * invDet
	if v < 0 || u+v > 1 {
		return SurfaceCandidate{}, false
	}

	distance := (e2x*qx + e2y*qy + e2z*qz) * invDet
	if !distanceInRange(distance, tMin, tMax) {
		return SurfaceCandidate{}, false
	}

	if f.EngravingFunc != nil {
		if f.EngravingFunc(map[string]interface{}{
			"ray_start": raySt,
			"ray_dir":   rayDir,
			"distance":  distance,
			"self":      f,
		}) {
			return SurfaceCandidate{}, false
		}
	}

	return SurfaceCandidate{
		Distance:        distance,
		GeometricNormal: f.Mem.Normal,
		ShadingNormal:   f.Mem.Normal,
		UV:              [2]float64{u, v},
		DPDU:            f.Mem.Edge1,
		DPDV:            f.Mem.Edge2,
		PrimitiveID:     -1,
	}, true
}

func (f *Triangle) GetNormalVector(_, res *mat.VecDense) *mat.VecDense {
	res.CloneFromVec(f.Mem.Normal)
	return res
}

func (f *Triangle) GetNormalVectorPure() *mat.VecDense {
	edge1 := mat.NewVecDense(f.P1.Len(), nil)
	edge2 := mat.NewVecDense(f.P1.Len(), nil)
	return maths.Normalize(maths.Cross2(maths.SubVec(edge1, f.P2, f.P1), maths.SubVec(edge2, f.P3, f.P1)))
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
