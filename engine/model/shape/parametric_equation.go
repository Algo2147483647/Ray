package shape

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

const (
	defaultParametricSamplesU      = 32
	defaultParametricSamplesV      = 32
	defaultParametricNewtonTol     = 1e-6
	defaultParametricNewtonMaxIter = 32
	defaultParametricDerivativeEps = 1e-5
	defaultParametricBoundsPadding = 1e-6
	defaultParametricResidualTol   = 1e-5
)

type ParametricFunction func(u, v float64) *mat.VecDense
type ParametricDerivative func(u, v float64, du, dv *mat.VecDense) (*mat.VecDense, *mat.VecDense)

type ParametricEquation struct {
	BaseShape
	Function   ParametricFunction
	Derivative ParametricDerivative
	URange     [2]float64
	VRange     [2]float64

	SamplesU      int
	SamplesV      int
	NewtonTol     float64
	NewtonMaxIter int
	DerivativeEps float64
	BoundsPadding float64
	ResidualTol   float64

	cachedBounds *Cuboid
	patches      []parametricPatch
	patchBVH     *parametricPatchBVHNode
	accelMu      sync.Mutex
}

type parametricHit struct {
	T        float64
	U        float64
	V        float64
	Residual float64
	PatchID  int
}

type parametricPatch struct {
	UMin    float64
	UMax    float64
	VMin    float64
	VMax    float64
	CenterU float64
	CenterV float64
	Bounds  *Cuboid
	PatchID int
}

type parametricPatchBVHNode struct {
	Bounds *Cuboid
	Left   *parametricPatchBVHNode
	Right  *parametricPatchBVHNode
	Patch  *parametricPatch
}

type parametricSeed struct {
	T       float64
	U       float64
	V       float64
	PatchID int
}

func NewParametricEquation(function ParametricFunction, ranges ...[2]float64) *ParametricEquation {
	equation := &ParametricEquation{
		Function:      function,
		URange:        [2]float64{0, 1},
		VRange:        [2]float64{0, 1},
		SamplesU:      defaultParametricSamplesU,
		SamplesV:      defaultParametricSamplesV,
		NewtonTol:     defaultParametricNewtonTol,
		NewtonMaxIter: defaultParametricNewtonMaxIter,
		DerivativeEps: defaultParametricDerivativeEps,
		BoundsPadding: defaultParametricBoundsPadding,
		ResidualTol:   defaultParametricResidualTol,
	}
	if len(ranges) > 0 {
		equation.URange = ranges[0]
	}
	if len(ranges) > 1 {
		equation.VRange = ranges[1]
	}
	return equation
}

func (p *ParametricEquation) Validate() error {
	if p == nil {
		return fmt.Errorf("parametric equation is nil")
	}
	if p.Function == nil {
		return fmt.Errorf("parametric equation requires a function")
	}
	if !validRange(p.URange) {
		return fmt.Errorf("parametric equation u_range must be finite and increasing")
	}
	if !validRange(p.VRange) {
		return fmt.Errorf("parametric equation v_range must be finite and increasing")
	}
	if p.samplesU() < 2 || p.samplesV() < 2 {
		return fmt.Errorf("parametric equation samples_u and samples_v must be >= 2")
	}
	probe := p.Function(midpoint(p.URange), midpoint(p.VRange))
	if probe == nil || probe.Len() < 3 || !finiteVec(probe, 3) {
		return fmt.Errorf("parametric equation function must return a finite 3D point")
	}
	return nil
}

func (p *ParametricEquation) BuildAcceleration() error {
	if p == nil {
		return fmt.Errorf("parametric equation is nil")
	}
	p.accelMu.Lock()
	defer p.accelMu.Unlock()
	return p.buildAccelerationLocked()
}

func (p *ParametricEquation) buildAccelerationLocked() error {
	if err := p.Validate(); err != nil {
		return err
	}

	patches := make([]parametricPatch, 0, p.samplesU()*p.samplesV())
	var bounds *Cuboid
	for patchV := 0; patchV < p.samplesV(); patchV++ {
		for patchU := 0; patchU < p.samplesU(); patchU++ {
			u0, u1 := p.patchRange(p.URange, patchU, p.samplesU())
			v0, v1 := p.patchRange(p.VRange, patchV, p.samplesV())
			patchBounds, ok := p.patchBounds(u0, u1, v0, v1)
			if !ok {
				continue
			}
			patchID := patchV*p.samplesU() + patchU
			patch := parametricPatch{
				UMin:    u0,
				UMax:    u1,
				VMin:    v0,
				VMax:    v1,
				CenterU: 0.5 * (u0 + u1),
				CenterV: 0.5 * (v0 + v1),
				Bounds:  patchBounds,
				PatchID: patchID,
			}
			patches = append(patches, patch)
			bounds = unionParametricBoxes(bounds, patchBounds)
		}
	}
	if len(patches) == 0 || bounds == nil {
		return fmt.Errorf("parametric equation produced no finite patches")
	}

	p.patches = patches
	p.patchBVH = buildParametricPatchBVH(p.patches)
	p.cachedBounds = bounds
	return nil
}

