package shape

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

const (
	defaultParametricCurveSamples       = 256
	defaultParametricCurveRefineIter    = 40
	defaultParametricCurveDerivativeEps = 1e-5
	defaultParametricCurveBoundsPadding = 1e-6
)

type ParametricCurveFunction func(t float64) *mat.VecDense
type ParametricCurveDerivative func(t float64, res *mat.VecDense) *mat.VecDense
type ParametricCurveRadius func(t float64) float64

type ParametricCurve struct {
	Function   ParametricCurveFunction
	Derivative ParametricCurveDerivative
	Radius     ParametricCurveRadius
	TRange     [2]float64

	Samples       int
	RefineIter    int
	DerivativeEps float64
	BoundsPadding float64

	cachedBounds *Cuboid
	segments     []parametricCurveSegment
	segmentBVH   *parametricCurveBVHNode
	accelMu      sync.Mutex
}

type parametricCurveHit struct {
	Distance float64
	T        float64
}

type parametricCurveSample struct {
	T      float64
	Point  *mat.VecDense
	Radius float64
}

type parametricCurveSegment struct {
	T0        float64
	T1        float64
	P0        *mat.VecDense
	P1        *mat.VecDense
	RadiusMax float64
	Bounds    *Cuboid
	SegmentID int
}

type parametricCurveBVHNode struct {
	Bounds  *Cuboid
	Left    *parametricCurveBVHNode
	Right   *parametricCurveBVHNode
	Segment *parametricCurveSegment
}

func NewParametricCurve(function ParametricCurveFunction, radius ParametricCurveRadius, tRange ...[2]float64) *ParametricCurve {
	curve := &ParametricCurve{
		Function:      function,
		Radius:        radius,
		TRange:        [2]float64{0, 1},
		Samples:       defaultParametricCurveSamples,
		RefineIter:    defaultParametricCurveRefineIter,
		DerivativeEps: defaultParametricCurveDerivativeEps,
		BoundsPadding: defaultParametricCurveBoundsPadding,
	}
	if len(tRange) > 0 {
		curve.TRange = tRange[0]
	}
	return curve
}

func (c *ParametricCurve) Validate() error {
	if c == nil {
		return fmt.Errorf("parametric curve is nil")
	}
	if c.Function == nil {
		return fmt.Errorf("parametric curve requires a function")
	}
	if c.Radius == nil {
		return fmt.Errorf("parametric curve requires a radius")
	}
	if !validRange(c.TRange) {
		return fmt.Errorf("parametric curve t_range must be finite and increasing")
	}
	if c.samples() < 2 {
		return fmt.Errorf("parametric curve samples must be >= 2")
	}
	probeT := midpoint(c.TRange)
	probe := c.Function(probeT)
	if probe == nil || probe.Len() < 3 || !finiteVec(probe, 3) {
		return fmt.Errorf("parametric curve function must return a finite 3D point")
	}
	radius := c.Radius(probeT)
	if radius <= 0 || !maths.IsFinite(radius) {
		return fmt.Errorf("parametric curve radius must be finite and > 0")
	}
	return nil
}

func (c *ParametricCurve) Name() string {
	return "Parametric Curve"
}

func (c *ParametricCurve) IntersectAffine(raySt, rayDir *mat.VecDense, options IntersectOptions) (SurfaceInteraction, bool) {
	if !options.valid() {
		return SurfaceInteraction{}, false
	}
	hit, ok := c.intersect(raySt, rayDir, options.Range.Min, options.Range.Max)
	if !ok {
		return SurfaceInteraction{}, false
	}
	return c.interactionAt(raySt, rayDir, hit), true
}

func (c *ParametricCurve) IntersectGeodesic(_, _ *mat.VecDense, _ geometry.Geometry, _ IntersectOptions) (SurfaceInteraction, bool) {
	return unsupportedGeodesicIntersection()
}

func (c *ParametricCurve) GetNormalVector(intersect, res *mat.VecDense) *mat.VecDense {
	if intersect == nil {
		return res
	}
	if res == nil || res.Len() != intersect.Len() {
		res = mat.NewVecDense(intersect.Len(), nil)
	} else {
		res.Zero()
	}

	centerT, ok := c.closestParameter(intersect)
	if !ok {
		return res
	}
	center := c.Function(centerT)
	if center == nil {
		return res
	}
	for axis := 0; axis < res.Len() && axis < center.Len(); axis++ {
		res.SetVec(axis, intersect.AtVec(axis)-center.AtVec(axis))
	}
	if mat.Norm(res, 2) <= utils.EPS {
		tangent := c.derivative(centerT)
		if tangent != nil && tangent.Len() >= 3 {
			res.SetVec(0, -tangent.AtVec(1))
			res.SetVec(1, tangent.AtVec(0))
			res.SetVec(2, 0)
		}
	}
	return maths.Normalize(res)
}

