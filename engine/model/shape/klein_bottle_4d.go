package shape

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

// KleinBottle4D is a 4D embedding of the Klein bottle rendered as the boundary
// of an epsilon tube around the base two-dimensional surface.
//
// S(u, v) = (
//
//	(R + r cos v) cos u,
//	(R + r cos v) sin u,
//	r sin v cos(u/2),
//	r sin v sin(u/2),
//
// )
//
// The surface itself has codimension 2, so a 1D ray almost never hits it
// directly. The ray tracer intersects the codimension-1 tube boundary defined
// by dist(p, S) == Thickness.
type KleinBottle4D struct {
	BaseShape

	Center    *mat.VecDense
	R         float64
	Minor     float64
	Thickness float64

	uSeeds int
	vSeeds int
	topK   int

	seeds []kleinSeed

	maxNewton  int
	newtonTol  float64
	maxMarch   int
	marchEps   float64
	stepFactor float64
	minStep    float64
}

type kleinSeed struct {
	u, v float64
	p    [4]float64
}

type closestResult struct {
	uRaw, vRaw float64
	uUV, vUV   float64

	q      [4]float64
	distSq float64
}

// NewKleinBottle4D builds a Klein bottle epsilon tube.
func NewKleinBottle4D(center *mat.VecDense, majorR, minorR, thickness float64) *KleinBottle4D {
	if majorR <= 0 {
		panic("KleinBottle4D: major radius must be positive")
	}
	if minorR <= 0 {
		panic("KleinBottle4D: minor radius must be positive")
	}
	if thickness <= 0 {
		panic("KleinBottle4D: thickness must be positive")
	}
	if majorR <= minorR {
		panic("KleinBottle4D: require majorR > minorR for a clean 4D embedding")
	}

	k := &KleinBottle4D{
		Center:    center,
		R:         majorR,
		Minor:     minorR,
		Thickness: thickness,

		uSeeds: 16,
		vSeeds: 8,
		topK:   8,

		maxNewton: 10,
		newtonTol: 1e-10,

		maxMarch:   128,
		marchEps:   math.Max(1e-6, thickness*0.01),
		stepFactor: 0.75,
		minStep:    math.Max(1e-6, thickness*0.02),
	}

	k.rebuildSeeds()
	return k
}

func (k *KleinBottle4D) Name() string { return "Klein Bottle 4D" }

func (k *KleinBottle4D) rebuildSeeds() {
	k.seeds = make([]kleinSeed, 0, k.uSeeds*k.vSeeds)

	for iu := 0; iu < k.uSeeds; iu++ {
		u := 2 * math.Pi * float64(iu) / float64(k.uSeeds)

		for iv := 0; iv < k.vSeeds; iv++ {
			v := 2 * math.Pi * float64(iv) / float64(k.vSeeds)
			p := k.surfacePoint4(u, v)

			k.seeds = append(k.seeds, kleinSeed{
				u: u,
				v: v,
				p: p,
			})
		}
	}
}

func (k *KleinBottle4D) surfacePoint(u, v float64) (x0, x1, x2, x3 float64) {
	cu, su := math.Cos(u), math.Sin(u)
	cv, sv := math.Cos(v), math.Sin(v)

	hu := u * 0.5
	chu, shu := math.Cos(hu), math.Sin(hu)

	rho := k.R + k.Minor*cv

	x0 = rho * cu
	x1 = rho * su
	x2 = k.Minor * sv * chu
	x3 = k.Minor * sv * shu

	return
}

func (k *KleinBottle4D) surfacePoint4(u, v float64) [4]float64 {
	x0, x1, x2, x3 := k.surfacePoint(u, v)
	return [4]float64{x0, x1, x2, x3}
}

func (k *KleinBottle4D) surfaceDeriv(u, v float64) (S, Su, Sv [4]float64) {
	cu, su := math.Cos(u), math.Sin(u)
	cv, sv := math.Cos(v), math.Sin(v)

	hu := u * 0.5
	chu, shu := math.Cos(hu), math.Sin(hu)

	r := k.Minor
	rho := k.R + r*cv
	rhoDv := -r * sv

	S = [4]float64{
		rho * cu,
		rho * su,
		r * sv * chu,
		r * sv * shu,
	}

	Su = [4]float64{
		-rho * su,
		rho * cu,
		-0.5 * r * sv * shu,
		0.5 * r * sv * chu,
	}

	Sv = [4]float64{
		rhoDv * cu,
		rhoDv * su,
		r * cv * chu,
		r * cv * shu,
	}

	return
}