func (p *ParametricEquation) ensureAcceleration() error {
	if p == nil {
		return fmt.Errorf("parametric equation is nil")
	}
	p.accelMu.Lock()
	defer p.accelMu.Unlock()
	if p.cachedBounds != nil && p.patchBVH != nil {
		return nil
	}
	return p.buildAccelerationLocked()
}

func (p *ParametricEquation) Name() string {
	return "Parametric Equation"
}

func (p *ParametricEquation) Intersect(raySt, rayDir *mat.VecDense) float64 {
	interaction, ok := p.IntersectRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64
	}
	return interaction.Distance
}

func (p *ParametricEquation) IntersectRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceInteraction, bool) {
	candidate, ok := p.IntersectCandidate(raySt, rayDir, tMin, tMax)
	if !ok {
		return SurfaceInteraction{}, false
	}
	return SurfaceInteractionFromCandidate(raySt, rayDir, candidate), true
}

func (p *ParametricEquation) IntersectCandidate(raySt, rayDir *mat.VecDense, tMin, tMax float64) (SurfaceCandidate, bool) {
	if p == nil || raySt == nil || rayDir == nil || raySt.Len() != rayDir.Len() || raySt.Len() < 3 || tMax < tMin {
		return SurfaceCandidate{}, false
	}
	if err := p.ensureAcceleration(); err != nil {
		return SurfaceCandidate{}, false
	}
	if p.patchBVH == nil {
		return SurfaceCandidate{}, false
	}

	best, found := p.intersectPatchBVH(raySt, rayDir, p.patchBVH, tMin, tMax, parametricHit{T: math.MaxFloat64}, false)
	if !found {
		return SurfaceCandidate{}, false
	}
	interaction := p.interactionAt(best)
	return SurfaceCandidate{
		Distance:        interaction.Distance,
		ArcLength:       interaction.ArcLength,
		Point:           interaction.Point,
		GeometricNormal: interaction.GeometricNormal,
		ShadingNormal:   interaction.ShadingNormal,
		UV:              interaction.UV,
		DPDU:            interaction.DPDU,
		DPDV:            interaction.DPDV,
		PrimitiveID:     interaction.PrimitiveID,
	}, true
}

func (p *ParametricEquation) IntersectPure(raySt, rayDir *mat.VecDense, u0, v0, tol float64, maxIter int) float64 {
	if p == nil {
		return math.Inf(1)
	}
	oldTol, oldMaxIter := p.NewtonTol, p.NewtonMaxIter
	if tol > 0 {
		p.NewtonTol = tol
	}
	if maxIter > 0 {
		p.NewtonMaxIter = maxIter
	}
	defer func() {
		p.NewtonTol = oldTol
		p.NewtonMaxIter = oldMaxIter
	}()

	hit, ok := p.refineIntersection(raySt, rayDir, parametricSeed{T: utils.EPS, U: u0, V: v0}, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.Inf(1)
	}
	return hit.T
}

func (p *ParametricEquation) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	if intersect == nil {
		return res
	}
	if res == nil || res.Len() != intersect.Len() {
		res = mat.NewVecDense(intersect.Len(), nil)
	} else {
		res.Zero()
	}

	u, v, ok := p.closestParameters(intersect)
	if !ok {
		return res
	}
	normal, _, _ := p.normalAndDerivatives(u, v, intersect.Len())
	if normal == nil {
		return res
	}
	res.CopyVec(normal)
	return res
}

func (p *ParametricEquation) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	if p == nil || p.ensureAcceleration() != nil || p.cachedBounds == nil {
		return (&BaseShape{}).BuildBoundingBox()
	}
	return p.cachedBounds.BuildBoundingBox()
}