func (c *ParametricCurve) BuildBoundingBox() (pmin, pmax *mat.VecDense) {
	if c == nil || c.ensureAcceleration() != nil || c.cachedBounds == nil {
		return unboundedBoundingBox(3)
	}
	return c.cachedBounds.BuildBoundingBox()
}

func (c *ParametricCurve) BuildAcceleration() error {
	if c == nil {
		return fmt.Errorf("parametric curve is nil")
	}
	c.accelMu.Lock()
	defer c.accelMu.Unlock()
	return c.buildAccelerationLocked()
}

func (c *ParametricCurve) ensureAcceleration() error {
	if c == nil {
		return fmt.Errorf("parametric curve is nil")
	}
	c.accelMu.Lock()
	defer c.accelMu.Unlock()
	if c.cachedBounds != nil && c.segmentBVH != nil {
		return nil
	}
	return c.buildAccelerationLocked()
}

func (c *ParametricCurve) buildAccelerationLocked() error {
	if err := c.Validate(); err != nil {
		return err
	}

	samples, err := c.sampleCurve()
	if err != nil {
		return err
	}

	segments := make([]parametricCurveSegment, 0, len(samples)-1)
	var bounds *Cuboid
	for i := 0; i < len(samples)-1; i++ {
		segmentBounds, radiusMax, ok := c.segmentBounds(samples[i], samples[i+1])
		if !ok {
			continue
		}
		segment := parametricCurveSegment{
			T0:        samples[i].T,
			T1:        samples[i+1].T,
			P0:        samples[i].Point,
			P1:        samples[i+1].Point,
			RadiusMax: radiusMax,
			Bounds:    segmentBounds,
			SegmentID: i,
		}
		segments = append(segments, segment)
		bounds = unionParametricBoxes(bounds, segmentBounds)
	}
	if len(segments) == 0 || bounds == nil {
		return fmt.Errorf("parametric curve produced no finite segments")
	}

	c.segments = segments
	c.segmentBVH = buildParametricCurveBVH(c.segments)
	c.cachedBounds = bounds
	return nil
}

func (c *ParametricCurve) intersect(raySt, rayDir *mat.VecDense, tMin, tMax float64) (parametricCurveHit, bool) {
	if c == nil || raySt == nil || rayDir == nil || raySt.Len() != rayDir.Len() || raySt.Len() < 3 || tMax < tMin {
		return parametricCurveHit{}, false
	}
	if mat.Dot(rayDir, rayDir) <= 0 {
		return parametricCurveHit{}, false
	}
	if err := c.ensureAcceleration(); err != nil {
		return parametricCurveHit{}, false
	}
	if c.segmentBVH == nil {
		return parametricCurveHit{}, false
	}

	return c.intersectSegmentBVH(
		raySt,
		rayDir,
		c.segmentBVH,
		tMin,
		tMax,
		parametricCurveHit{Distance: math.MaxFloat64},
		false,
	)
}

func (c *ParametricCurve) intersectSegmentBVH(
	raySt, rayDir *mat.VecDense,
	node *parametricCurveBVHNode,
	tMin, tMax float64,
	best parametricCurveHit,
	found bool,
) (parametricCurveHit, bool) {
	if node == nil || node.Bounds == nil {
		return best, found
	}
	clipped, ok := node.Bounds.ClipAffine(raySt, rayDir, NewIntersectOptions(tMin, min(tMax, best.Distance)))
	if !ok {
		return best, found
	}
	near := clipped.Min
	if node.Segment != nil {
		if !node.Segment.overlapsCapsule(raySt, rayDir, tMin, min(tMax, best.Distance)) {
			return best, found
		}
		hit, ok := c.refineHitInterval(raySt, rayDir, node.Segment.T0, node.Segment.T1, tMin, min(tMax, best.Distance))
		if ok && hit.Distance < best.Distance {
			hit.T = maths.Clamp(hit.T, node.Segment.T0, node.Segment.T1)
			return hit, true
		}
		return best, found
	}

	leftNear, leftOK := curveNodeChildNear(raySt, rayDir, node.Left, tMin, min(tMax, best.Distance))
	rightNear, rightOK := curveNodeChildNear(raySt, rayDir, node.Right, tMin, min(tMax, best.Distance))
	if near > best.Distance {
		return best, found
	}
	if rightOK && (!leftOK || rightNear < leftNear) {
		best, found = c.intersectSegmentBVH(raySt, rayDir, node.Right, tMin, tMax, best, found)
		best, found = c.intersectSegmentBVH(raySt, rayDir, node.Left, tMin, tMax, best, found)
		return best, found
	}
	if leftOK {
		best, found = c.intersectSegmentBVH(raySt, rayDir, node.Left, tMin, tMax, best, found)
	}
	if rightOK {
		best, found = c.intersectSegmentBVH(raySt, rayDir, node.Right, tMin, tMax, best, found)
	}
	return best, found
}

