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
	candidate, obj, ok := t.getSurfaceCandidate(raySt, rayDir, t.Root, utils.EPS, math.MaxFloat64)
	if !ok || obj == nil {
		return nil, false
	}
	interaction := shape.SurfaceInteractionFromCandidate(raySt, rayDir, candidate)

	geometricNormal := interaction.GeometricNormal
	if geometricNormal == nil {
		geometricNormal = obj.Shape.GetNormalVector(interaction.Point, mat.NewVecDense(interaction.Point.Len(), nil))
	}
	geometricNormal = mat.VecDenseCopyOf(geometricNormal)
	maths.Normalize(geometricNormal)

	frontFace := mat.Dot(geometricNormal, rayDir) < 0
	shadingNormal := geometricNormal
	if !frontFace {
		shadingNormal = mat.VecDenseCopyOf(geometricNormal)
		shadingNormal.ScaleVec(-1, shadingNormal)
	}

	return &SurfaceHit{
		Distance:        interaction.Distance,
		Point:           interaction.Point,
		GeometricNormal: geometricNormal,
		ShadingNormal:   shadingNormal,
		UV:              interaction.UV,
		DPDU:            interaction.DPDU,
		DPDV:            interaction.DPDV,
		PrimitiveID:     interaction.PrimitiveID,
		FrontFace:       frontFace,
		Object:          obj,
	}, true
}