func (p *ParametricEquation) intersectPatchBVH(
	raySt, rayDir *mat.VecDense,
	node *parametricPatchBVHNode,
	tMin, tMax float64,
	best parametricHit,
	found bool,
) (parametricHit, bool) {
	if node == nil || node.Bounds == nil {
		return best, found
	}
	near, _, ok := node.Bounds.OverlapRange(raySt, rayDir, tMin, minFloat(tMax, best.T))
	if !ok {
		return best, found
	}
	if node.Patch != nil {
		seed := parametricSeed{
			T:       maxFloat(near, tMin),
			U:       node.Patch.CenterU,
			V:       node.Patch.CenterV,
			PatchID: node.Patch.PatchID,
		}
		hit, ok := p.refineIntersection(raySt, rayDir, seed, tMin, minFloat(tMax, best.T))
		if ok && hit.T < best.T {
			return hit, true
		}
		return best, found
	}

	leftNear, leftOK := nodeChildNear(raySt, rayDir, node.Left, tMin, minFloat(tMax, best.T))
	rightNear, rightOK := nodeChildNear(raySt, rayDir, node.Right, tMin, minFloat(tMax, best.T))
	if rightOK && (!leftOK || rightNear < leftNear) {
		best, found = p.intersectPatchBVH(raySt, rayDir, node.Right, tMin, tMax, best, found)
		best, found = p.intersectPatchBVH(raySt, rayDir, node.Left, tMin, tMax, best, found)
		return best, found
	}
	if leftOK {
		best, found = p.intersectPatchBVH(raySt, rayDir, node.Left, tMin, tMax, best, found)
	}
	if rightOK {
		best, found = p.intersectPatchBVH(raySt, rayDir, node.Right, tMin, tMax, best, found)
	}
	return best, found
}

func (p *ParametricEquation) refineIntersection(raySt, rayDir *mat.VecDense, seed parametricSeed, tMin, tMax float64) (parametricHit, bool) {
	if p == nil || raySt == nil || rayDir == nil || p.Function == nil {
		return parametricHit{}, false
	}

	x := []float64{seed.T, seed.U, seed.V}
	for iter := 0; iter < p.newtonMaxIter(); iter++ {
		fx, jacobian, ok := p.systemAndJacobian(raySt, rayDir, x)
		if !ok {
			return parametricHit{}, false
		}
		residual := infNormLocal(fx)
		if residual <= p.newtonTol() {
			return p.acceptHit(x, residual, seed.PatchID, tMin, tMax)
		}

		rhs := []float64{-fx[0], -fx[1], -fx[2]}
		delta, err := maths.SolveLinearSystem(jacobian, rhs)
		if err != nil {
			return parametricHit{}, false
		}
		if infNormLocal(delta) <= p.newtonTol() {
			for i := range x {
				x[i] += delta[i]
			}
			return p.acceptHit(x, residual, seed.PatchID, tMin, tMax)
		}

		alpha := 1.0
		accepted := false
		for alpha >= 1e-6 {
			candidate := []float64{
				x[0] + alpha*delta[0],
				x[1] + alpha*delta[1],
				x[2] + alpha*delta[2],
			}
			candidateResidual := p.residualNorm(raySt, rayDir, candidate)
			if isFinite(candidateResidual) && candidateResidual < residual {
				x = candidate
				accepted = true
				break
			}
			alpha *= 0.5
		}
		if !accepted {
			for i := range x {
				x[i] += delta[i]
			}
		}
	}

	fx, _, ok := p.systemAndJacobian(raySt, rayDir, x)
	if !ok {
		return parametricHit{}, false
	}
	return p.acceptHit(x, infNormLocal(fx), seed.PatchID, tMin, tMax)
}

func (p *ParametricEquation) systemAndJacobian(raySt, rayDir *mat.VecDense, x []float64) ([]float64, [][]float64, bool) {
	t, u, v := x[0], x[1], x[2]
	point := p.Function(u, v)
	if point == nil || point.Len() < 3 || !finiteVec(point, 3) {
		return nil, nil, false
	}
	du, dv := p.derivatives(u, v)
	if du == nil || dv == nil || du.Len() < 3 || dv.Len() < 3 || !finiteVec(du, 3) || !finiteVec(dv, 3) {
		return nil, nil, false
	}

	fx := make([]float64, 3)
	jacobian := make([][]float64, 3)
	for i := 0; i < 3; i++ {
		fx[i] = raySt.AtVec(i) + t*rayDir.AtVec(i) - point.AtVec(i)
		jacobian[i] = []float64{rayDir.AtVec(i), -du.AtVec(i), -dv.AtVec(i)}
	}
	return fx, jacobian, true
}