func wrapKleinUV(u, v float64) (float64, float64) {
	twoPi := 2 * math.Pi

	nFloat := math.Floor(u / twoPi)
	u = u - nFloat*twoPi

	n := int64(nFloat)
	if n%2 != 0 {
		v = -v
	}

	v = math.Mod(v, twoPi)
	if v < 0 {
		v += twoPi
	}

	return u, v
}

func (k *KleinBottle4D) closestPointFull(p [4]float64) closestResult {
	if len(k.seeds) == 0 {
		k.rebuildSeeds()
	}

	type seedCandidate struct {
		u, v float64
		sq   float64
	}

	topK := k.topK
	if topK <= 0 {
		topK = 8
	}

	candidates := make([]seedCandidate, 0, topK)

	insertSeed := func(s seedCandidate) {
		idx := len(candidates)
		for idx > 0 && candidates[idx-1].sq > s.sq {
			idx--
		}
		if idx >= topK {
			return
		}
		if len(candidates) < topK {
			candidates = append(candidates, seedCandidate{})
		}
		copy(candidates[idx+1:], candidates[idx:len(candidates)-1])
		candidates[idx] = s
	}

	for _, s := range k.seeds {
		insertSeed(seedCandidate{
			u:  s.u,
			v:  s.v,
			sq: distSq4(s.p, p),
		})
	}

	bestU, bestV := 0.0, 0.0
	bestQ := [4]float64{}
	bestSq := math.Inf(1)
	maxStep := 0.25 * math.Pi

	for _, s := range candidates {
		uu, vv := s.u, s.v

		q := k.surfacePoint4(uu, vv)
		sq := distSq4(q, p)

		if sq < bestSq {
			bestSq = sq
			bestU = uu
			bestV = vv
			bestQ = q
		}

		for iter := 0; iter < k.maxNewton; iter++ {
			S, Su, Sv := k.surfaceDeriv(uu, vv)

			diff := [4]float64{
				S[0] - p[0],
				S[1] - p[1],
				S[2] - p[2],
				S[3] - p[3],
			}

			fU := dot4(Su, diff)
			fV := dot4(Sv, diff)

			if fU*fU+fV*fV < k.newtonTol {
				break
			}

			guu := dot4(Su, Su)
			guv := dot4(Su, Sv)
			gvv := dot4(Sv, Sv)

			det := guu*gvv - guv*guv
			if math.Abs(det) < 1e-14 {
				break
			}

			du := (gvv*fU - guv*fV) / det
			dv := (-guv*fU + guu*fV) / det

			if stepLen := math.Hypot(du, dv); stepLen > maxStep {
				scale := maxStep / stepLen
				du *= scale
				dv *= scale
			}

			oldSq := sq
			accepted := false
			trialDu := du
			trialDv := dv

			for ls := 0; ls < 6; ls++ {
				tu := uu - trialDu
				tv := vv - trialDv

				tq := k.surfacePoint4(tu, tv)
				tsq := distSq4(tq, p)

				if tsq <= oldSq {
					uu = tu
					vv = tv
					q = tq
					sq = tsq
					accepted = true
					break
				}

				trialDu *= 0.5
				trialDv *= 0.5
			}

			if !accepted {
				break
			}

			if sq < bestSq {
				bestSq = sq
				bestU = uu
				bestV = vv
				bestQ = q
			}
		}
	}

	uUV, vUV := wrapKleinUV(bestU, bestV)

	return closestResult{
		uRaw:   bestU,
		vRaw:   bestV,
		uUV:    uUV,
		vUV:    vUV,
		q:      bestQ,
		distSq: math.Max(bestSq, 0),
	}
}

func (k *KleinBottle4D) closestPoint(p [4]float64) (u, v, distSq float64) {
	cp := k.closestPointFull(p)
	return cp.uUV, cp.vUV, cp.distSq
}

// SDF returns dist(p, S) - Thickness.
func (k *KleinBottle4D) SDF(p *mat.VecDense) float64 {
	if p == nil || p.Len() != 4 {
		return math.Inf(1)
	}

	pl := localPoint(p, k.Center)
	cp := k.closestPointFull(pl)

	return math.Sqrt(math.Max(cp.distSq, 0)) - k.Thickness
}

