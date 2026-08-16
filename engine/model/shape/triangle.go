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

func (f *Triangle) IntersectAffine(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if !options.valid() {
		return SurfaceInteraction{}, false
	}
	if raySt.Len() == 3 && rayDir.Len() == 3 {
		return f.intersect3D(raySt, rayDir, options.Range)
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
		return SurfaceInteraction{}, false
	}

	maths.Cross(q, t, f.Mem.Edge1)
	u := mat.Dot(t, p) / a
	v := mat.Dot(rayDir, q) / a
	if u < 0 || u > 1 {
		return SurfaceInteraction{}, false
	}
	if v < 0 || u+v > 1 {
		return SurfaceInteraction{}, false
	}

	distance := mat.Dot(f.Mem.Edge2, q) / a
	if !distanceInRange(distance, options.Range.Min, options.Range.Max) {
		return SurfaceInteraction{}, false
	}

	return f.interactionAt(raySt, rayDir, distance, u, v), true
}

func (f *Triangle) intersect3D(raySt, rayDir *mat.VecDense, interval Interval) (SurfaceInteraction, bool) {
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
		return SurfaceInteraction{}, false
	}

	invDet := 1 / det
	tx := ox - p1x
	ty := oy - p1y
	tz := oz - p1z
	u := (tx*px + ty*py + tz*pz) * invDet
	if u < 0 || u > 1 {
		return SurfaceInteraction{}, false
	}

	qx := ty*e1z - tz*e1y
	qy := tz*e1x - tx*e1z
	qz := tx*e1y - ty*e1x
	v := (dx*qx + dy*qy + dz*qz) * invDet
	if v < 0 || u+v > 1 {
		return SurfaceInteraction{}, false
	}

	distance := (e2x*qx + e2y*qy + e2z*qz) * invDet
	if !distanceInRange(distance, interval.Min, interval.Max) {
		return SurfaceInteraction{}, false
	}

	return f.interactionAt(raySt, rayDir, distance, u, v), true
}