func (p *ParametricEquation) derivatives(u, v float64) (*mat.VecDense, *mat.VecDense) {
	du := mat.NewVecDense(3, nil)
	dv := mat.NewVecDense(3, nil)
	if p.Derivative != nil {
		du, dv = p.Derivative(u, v, du, dv)
		return du, dv
	}

	eps := p.derivativeEps()
	uPlus := p.Function(minFloat(u+eps, p.URange[1]), v)
	uMinus := p.Function(maxFloat(u-eps, p.URange[0]), v)
	vPlus := p.Function(u, minFloat(v+eps, p.VRange[1]))
	vMinus := p.Function(u, maxFloat(v-eps, p.VRange[0]))
	if uPlus == nil || uMinus == nil || vPlus == nil || vMinus == nil {
		return nil, nil
	}

	uDenom := minFloat(u+eps, p.URange[1]) - maxFloat(u-eps, p.URange[0])
	vDenom := minFloat(v+eps, p.VRange[1]) - maxFloat(v-eps, p.VRange[0])
	if uDenom <= 0 || vDenom <= 0 {
		return nil, nil
	}
	for i := 0; i < 3; i++ {
		du.SetVec(i, (uPlus.AtVec(i)-uMinus.AtVec(i))/uDenom)
		dv.SetVec(i, (vPlus.AtVec(i)-vMinus.AtVec(i))/vDenom)
	}
	return du, dv
}

func (p *ParametricEquation) acceptHit(x []float64, residual float64, patchID int, tMin, tMax float64) (parametricHit, bool) {
	t, u, v := x[0], x[1], x[2]
	if !distanceInRange(t, tMin, tMax) || residual > p.residualTol() {
		return parametricHit{}, false
	}
	if u < p.URange[0]-p.derivativeEps() || u > p.URange[1]+p.derivativeEps() {
		return parametricHit{}, false
	}
	if v < p.VRange[0]-p.derivativeEps() || v > p.VRange[1]+p.derivativeEps() {
		return parametricHit{}, false
	}
	return parametricHit{
		T:        t,
		U:        clampFloat(u, p.URange[0], p.URange[1]),
		V:        clampFloat(v, p.VRange[0], p.VRange[1]),
		Residual: residual,
		PatchID:  patchID,
	}, true
}

func (p *ParametricEquation) interactionAt(hit parametricHit) SurfaceInteraction {
	point := p.Function(hit.U, hit.V)
	normal, dpdu, dpdv := p.normalAndDerivatives(hit.U, hit.V, point.Len())
	return SurfaceInteraction{
		Distance:        hit.T,
		ArcLength:       0,
		Point:           point,
		GeometricNormal: normal,
		ShadingNormal:   normal,
		UV:              [2]float64{normalizeRange(hit.U, p.URange), normalizeRange(hit.V, p.VRange)},
		DPDU:            dpdu,
		DPDV:            dpdv,
		PrimitiveID:     hit.PatchID,
	}
}

func (p *ParametricEquation) normalAndDerivatives(u, v float64, dim int) (*mat.VecDense, *mat.VecDense, *mat.VecDense) {
	du3, dv3 := p.derivatives(u, v)
	if du3 == nil || dv3 == nil {
		return nil, nil, nil
	}
	dpdu := mat.NewVecDense(dim, nil)
	dpdv := mat.NewVecDense(dim, nil)
	for i := 0; i < dim && i < 3; i++ {
		dpdu.SetVec(i, du3.AtVec(i))
		dpdv.SetVec(i, dv3.AtVec(i))
	}
	normal3 := maths.Cross2(du3, dv3)
	if mat.Norm(normal3, 2) <= utils.EPS {
		return mat.NewVecDense(dim, nil), dpdu, dpdv
	}
	maths.Normalize(normal3)
	normal := mat.NewVecDense(dim, nil)
	for i := 0; i < dim && i < 3; i++ {
		normal.SetVec(i, normal3.AtVec(i))
	}
	return normal, dpdu, dpdv
}

func (p *ParametricEquation) patchBounds(u0, u1, v0, v1 float64) (*Cuboid, bool) {
	points := [][2]float64{
		{u0, v0}, {u1, v0}, {u0, v1}, {u1, v1},
		{0.5 * (u0 + u1), 0.5 * (v0 + v1)},
		{0.5 * (u0 + u1), v0}, {0.5 * (u0 + u1), v1},
		{u0, 0.5 * (v0 + v1)}, {u1, 0.5 * (v0 + v1)},
	}
	return boundsFromParamPoints(p, points, p.boundsPadding())
}