func (k *KleinBottle4D) UVAtPoint(p *mat.VecDense) (u, v float64) {
	if p == nil || p.Len() != 4 {
		return 0, 0
	}

	pl := localPoint(p, k.Center)
	cp := k.closestPointFull(pl)

	return cp.uUV, cp.vUV
}

func (k *KleinBottle4D) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	if raySt == nil || rayDir == nil || raySt.Len() != 4 || rayDir.Len() != 4 {
		return SurfaceInteraction{}, false
	}

	pmin, pmax := k.BuildBoundingBox()
	tEnter, tExit, ok := rayBoxRange(raySt, rayDir, pmin, pmax, tMin, tMax)
	if !ok {
		return SurfaceInteraction{}, false
	}

	t := math.Max(tEnter, tMin)
	tEnd := math.Min(tExit, tMax)

	if t > tEnd {
		return SurfaceInteraction{}, false
	}

	point := mat.NewVecDense(4, nil)

	hasPrev := false
	prevT := 0.0
	prevSDF := 0.0

	for iter := 0; iter < k.maxMarch && t <= tEnd; iter++ {
		point.AddScaledVec(raySt, t, rayDir)
		sdf := k.SDF(point)

		if math.Abs(sdf) < k.marchEps {
			return k.makeInteraction(t, point), true
		}

		if hasPrev && signChanged(prevSDF, sdf) {
			tHit, ok := k.refineCrossing(raySt, rayDir, prevT, t)
			if ok {
				point.AddScaledVec(raySt, tHit, rayDir)
				return k.makeInteraction(tHit, point), true
			}
		}

		step := math.Abs(sdf) * k.stepFactor
		if step < k.minStep {
			step = k.minStep
		}

		prevT = t
		prevSDF = sdf
		hasPrev = true

		t += step
	}

	return SurfaceInteraction{}, false
}

func (k *KleinBottle4D) refineCrossing(raySt, rayDir *mat.VecDense, lo, hi float64) (float64, bool) {
	point := mat.NewVecDense(4, nil)

	point.AddScaledVec(raySt, lo, rayDir)
	sdfLo := k.SDF(point)

	point.AddScaledVec(raySt, hi, rayDir)
	sdfHi := k.SDF(point)

	if !signChanged(sdfLo, sdfHi) {
		return 0, false
	}

	for iter := 0; iter < 32; iter++ {
		mid := 0.5 * (lo + hi)

		point.AddScaledVec(raySt, mid, rayDir)
		sdfMid := k.SDF(point)

		if math.Abs(sdfMid) < k.marchEps || hi-lo < k.marchEps {
			return mid, true
		}

		if signChanged(sdfLo, sdfMid) {
			hi = mid
			sdfHi = sdfMid
		} else {
			lo = mid
			sdfLo = sdfMid
		}
	}

	_ = sdfHi
	return 0.5 * (lo + hi), true
}

func (k *KleinBottle4D) makeInteraction(t float64, point *mat.VecDense) SurfaceInteraction {
	pt := mat.VecDenseCopyOf(point)

	pl := localPoint(pt, k.Center)
	cp := k.closestPointFull(pl)

	normal := mat.NewVecDense(4, nil)
	k.normalFromClosestLocal(pl, cp, normal)

	return SurfaceInteraction{
		Distance:        t,
		Point:           pt,
		GeometricNormal: normal,
		ShadingNormal:   normal,
		UV:              [2]float64{cp.uUV, cp.vUV},
		PrimitiveID:     -1,
	}
}

func (k *KleinBottle4D) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	if res == nil || res.Len() != 4 {
		res = mat.NewVecDense(4, nil)
	}

	if intersect == nil || intersect.Len() != 4 {
		return res
	}

	pl := localPoint(intersect, k.Center)
	cp := k.closestPointFull(pl)

	return k.normalFromClosestLocal(pl, cp, res)
}