func (c *ParametricCurve) refineHitInterval(raySt, rayDir *mat.VecDense, left, right, tMin, tMax float64) (parametricCurveHit, bool) {
	if right < left {
		return parametricCurveHit{}, false
	}
	phi := 0.5 * (math.Sqrt(5) - 1)
	x1 := right - phi*(right-left)
	x2 := left + phi*(right-left)
	f1 := c.raySphereDistanceOrInf(raySt, rayDir, x1, tMin, tMax)
	f2 := c.raySphereDistanceOrInf(raySt, rayDir, x2, tMin, tMax)

	for i := 0; i < c.refineIter(); i++ {
		if f1 > f2 {
			left = x1
			x1 = x2
			f1 = f2
			x2 = left + phi*(right-left)
			f2 = c.raySphereDistanceOrInf(raySt, rayDir, x2, tMin, tMax)
		} else {
			right = x2
			x2 = x1
			f2 = f1
			x1 = right - phi*(right-left)
			f1 = c.raySphereDistanceOrInf(raySt, rayDir, x1, tMin, tMax)
		}
	}

	bestT, bestDistance := x1, f1
	if f2 < bestDistance {
		bestT, bestDistance = x2, f2
	}
	for _, t := range []float64{left, right, midpoint([2]float64{left, right})} {
		if distance, ok := c.raySphereDistanceAt(raySt, rayDir, t, tMin, tMax); ok && distance < bestDistance {
			bestT, bestDistance = t, distance
		}
	}
	if !distanceInRange(bestDistance, tMin, tMax) {
		return parametricCurveHit{}, false
	}
	return parametricCurveHit{Distance: bestDistance, T: bestT}, true
}

func (c *ParametricCurve) raySphereDistanceOrInf(raySt, rayDir *mat.VecDense, curveT, tMin, tMax float64) float64 {
	distance, ok := c.raySphereDistanceAt(raySt, rayDir, curveT, tMin, tMax)
	if !ok {
		return math.Inf(1)
	}
	return distance
}

func (c *ParametricCurve) raySphereDistanceAt(raySt, rayDir *mat.VecDense, curveT, tMin, tMax float64) (float64, bool) {
	center := c.Function(curveT)
	if center == nil || center.Len() < 3 || !finiteVec(center, 3) {
		return 0, false
	}
	radius := c.Radius(curveT)
	if radius <= 0 || !maths.IsFinite(radius) {
		return 0, false
	}

	dd := mat.Dot(rayDir, rayDir)
	if dd <= 0 {
		return 0, false
	}
	qd := 0.0
	q2 := 0.0
	for axis := 0; axis < 3; axis++ {
		q := center.AtVec(axis) - raySt.AtVec(axis)
		qd += q * rayDir.AtVec(axis)
		q2 += q * q
	}
	perp2 := q2 - qd*qd/dd
	if perp2 < 0 && perp2 > -1e-10 {
		perp2 = 0
	}
	disc := radius*radius - perp2
	if disc < 0 {
		return 0, false
	}

	projection := qd / dd
	offset := math.Sqrt(disc / dd)
	sIn := projection - offset
	sOut := projection + offset
	if sOut < tMin || sIn > tMax {
		return 0, false
	}
	if sIn < tMin {
		return tMin, true
	}
	if !distanceInRange(sIn, tMin, tMax) {
		return 0, false
	}
	return sIn, true
}