func (p *ParametricEquation) sampledBounds() *Cuboid {
	points := make([][2]float64, 0, (p.samplesU()+1)*(p.samplesV()+1))
	for iv := 0; iv <= p.samplesV(); iv++ {
		v := lerp(p.VRange, float64(iv)/float64(p.samplesV()))
		for iu := 0; iu <= p.samplesU(); iu++ {
			u := lerp(p.URange, float64(iu)/float64(p.samplesU()))
			points = append(points, [2]float64{u, v})
		}
	}
	bounds, ok := boundsFromParamPoints(p, points, p.boundsPadding())
	if !ok {
		pmin, pmax := (&BaseShape{}).BuildBoundingBox()
		return NewCuboid(pmin, pmax)
	}
	return bounds
}

func (p *ParametricEquation) closestParameters(point *mat.VecDense) (float64, float64, bool) {
	if p == nil || p.Function == nil || point == nil {
		return 0, 0, false
	}
	bestDistance := math.MaxFloat64
	bestU, bestV := 0.0, 0.0
	for iv := 0; iv <= p.samplesV(); iv++ {
		v := lerp(p.VRange, float64(iv)/float64(p.samplesV()))
		for iu := 0; iu <= p.samplesU(); iu++ {
			u := lerp(p.URange, float64(iu)/float64(p.samplesU()))
			candidate := p.Function(u, v)
			if candidate == nil || candidate.Len() < 3 {
				continue
			}
			d2 := 0.0
			for axis := 0; axis < 3; axis++ {
				d := candidate.AtVec(axis) - point.AtVec(axis)
				d2 += d * d
			}
			if d2 < bestDistance {
				bestDistance = d2
				bestU, bestV = u, v
			}
		}
	}
	return bestU, bestV, bestDistance < math.MaxFloat64
}

func (p *ParametricEquation) residualNorm(raySt, rayDir *mat.VecDense, x []float64) float64 {
	fx, _, ok := p.systemAndJacobian(raySt, rayDir, x)
	if !ok {
		return math.Inf(1)
	}
	return infNormLocal(fx)
}

func (p *ParametricEquation) patchRange(r [2]float64, index, count int) (float64, float64) {
	step := (r[1] - r[0]) / float64(count)
	return r[0] + float64(index)*step, r[0] + float64(index+1)*step
}

func (p *ParametricEquation) samplesU() int {
	if p != nil && p.SamplesU >= 2 {
		return p.SamplesU
	}
	return defaultParametricSamplesU
}

func (p *ParametricEquation) samplesV() int {
	if p != nil && p.SamplesV >= 2 {
		return p.SamplesV
	}
	return defaultParametricSamplesV
}

func (p *ParametricEquation) newtonTol() float64 {
	if p != nil && p.NewtonTol > 0 {
		return p.NewtonTol
	}
	return defaultParametricNewtonTol
}

func (p *ParametricEquation) newtonMaxIter() int {
	if p != nil && p.NewtonMaxIter > 0 {
		return p.NewtonMaxIter
	}
	return defaultParametricNewtonMaxIter
}

func (p *ParametricEquation) derivativeEps() float64 {
	if p != nil && p.DerivativeEps > 0 {
		return p.DerivativeEps
	}
	return defaultParametricDerivativeEps
}

func (p *ParametricEquation) boundsPadding() float64 {
	if p != nil && p.BoundsPadding >= 0 {
		return p.BoundsPadding
	}
	return defaultParametricBoundsPadding
}

func (p *ParametricEquation) residualTol() float64 {
	if p != nil && p.ResidualTol > 0 {
		return p.ResidualTol
	}
	return defaultParametricResidualTol
}

func boundsFromParamPoints(p *ParametricEquation, params [][2]float64, padding float64) (*Cuboid, bool) {
	pmin := []float64{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}
	pmax := []float64{-math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64}
	found := false
	for _, param := range params {
		point := p.Function(param[0], param[1])
		if point == nil || point.Len() < 3 || !finiteVec(point, 3) {
			continue
		}
		found = true
		for axis := 0; axis < 3; axis++ {
			value := point.AtVec(axis)
			if value < pmin[axis] {
				pmin[axis] = value
			}
			if value > pmax[axis] {
				pmax[axis] = value
			}
		}
	}
	if !found {
		return nil, false
	}
	for axis := 0; axis < 3; axis++ {
		pmin[axis] -= padding
		pmax[axis] += padding
		if pmin[axis] == pmax[axis] {
			pmin[axis] -= defaultParametricBoundsPadding
			pmax[axis] += defaultParametricBoundsPadding
		}
	}
	return NewCuboid(mat.NewVecDense(3, pmin), mat.NewVecDense(3, pmax)), true
}

