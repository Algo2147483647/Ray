package object

import (
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
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
	interaction, obj, ok := t.GetSurfaceInteraction(raySt, rayDir, node, utils.EPS, math.MaxFloat64)
	if !ok {
		return math.MaxFloat64, nil
	}
	return interaction.Distance, obj
}

func (t *ObjectTree) GetSurfaceInteraction(raySt, rayDir *mat.VecDense, node *ObjectNode, tMin, tMax float64) (shape.SurfaceInteraction, *Object, bool) {
	if node == nil {
		return shape.SurfaceInteraction{}, nil, false
	}
	if node.BoundBox != nil {
		if !node.BoundBox.OverlapsRange(raySt, rayDir, tMin, tMax) {
			return shape.SurfaceInteraction{}, nil, false
		}
	}
	if node.Obj != nil {
		interaction, ok := node.Obj.Shape.IntersectRange(raySt, rayDir, tMin, tMax)
		if !ok {
			return shape.SurfaceInteraction{}, nil, false
		}
		interaction.PrimitiveID = node.PrimitiveID
		return interaction, node.Obj, true
	}

	leftInteraction, leftObj, leftOK := t.GetSurfaceInteraction(raySt, rayDir, node.Children[0], tMin, tMax)
	if leftOK {
		tMax = leftInteraction.Distance
	}

	rightInteraction, rightObj, rightOK := t.GetSurfaceInteraction(raySt, rayDir, node.Children[1], tMin, tMax)
	if rightOK && (!leftOK || rightInteraction.Distance < leftInteraction.Distance) {
		return rightInteraction, rightObj, true
	} else if leftOK {
		return leftInteraction, leftObj, true
	}
	return shape.SurfaceInteraction{}, nil, false
}

func (t *ObjectTree) GetSurfaceHit(raySt, rayDir *mat.VecDense) (*SurfaceHit, bool) {
	interaction, obj, ok := t.GetSurfaceInteraction(raySt, rayDir, t.Root, utils.EPS, math.MaxFloat64)
	if !ok || obj == nil {
		return nil, false
	}

	geometricNormal := interaction.GeometricNormal
	if geometricNormal == nil {
		geometricNormal = obj.Shape.GetNormalVector(interaction.Point, mat.NewVecDense(interaction.Point.Len(), nil))
	}
	math_lib.Normalize(geometricNormal)

	frontFace := mat.Dot(geometricNormal, rayDir) < 0
	shadingNormal := mat.VecDenseCopyOf(geometricNormal)
	if !frontFace {
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