func (c *ParametricCurve) interactionAt(raySt, rayDir *mat.VecDense, hit parametricCurveHit) SurfaceInteraction {
	point := affinePointAt(raySt, rayDir, hit.Distance)
	center := c.Function(hit.T)
	normal := mat.NewVecDense(point.Len(), nil)
	if center != nil {
		for axis := 0; axis < normal.Len() && axis < center.Len(); axis++ {
			normal.SetVec(axis, point.AtVec(axis)-center.AtVec(axis))
		}
	}
	if mat.Norm(normal, 2) <= utils.EPS {
		normal = c.GetNormalVector(point, normal)
	} else {
		maths.Normalize(normal)
	}
	dpdt := c.derivative(hit.T)
	if dpdt != nil && dpdt.Len() != point.Len() {
		expanded := mat.NewVecDense(point.Len(), nil)
		for axis := 0; axis < expanded.Len() && axis < dpdt.Len(); axis++ {
			expanded.SetVec(axis, dpdt.AtVec(axis))
		}
		dpdt = expanded
	}
	return SurfaceInteraction{
		Distance:        hit.Distance,
		Point:           point,
		GeometricNormal: normal,
		ShadingNormal:   normal,
		UV:              [2]float64{normalizeRange(hit.T, c.TRange), 0},
		DPDU:            dpdt,
		PrimitiveID:     int(math.Round(normalizeRange(hit.T, c.TRange) * float64(c.samples()))),
	}
}

func (c *ParametricCurve) derivative(t float64) *mat.VecDense {
	if c == nil || c.Function == nil {
		return nil
	}
	if c.Derivative != nil {
		return c.Derivative(t, mat.NewVecDense(3, nil))
	}
	eps := c.derivativeEps()
	left := max(c.TRange[0], t-eps)
	right := min(c.TRange[1], t+eps)
	if right <= left {
		return nil
	}
	pLeft := c.Function(left)
	pRight := c.Function(right)
	if pLeft == nil || pRight == nil || pLeft.Len() < 3 || pRight.Len() < 3 {
		return nil
	}
	result := mat.NewVecDense(3, nil)
	for axis := 0; axis < 3; axis++ {
		result.SetVec(axis, (pRight.AtVec(axis)-pLeft.AtVec(axis))/(right-left))
	}
	return result
}

func (c *ParametricCurve) closestParameter(point *mat.VecDense) (float64, bool) {
	if c == nil || c.Function == nil || point == nil {
		return 0, false
	}
	bestDistance := math.MaxFloat64
	bestT := c.TRange[0]
	for i := 0; i <= c.samples(); i++ {
		t := maths.Lerp(c.TRange[0], c.TRange[1], float64(i)/float64(c.samples()))
		center := c.Function(t)
		if center == nil || center.Len() < 3 || !finiteVec(center, 3) {
			continue
		}
		d2 := 0.0
		for axis := 0; axis < 3; axis++ {
			d := center.AtVec(axis) - point.AtVec(axis)
			d2 += d * d
		}
		if d2 < bestDistance {
			bestDistance = d2
			bestT = t
		}
	}
	return bestT, bestDistance < math.MaxFloat64
}

func (c *ParametricCurve) sampledBounds() *Cuboid {
	if c == nil || c.ensureAcceleration() != nil || c.cachedBounds == nil {
		pmin, pmax := unboundedBoundingBox(3)
		return NewCuboid(pmin, pmax)
	}
	return c.cachedBounds
}

func (c *ParametricCurve) sampleCurve() ([]parametricCurveSample, error) {
	sampleCount := c.samples()
	samples := make([]parametricCurveSample, 0, sampleCount+1)
	for i := 0; i <= sampleCount; i++ {
		t := maths.Lerp(c.TRange[0], c.TRange[1], float64(i)/float64(sampleCount))
		point := c.Function(t)
		if point == nil || point.Len() < 3 || !finiteVec(point, 3) {
			return nil, fmt.Errorf("parametric curve sample %d produced a non-finite 3D point", i)
		}
		radius := c.Radius(t)
		if radius <= 0 || !maths.IsFinite(radius) {
			return nil, fmt.Errorf("parametric curve sample %d produced invalid radius", i)
		}
		samples = append(samples, parametricCurveSample{
			T:      t,
			Point:  mat.VecDenseCopyOf(point),
			Radius: radius,
		})
	}
	return samples, nil
}