func buildParametricPatchBVH(patches []parametricPatch) *parametricPatchBVHNode {
	if len(patches) == 0 {
		return nil
	}
	if len(patches) == 1 {
		patch := patches[0]
		return &parametricPatchBVHNode{
			Bounds: cloneParametricBox(patch.Bounds),
			Patch:  &patch,
		}
	}

	bounds := unionParametricPatchBounds(patches)
	axis := largestParametricCentroidExtent(patches)
	sort.Slice(patches, func(i, j int) bool {
		return parametricPatchCentroid(patches[i], axis) < parametricPatchCentroid(patches[j], axis)
	})
	mid := len(patches) / 2
	leftPatches := append([]parametricPatch(nil), patches[:mid]...)
	rightPatches := append([]parametricPatch(nil), patches[mid:]...)
	return &parametricPatchBVHNode{
		Bounds: bounds,
		Left:   buildParametricPatchBVH(leftPatches),
		Right:  buildParametricPatchBVH(rightPatches),
	}
}

func unionParametricPatchBounds(patches []parametricPatch) *Cuboid {
	var bounds *Cuboid
	for _, patch := range patches {
		bounds = unionParametricBoxes(bounds, patch.Bounds)
	}
	return bounds
}

func unionParametricBoxes(a, b *Cuboid) *Cuboid {
	if a == nil {
		return cloneParametricBox(b)
	}
	if b == nil {
		return cloneParametricBox(a)
	}
	dim := a.Pmin.Len()
	pmin := mat.NewVecDense(dim, nil)
	pmax := mat.NewVecDense(dim, nil)
	for i := 0; i < dim; i++ {
		pmin.SetVec(i, math.Min(a.Pmin.AtVec(i), b.Pmin.AtVec(i)))
		pmax.SetVec(i, math.Max(a.Pmax.AtVec(i), b.Pmax.AtVec(i)))
	}
	return NewCuboid(pmin, pmax)
}

func cloneParametricBox(box *Cuboid) *Cuboid {
	if box == nil {
		return nil
	}
	return NewCuboid(mat.VecDenseCopyOf(box.Pmin), mat.VecDenseCopyOf(box.Pmax))
}

func largestParametricCentroidExtent(patches []parametricPatch) int {
	bestAxis := 0
	bestExtent := math.Inf(-1)
	for axis := 0; axis < 3; axis++ {
		minValue := math.Inf(1)
		maxValue := math.Inf(-1)
		for _, patch := range patches {
			center := parametricPatchCentroid(patch, axis)
			if center < minValue {
				minValue = center
			}
			if center > maxValue {
				maxValue = center
			}
		}
		if extent := maxValue - minValue; extent > bestExtent {
			bestExtent = extent
			bestAxis = axis
		}
	}
	return bestAxis
}

func parametricPatchCentroid(patch parametricPatch, axis int) float64 {
	if patch.Bounds == nil {
		return 0
	}
	return 0.5 * (patch.Bounds.Pmin.AtVec(axis) + patch.Bounds.Pmax.AtVec(axis))
}

func nodeChildNear(raySt, rayDir *mat.VecDense, node *parametricPatchBVHNode, tMin, tMax float64) (float64, bool) {
	if node == nil || node.Bounds == nil {
		return 0, false
	}
	return node.Bounds.OverlapRangeNear(raySt, rayDir, tMin, tMax)
}

func finiteVec(v *mat.VecDense, dim int) bool {
	for i := 0; i < dim; i++ {
		value := v.AtVec(i)
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

func validRange(r [2]float64) bool {
	return isFinite(r[0]) && isFinite(r[1]) && r[0] < r[1]
}

func midpoint(r [2]float64) float64 {
	return 0.5 * (r[0] + r[1])
}

func lerp(r [2]float64, t float64) float64 {
	return r[0] + t*(r[1]-r[0])
}

func normalizeRange(value float64, r [2]float64) float64 {
	return (value - r[0]) / (r[1] - r[0])
}

func infNormLocal(values []float64) float64 {
	result := 0.0
	for _, value := range values {
		if math.Abs(value) > result {
			result = math.Abs(value)
		}
	}
	return result
}

func clampFloat(value, lo, hi float64) float64 {
	return minFloat(maxFloat(value, lo), hi)
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
