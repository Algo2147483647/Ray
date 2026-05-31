package object

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
)

type SurfaceHit struct {
	Distance        float64
	ArcLength       float64
	Point           *mat.VecDense
	GeometricNormal *mat.VecDense
	ShadingNormal   *mat.VecDense
	UV              [2]float64
	DPDU            *mat.VecDense
	DPDV            *mat.VecDense
	PrimitiveID     int
	FrontFace       bool
	Object          *Object
}

// GetIntersection finds the intersection between a ray and an object.
func (t *ObjectTree) GetIntersection(raySt, rayDir *mat.VecDense, node *ObjectNode) (float64, *Object) {
	candidate, obj, ok := t.getSurfaceCandidate(raySt, rayDir, node, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64, nil
	}
	return candidate.Distance, obj
}

func (t *ObjectTree) GetSurfaceInteraction(raySt, rayDir *mat.VecDense, node *ObjectNode, tMin, tMax float64) (shape.SurfaceInteraction, *Object, bool) {
	candidate, obj, ok := t.getSurfaceCandidate(raySt, rayDir, node, tMin, tMax)
	if !ok {
		return shape.SurfaceInteraction{}, nil, false
	}
	interaction := shape.SurfaceInteractionFromCandidate(raySt, rayDir, candidate)
	if interaction.GeometricNormal == nil && obj != nil {
		interaction.GeometricNormal = obj.Shape.GetNormalVector(interaction.Point, mat.NewVecDense(interaction.Point.Len(), nil))
		interaction.ShadingNormal = interaction.GeometricNormal
	}
	return interaction, obj, true
}

func (t *ObjectTree) getSurfaceCandidate(raySt, rayDir *mat.VecDense, node *ObjectNode, tMin, tMax float64) (shape.SurfaceCandidate, *Object, bool) {
	if _, ok := nodeOverlapNear(raySt, rayDir, node, tMin, tMax); !ok {
		return shape.SurfaceCandidate{}, nil, false
	}
	return t.getSurfaceCandidateInOverlappedNode(raySt, rayDir, node, tMin, tMax)
}

func (t *ObjectTree) getSurfaceCandidateInOverlappedNode(raySt, rayDir *mat.VecDense, node *ObjectNode, tMin, tMax float64) (shape.SurfaceCandidate, *Object, bool) {
	if node.Obj != nil {
		candidate, ok := surfaceCandidate(raySt, rayDir, node.Obj.Shape, tMin, tMax)
		if !ok {
			return shape.SurfaceCandidate{}, nil, false
		}
		candidate.PrimitiveID = node.PrimitiveID
		return candidate, node.Obj, true
	}

	left := node.Children[0]
	right := node.Children[1]
	leftNear, leftOK := nodeOverlapNear(raySt, rayDir, left, tMin, tMax)
	rightNear, rightOK := nodeOverlapNear(raySt, rayDir, right, tMin, tMax)
	if !leftOK && !rightOK {
		return shape.SurfaceCandidate{}, nil, false
	}
	if rightOK && (!leftOK || rightNear < leftNear) {
		left, right = right, left
		leftNear, rightNear = rightNear, leftNear
		leftOK, rightOK = rightOK, leftOK
	}

	var (
		bestCandidate shape.SurfaceCandidate
		bestObj       *Object
		bestOK        bool
	)
	if leftOK && leftNear <= tMax {
		candidate, obj, ok := t.getSurfaceCandidateInOverlappedNode(raySt, rayDir, left, tMin, tMax)
		if ok {
			bestCandidate = candidate
			bestObj = obj
			bestOK = true
			tMax = candidate.Distance
		}
	}
	if rightOK && rightNear <= tMax {
		candidate, obj, ok := t.getSurfaceCandidateInOverlappedNode(raySt, rayDir, right, tMin, tMax)
		if ok && (!bestOK || candidate.Distance < bestCandidate.Distance) {
			bestCandidate = candidate
			bestObj = obj
			bestOK = true
		}
	}
	return bestCandidate, bestObj, bestOK
}

func nodeOverlapNear(raySt, rayDir *mat.VecDense, node *ObjectNode, tMin, tMax float64) (float64, bool) {
	if node == nil {
		return 0, false
	}
	if node.BoundBox == nil {
		return tMin, true
	}
	return node.BoundBox.OverlapRangeNear(raySt, rayDir, tMin, tMax)
}