func (c *ParametricCurve) segmentBounds(a, b parametricCurveSample) (*Cuboid, float64, bool) {
	midT := 0.5 * (a.T + b.T)
	midPoint := c.Function(midT)
	midRadius := c.Radius(midT)
	if midPoint == nil || midPoint.Len() < 3 || !finiteVec(midPoint, 3) || midRadius <= 0 || !maths.IsFinite(midRadius) {
		return nil, 0, false
	}

	sagitta := pointSegmentDistance(midPoint, a.Point, b.Point)
	radiusMax := math.Max(math.Max(a.Radius, b.Radius), midRadius) + sagitta + c.boundsPadding()
	points := []*mat.VecDense{a.Point, b.Point, midPoint}

	pmin := []float64{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}
	pmax := []float64{-math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64}
	for _, point := range points {
		for axis := 0; axis < 3; axis++ {
			pmin[axis] = math.Min(pmin[axis], point.AtVec(axis)-radiusMax)
			pmax[axis] = math.Max(pmax[axis], point.AtVec(axis)+radiusMax)
		}
	}
	return NewCuboid(mat.NewVecDense(3, pmin), mat.NewVecDense(3, pmax)), radiusMax, true
}

func buildParametricCurveBVH(segments []parametricCurveSegment) *parametricCurveBVHNode {
	if len(segments) == 0 {
		return nil
	}
	if len(segments) == 1 {
		segment := segments[0]
		return &parametricCurveBVHNode{
			Bounds:  cloneParametricBox(segment.Bounds),
			Segment: &segment,
		}
	}

	bounds := unionParametricCurveSegmentBounds(segments)
	axis := largestParametricCurveCentroidExtent(segments)
	sort.Slice(segments, func(i, j int) bool {
		return parametricCurveSegmentCentroid(segments[i], axis) < parametricCurveSegmentCentroid(segments[j], axis)
	})
	mid := len(segments) / 2
	leftSegments := append([]parametricCurveSegment(nil), segments[:mid]...)
	rightSegments := append([]parametricCurveSegment(nil), segments[mid:]...)
	return &parametricCurveBVHNode{
		Bounds: bounds,
		Left:   buildParametricCurveBVH(leftSegments),
		Right:  buildParametricCurveBVH(rightSegments),
	}
}

func unionParametricCurveSegmentBounds(segments []parametricCurveSegment) *Cuboid {
	var bounds *Cuboid
	for _, segment := range segments {
		bounds = unionParametricBoxes(bounds, segment.Bounds)
	}
	return bounds
}