func (f *Triangle) interactionAt(raySt, rayDir *mat.VecDense, distance, u, v float64) SurfaceInteraction {
	interaction := newAffineSurfaceInteraction(raySt, rayDir, distance, f.Mem.Normal)
	interaction.UV = [2]float64{u, v}
	interaction.DPDU = f.Mem.Edge1
	interaction.DPDV = f.Mem.Edge2
	return interaction
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

func (f *Triangle) SurfaceArea() float64 {
	if f == nil || f.P1 == nil || f.P1.Len() != 3 {
		return 0
	}
	cross := maths.Cross2(f.Mem.Edge1, f.Mem.Edge2)
	return 0.5 * mat.Norm(cross, 2)
}

func (f *Triangle) SampleSurface(u maths.Sample2D) (SurfaceSample, bool) {
	area := f.SurfaceArea()
	if area <= 0 {
		return SurfaceSample{}, false
	}
	su := math.Sqrt(clampUnit(u.U))
	b1 := 1 - su
	b2 := clampUnit(u.V) * su
	point := mat.NewVecDense(3, nil)
	point.AddScaledVec(point, b1, f.Mem.Edge1)
	point.AddScaledVec(point, b2, f.Mem.Edge2)
	point.AddVec(point, f.P1)
	normal := mat.VecDenseCopyOf(f.Mem.Normal)
	return SurfaceSample{Point: point, Normal: normal, UV: [2]float64{b1, b2}, PDFArea: 1 / area}, true
}

const (
	minSphericalTriangleArea = 3e-4
	maxSphericalTriangleArea = 6.22
)

// SampleSurfaceFrom uniformly samples the solid angle subtended by this
// triangle at reference (Arvo's spherical-triangle construction), then
// converts the density back to area measure. Very small or near-hemispherical
// configurations use the numerically safer uniform-area sampler.
func (f *Triangle) SampleSurfaceFrom(reference *mat.VecDense, u maths.Sample2D) (SurfaceSample, bool) {
	if f == nil || reference == nil || reference.Len() != 3 || f.P1 == nil || f.P1.Len() != 3 {
		return SurfaceSample{}, false
	}
	solidAngle, useReferenceDistribution := f.referenceSamplingSolidAngle(reference)
	if !useReferenceDistribution {
		return f.SampleSurface(u)
	}
	vertices := [3][3]float64{vecDenseXYZ(f.P1), vecDenseXYZ(f.P2), vecDenseXYZ(f.P3)}
	ref := vecDenseXYZ(reference)
	directions := [3][3]float64{}
	for index := range 3 {
		for axis := range 3 {
			directions[index][axis] = vertices[index][axis] - ref[axis]
		}
		if !normalize3(&directions[index]) {
			return SurfaceSample{}, false
		}
	}

	nAB, nBC, nCA := cross3(directions[0], directions[1]), cross3(directions[1], directions[2]), cross3(directions[2], directions[0])
	if !normalize3(&nAB) || !normalize3(&nBC) || !normalize3(&nCA) {
		return SurfaceSample{}, false
	}
	alpha := angleBetween3(nAB, negate3(nCA))

	apPi := math.Pi + clampUnit(u.U)*solidAngle
	cosAlpha, sinAlpha := math.Cos(alpha), math.Sin(alpha)
	sinPhi := math.Sin(apPi)*cosAlpha - math.Cos(apPi)*sinAlpha
	cosPhi := math.Cos(apPi)*cosAlpha + math.Sin(apPi)*sinAlpha
	k1 := cosPhi + cosAlpha
	k2 := sinPhi - sinAlpha*dot3(directions[0], directions[1])
	denominator := (k2*sinPhi + k1*cosPhi) * sinAlpha
	if math.Abs(denominator) <= utils.EPS {
		return SurfaceSample{}, false
	}
	cosBP := (k2 + (k2*cosPhi-k1*sinPhi)*cosAlpha) / denominator
	cosBP = math.Max(-1, math.Min(1, cosBP))
	orthogonalC := gramSchmidt3(directions[2], directions[0])
	if !normalize3(&orthogonalC) {
		return SurfaceSample{}, false
	}
	cp := addScaled3(scale3(directions[0], cosBP), orthogonalC, math.Sqrt(math.Max(0, 1-cosBP*cosBP)))
	cosTheta := 1 - clampUnit(u.V)*(1-dot3(cp, directions[1]))
	orthogonalCP := gramSchmidt3(cp, directions[1])
	if !normalize3(&orthogonalCP) {
		return SurfaceSample{}, false
	}
	w := addScaled3(scale3(directions[1], cosTheta), orthogonalCP, math.Sqrt(math.Max(0, 1-cosTheta*cosTheta)))
	if !normalize3(&w) {
		return SurfaceSample{}, false
	}

	normal := vecDenseXYZ(f.Mem.Normal)
	planeDistance := dot3(sub3(vertices[0], ref), normal)
	projection := dot3(w, normal)
	if math.Abs(projection) <= utils.EPS {
		return SurfaceSample{}, false
	}
	distance := planeDistance / projection
	if distance <= utils.EPS || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return SurfaceSample{}, false
	}
	pointXYZ := addScaled3(ref, w, distance)
	b1, b2, ok := triangleBarycentrics(pointXYZ, vertices[0], vertices[1], vertices[2])
	if !ok {
		return SurfaceSample{}, false
	}
	b1 = clampUnit(b1)
	b2 = clampUnit(b2)
	if b1+b2 > 1 {
		sum := b1 + b2
		b1, b2 = b1/sum, b2/sum
		pointXYZ = addScaled3(addScaled3(vertices[0], f.Mem.Edge1XYZ, b1), f.Mem.Edge2XYZ, b2)
	}
	pdfArea := math.Abs(projection) / (solidAngle * distance * distance)
	if pdfArea <= 0 || math.IsNaN(pdfArea) || math.IsInf(pdfArea, 0) {
		return SurfaceSample{}, false
	}
	return SurfaceSample{
		Point: mat.NewVecDense(3, pointXYZ[:]), Normal: mat.VecDenseCopyOf(f.Mem.Normal),
		UV: [2]float64{b1, b2}, PDFArea: pdfArea,
	}, true
}