func surfaceCandidate(raySt, rayDir *mat.VecDense, s shape.Shape, tMin, tMax float64) (shape.SurfaceCandidate, bool) {
	if provider, ok := s.(shape.SurfaceCandidateProvider); ok {
		return provider.IntersectCandidate(raySt, rayDir, tMin, tMax)
	}

	interaction, ok := s.IntersectRange(raySt, rayDir, tMin, tMax)
	if !ok {
		return shape.SurfaceCandidate{}, false
	}
	return shape.SurfaceCandidate{
		Distance:        interaction.Distance,
		Point:           interaction.Point,
		GeometricNormal: interaction.GeometricNormal,
		ShadingNormal:   interaction.ShadingNormal,
		UV:              interaction.UV,
		DPDU:            interaction.DPDU,
		DPDV:            interaction.DPDV,
		PrimitiveID:     interaction.PrimitiveID,
	}, true
}

func (t *ObjectTree) GetSurfaceHit(raySt, rayDir *mat.VecDense) (*SurfaceHit, bool) {
	return t.GetSurfaceHitRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
}

func (t *ObjectTree) GetSurfaceHitRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (*SurfaceHit, bool) {
	candidate, obj, ok := t.getSurfaceCandidate(raySt, rayDir, t.Root, tMin, tMax)
	if !ok || obj == nil {
		return nil, false
	}
	interaction := shape.SurfaceInteractionFromCandidate(raySt, rayDir, candidate)
	return newSurfaceHitFromInteraction(interaction, obj, rayDir), true
}

func (t *ObjectTree) GetSphericalSurfaceHit(raySt, rayDir *mat.VecDense, sMin, sMax float64) (*SurfaceHit, bool) {
	var (
		bestInteraction shape.SurfaceInteraction
		bestObj         *Object
		bestDirection   *mat.VecDense
		bestOK          bool
	)

	for _, obj := range t.Objects {
		if obj == nil || obj.Shape == nil {
			continue
		}
		provider, ok := obj.Shape.(shape.SphericalSurfaceCandidateProvider)
		if !ok {
			continue
		}
		candidate, ok := provider.IntersectSphericalCandidate(raySt, rayDir, sMin, sMax)
		if !ok {
			continue
		}
		interaction := shape.SurfaceInteractionFromCandidate(raySt, rayDir, candidate)
		arcLen := interaction.ArcLength
		if arcLen <= 0 {
			arcLen = interaction.Distance
		}
		if !bestOK || arcLen < bestInteraction.ArcLength {
			direction := sphericalDirectionAt(raySt, rayDir, arcLen)
			bestInteraction = interaction
			bestInteraction.Distance = arcLen
			bestInteraction.ArcLength = arcLen
			bestObj = obj
			bestDirection = direction
			bestOK = true
		}
	}

	if !bestOK {
		return nil, false
	}
	return newSurfaceHitFromInteraction(bestInteraction, bestObj, bestDirection), true
}

func newSurfaceHitFromInteraction(interaction shape.SurfaceInteraction, obj *Object, frontFaceDir *mat.VecDense) *SurfaceHit {
	geometricNormal := interaction.GeometricNormal
	if geometricNormal == nil {
		geometricNormal = obj.Shape.GetNormalVector(interaction.Point, mat.NewVecDense(interaction.Point.Len(), nil))
	}
	geometricNormal = mat.VecDenseCopyOf(geometricNormal)
	maths.Normalize(geometricNormal)

	frontFace := mat.Dot(geometricNormal, frontFaceDir) < 0
	shadingNormal := geometricNormal
	if !frontFace {
		shadingNormal = mat.VecDenseCopyOf(geometricNormal)
		shadingNormal.ScaleVec(-1, shadingNormal)
	}

	return &SurfaceHit{
		Distance:        interaction.Distance,
		ArcLength:       interaction.ArcLength,
		Point:           interaction.Point,
		GeometricNormal: geometricNormal,
		ShadingNormal:   shadingNormal,
		UV:              interaction.UV,
		DPDU:            interaction.DPDU,
		DPDV:            interaction.DPDV,
		PrimitiveID:     interaction.PrimitiveID,
		FrontFace:       frontFace,
		Object:          obj,
	}
}

func sphericalDirectionAt(raySt, rayDir *mat.VecDense, arcLen float64) *mat.VecDense {
	v := mat.NewVecDense(rayDir.Len(), nil)
	v.CopyVec(rayDir)
	v.AddScaledVec(v, -mat.Dot(v, raySt), raySt)
	n := mat.Norm(v, 2)
	if n == 0 {
		return v
	}
	v.ScaleVec(1/n, v)

	direction := mat.NewVecDense(rayDir.Len(), nil)
	direction.CopyVec(raySt)
	direction.ScaleVec(-math.Sin(arcLen), direction)
	direction.AddScaledVec(direction, math.Cos(arcLen), v)
	return maths.Normalize(direction)
}