func largestParametricCurveCentroidExtent(segments []parametricCurveSegment) int {
	bestAxis := 0
	bestExtent := math.Inf(-1)
	for axis := 0; axis < 3; axis++ {
		minValue := math.Inf(1)
		maxValue := math.Inf(-1)
		for _, segment := range segments {
			center := parametricCurveSegmentCentroid(segment, axis)
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

func parametricCurveSegmentCentroid(segment parametricCurveSegment, axis int) float64 {
	if segment.Bounds == nil {
		return 0
	}
	return 0.5 * (segment.Bounds.Pmin.AtVec(axis) + segment.Bounds.Pmax.AtVec(axis))
}

func curveNodeChildNear(raySt, rayDir *mat.VecDense, node *parametricCurveBVHNode, tMin, tMax float64) (float64, bool) {
	if node == nil || node.Bounds == nil {
		return 0, false
	}
	clipped, ok := node.Bounds.ClipAffine(raySt, rayDir, NewIntersectOptions(tMin, tMax))
	return clipped.Min, ok
}

func (s *parametricCurveSegment) overlapsCapsule(raySt, rayDir *mat.VecDense, tMin, tMax float64) bool {
	if s == nil || s.P0 == nil || s.P1 == nil || s.RadiusMax <= 0 {
		return false
	}
	d2 := constrainedRaySegmentDistanceSquared(raySt, rayDir, s.P0, s.P1, tMin, tMax)
	radius := s.RadiusMax + utils.EPS
	return d2 <= radius*radius
}

func constrainedRaySegmentDistanceSquared(raySt, rayDir, a, b *mat.VecDense, tMin, tMax float64) float64 {
	best := math.Inf(1)
	addCandidate := func(rayT, segmentU float64) {
		if !maths.IsFinite(rayT) || rayT < tMin || rayT > tMax || segmentU < 0 || segmentU > 1 {
			return
		}
		d2 := raySegmentParamsDistanceSquared(raySt, rayDir, a, b, rayT, segmentU)
		if d2 < best {
			best = d2
		}
	}

	dd := mat.Dot(rayDir, rayDir)
	if dd <= 0 {
		return best
	}

	edge := mat.NewVecDense(3, nil)
	edge.SubVec(b, a)
	ee := mat.Dot(edge, edge)
	if ee <= utils.EPS {
		rayT := maths.Clamp(pointRayProjection(a, raySt, rayDir, dd), tMin, tMax)
		addCandidate(rayT, 0)
		return best
	}

	w0 := mat.NewVecDense(3, nil)
	w0.SubVec(raySt, a)
	de := mat.Dot(rayDir, edge)
	dw := mat.Dot(rayDir, w0)
	ew := mat.Dot(edge, w0)
	denominator := dd*ee - de*de
	if math.Abs(denominator) > utils.EPS {
		rayT := (de*ew - ee*dw) / denominator
		segmentU := (dd*ew - de*dw) / denominator
		addCandidate(rayT, segmentU)
		addCandidate(maths.Clamp(rayT, tMin, tMax), maths.Clamp(segmentU, 0, 1))
	}

	for _, segmentU := range []float64{0, 1} {
		point := pointOnSegment(a, b, segmentU)
		rayT := maths.Clamp(pointRayProjection(point, raySt, rayDir, dd), tMin, tMax)
		addCandidate(rayT, segmentU)
	}

	if maths.IsFinite(tMin) {
		point := affinePointAt(raySt, rayDir, tMin)
		segmentU := maths.Clamp(pointSegmentProjection(point, a, b, ee), 0, 1)
		addCandidate(tMin, segmentU)
	}
	if maths.IsFinite(tMax) && tMax < math.MaxFloat64/4 {
		point := affinePointAt(raySt, rayDir, tMax)
		segmentU := maths.Clamp(pointSegmentProjection(point, a, b, ee), 0, 1)
		addCandidate(tMax, segmentU)
	}

	return best
}

func raySegmentParamsDistanceSquared(raySt, rayDir, a, b *mat.VecDense, rayT, segmentU float64) float64 {
	d2 := 0.0
	for axis := 0; axis < 3; axis++ {
		rayValue := raySt.AtVec(axis) + rayT*rayDir.AtVec(axis)
		segmentValue := a.AtVec(axis) + segmentU*(b.AtVec(axis)-a.AtVec(axis))
		d := rayValue - segmentValue
		d2 += d * d
	}
	return d2
}

func pointRayProjection(point, raySt, rayDir *mat.VecDense, dd float64) float64 {
	qd := 0.0
	for axis := 0; axis < 3; axis++ {
		qd += (point.AtVec(axis) - raySt.AtVec(axis)) * rayDir.AtVec(axis)
	}
	return qd / dd
}

func pointSegmentProjection(point, a, b *mat.VecDense, ee float64) float64 {
	dot := 0.0
	for axis := 0; axis < 3; axis++ {
		dot += (point.AtVec(axis) - a.AtVec(axis)) * (b.AtVec(axis) - a.AtVec(axis))
	}
	return dot / ee
}

func pointSegmentDistance(point, a, b *mat.VecDense) float64 {
	edge := mat.NewVecDense(3, nil)
	edge.SubVec(b, a)
	ee := mat.Dot(edge, edge)
	if ee <= utils.EPS {
		return math.Sqrt(raySegmentParamsDistanceSquared(a, edge, point, point, 0, 0))
	}
	u := maths.Clamp(pointSegmentProjection(point, a, b, ee), 0, 1)
	return math.Sqrt(raySegmentParamsDistanceSquared(point, edge, a, b, 0, u))
}

func pointOnSegment(a, b *mat.VecDense, u float64) *mat.VecDense {
	point := mat.NewVecDense(3, nil)
	for axis := 0; axis < 3; axis++ {
		point.SetVec(axis, a.AtVec(axis)+u*(b.AtVec(axis)-a.AtVec(axis)))
	}
	return point
}

func (c *ParametricCurve) samples() int {
	if c != nil && c.Samples >= 2 {
		return c.Samples
	}
	return defaultParametricCurveSamples
}

func (c *ParametricCurve) refineIter() int {
	if c != nil && c.RefineIter > 0 {
		return c.RefineIter
	}
	return defaultParametricCurveRefineIter
}

func (c *ParametricCurve) derivativeEps() float64 {
	if c != nil && c.DerivativeEps > 0 {
		return c.DerivativeEps
	}
	return defaultParametricCurveDerivativeEps
}

func (c *ParametricCurve) boundsPadding() float64 {
	if c != nil && c.BoundsPadding >= 0 {
		return c.BoundsPadding
	}
	return defaultParametricCurveBoundsPadding
}