func (k *KleinBottle4D) normalFromClosestLocal(p [4]float64, cp closestResult, res *mat.VecDense) *mat.VecDense {
	n0 := p[0] - cp.q[0]
	n1 := p[1] - cp.q[1]
	n2 := p[2] - cp.q[2]
	n3 := p[3] - cp.q[3]

	length := math.Sqrt(n0*n0 + n1*n1 + n2*n2 + n3*n3)

	if length < 1e-14 {
		res.SetVec(0, 1)
		res.SetVec(1, 0)
		res.SetVec(2, 0)
		res.SetVec(3, 0)
		return res
	}

	inv := 1.0 / length

	res.SetVec(0, n0*inv)
	res.SetVec(1, n1*inv)
	res.SetVec(2, n2*inv)
	res.SetVec(3, n3*inv)

	return res
}

func (k *KleinBottle4D) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	dim := utils.Dimension
	if dim != 4 && k.Center != nil {
		dim = k.Center.Len()
	}
	if dim < 4 {
		dim = 4
	}

	pmin = mat.NewVecDense(dim, nil)
	pmax = mat.NewVecDense(dim, nil)

	pad := k.Thickness + k.marchEps

	limits := [4]float64{
		k.R + k.Minor + pad,
		k.R + k.Minor + pad,
		k.Minor + pad,
		k.Minor + pad,
	}

	for i := 0; i < 4; i++ {
		c := 0.0
		if k.Center != nil && i < k.Center.Len() {
			c = k.Center.AtVec(i)
		}

		pmin.SetVec(i, c-limits[i])
		pmax.SetVec(i, c+limits[i])
	}

	return
}

func (k *KleinBottle4D) Intersect(raySt, rayDir *mat.VecDense) float64 {
	si, ok := k.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return si.Distance
}

func dot4(a, b [4]float64) float64 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] + a[3]*b[3]
}

func distSq4(a, b [4]float64) float64 {
	d0 := a[0] - b[0]
	d1 := a[1] - b[1]
	d2 := a[2] - b[2]
	d3 := a[3] - b[3]

	return d0*d0 + d1*d1 + d2*d2 + d3*d3
}

func localPoint(p, center *mat.VecDense) [4]float64 {
	out := [4]float64{}

	if p == nil || p.Len() < 4 {
		return out
	}

	for i := 0; i < 4; i++ {
		c := 0.0
		if center != nil && i < center.Len() {
			c = center.AtVec(i)
		}
		out[i] = p.AtVec(i) - c
	}

	return out
}

func signChanged(a, b float64) bool {
	return (a < 0 && b > 0) || (a > 0 && b < 0)
}

func rayBoxRange(raySt, rayDir, pmin, pmax *mat.VecDense, tMin, tMax float64) (float64, float64, bool) {
	tEnter, tExit := tMin, tMax

	for i := 0; i < raySt.Len(); i++ {
		o := raySt.AtVec(i)
		d := rayDir.AtVec(i)
		lo := pmin.AtVec(i)
		hi := pmax.AtVec(i)

		if math.Abs(d) < 1e-12 {
			if o < lo || o > hi {
				return 0, 0, false
			}
			continue
		}

		t1 := (lo - o) / d
		t2 := (hi - o) / d

		if t1 > t2 {
			t1, t2 = t2, t1
		}

		if t1 > tEnter {
			tEnter = t1
		}
		if t2 < tExit {
			tExit = t2
		}

		if tEnter > tExit {
			return 0, 0, false
		}
	}

	return tEnter, tExit, true
}

func (k *KleinBottle4D) GetNormalVectorFiniteDifference(intersect, res *mat.VecDense) *mat.VecDense {
	if res == nil || res.Len() != 4 {
		res = mat.NewVecDense(4, nil)
	}

	if intersect == nil || intersect.Len() != 4 {
		return res
	}

	h := math.Max(1e-5, k.Thickness*0.002)

	probe := mat.VecDenseCopyOf(intersect)
	g := [4]float64{}

	for i := 0; i < 4; i++ {
		probe.SetVec(i, intersect.AtVec(i)+h)
		fp := k.SDF(probe)

		probe.SetVec(i, intersect.AtVec(i)-h)
		fm := k.SDF(probe)

		probe.SetVec(i, intersect.AtVec(i))

		g[i] = (fp - fm) / (2 * h)
	}

	res.SetVec(0, g[0])
	res.SetVec(1, g[1])
	res.SetVec(2, g[2])
	res.SetVec(3, g[3])

	return maths.Normalize(res)
}