func (f *Triangle) SurfacePDFFrom(reference, point *mat.VecDense) float64 {
	if f == nil || reference == nil || point == nil || reference.Len() != 3 || point.Len() != 3 {
		return 0
	}
	solidAngle, useReferenceDistribution := f.referenceSamplingSolidAngle(reference)
	if !useReferenceDistribution {
		return SurfacePDF(f, point)
	}
	direction := [3]float64{}
	distance2 := 0.0
	for axis := range 3 {
		direction[axis] = point.AtVec(axis) - reference.AtVec(axis)
		distance2 += direction[axis] * direction[axis]
	}
	if !normalize3(&direction) {
		return 0
	}
	cosine := math.Abs(dot3(vecDenseXYZ(f.Mem.Normal), direction))
	pdf := cosine / (solidAngle * distance2)
	if pdf <= 0 || math.IsNaN(pdf) || math.IsInf(pdf, 0) {
		return 0
	}
	return pdf
}

func (f *Triangle) referenceSamplingSolidAngle(reference *mat.VecDense) (float64, bool) {
	solidAngle := f.solidAngleFrom(reference)
	if solidAngle < minSphericalTriangleArea || solidAngle > maxSphericalTriangleArea ||
		math.IsNaN(solidAngle) || math.IsInf(solidAngle, 0) {
		return 0, false
	}
	return solidAngle, true
}

func (f *Triangle) solidAngleFrom(reference *mat.VecDense) float64 {
	if f == nil || reference == nil || reference.Len() != 3 || f.P1.Len() != 3 {
		return 0
	}
	ref := vecDenseXYZ(reference)
	directions := [3][3]float64{}
	for index, vertex := range [3][3]float64{vecDenseXYZ(f.P1), vecDenseXYZ(f.P2), vecDenseXYZ(f.P3)} {
		directions[index] = sub3(vertex, ref)
		if !normalize3(&directions[index]) {
			return 0
		}
	}
	determinant := math.Abs(dot3(directions[0], cross3(directions[1], directions[2])))
	denominator := 1 + dot3(directions[0], directions[1]) +
		dot3(directions[1], directions[2]) + dot3(directions[2], directions[0])
	return 2 * math.Atan2(determinant, denominator)
}

func normalize3(v *[3]float64) bool {
	length2 := dot3(*v, *v)
	if length2 <= utils.EPS*utils.EPS || math.IsNaN(length2) || math.IsInf(length2, 0) {
		return false
	}
	inverse := 1 / math.Sqrt(length2)
	for axis := range 3 {
		v[axis] *= inverse
	}
	return true
}

func dot3(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func cross3(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

func sub3(a, b [3]float64) [3]float64 {
	return [3]float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

func negate3(v [3]float64) [3]float64 { return [3]float64{-v[0], -v[1], -v[2]} }

func scale3(v [3]float64, scale float64) [3]float64 {
	return [3]float64{v[0] * scale, v[1] * scale, v[2] * scale}
}

func addScaled3(a, b [3]float64, scale float64) [3]float64 {
	return [3]float64{a[0] + b[0]*scale, a[1] + b[1]*scale, a[2] + b[2]*scale}
}

func gramSchmidt3(a, b [3]float64) [3]float64 {
	return addScaled3(a, b, -dot3(a, b))
}

func angleBetween3(a, b [3]float64) float64 {
	return math.Atan2(math.Sqrt(math.Max(0, dot3(cross3(a, b), cross3(a, b)))), dot3(a, b))
}

func triangleBarycentrics(point, p0, p1, p2 [3]float64) (float64, float64, bool) {
	v0, v1, v2 := sub3(p1, p0), sub3(p2, p0), sub3(point, p0)
	d00, d01, d11 := dot3(v0, v0), dot3(v0, v1), dot3(v1, v1)
	d20, d21 := dot3(v2, v0), dot3(v2, v1)
	denominator := d00*d11 - d01*d01
	if math.Abs(denominator) <= utils.EPS {
		return 0, 0, false
	}
	return (d11*d20 - d01*d21) / denominator,
		(d00*d21 - d01*d20) / denominator, true
}
