package object

import (
	"github.com/Algo2147483647/ray/engine/maths/geometry"
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
	interaction, obj, ok := t.getClosestInteraction(raySt, rayDir, node, shape.NewIntersectOptions(utils.EPS, math.MaxFloat64))
	if !ok {
		return math.MaxFloat64, nil
	}
	return interaction.Distance, obj
}

func (t *ObjectTree) GetSurfaceInteraction(raySt, rayDir *mat.VecDense, node *ObjectNode, tMin, tMax float64) (shape.SurfaceInteraction, *Object, bool) {
	interaction, obj, ok := t.getClosestInteraction(raySt, rayDir, node, shape.NewIntersectOptions(tMin, tMax))
	if !ok {
		return shape.SurfaceInteraction{}, nil, false
	}
	if interaction.GeometricNormal == nil && obj != nil {
		interaction.GeometricNormal = obj.Shape.GetNormalVector(interaction.Point, mat.NewVecDense(interaction.Point.Len(), nil))
		interaction.ShadingNormal = interaction.GeometricNormal
	}
	return interaction, obj, true
}

func (t *ObjectTree) getClosestInteraction(raySt, rayDir *mat.VecDense, node *ObjectNode, options shape.IntersectOptions) (shape.SurfaceInteraction, *Object, bool) {
	if _, ok := nodeOverlapNear(raySt, rayDir, node, options); !ok {
		return shape.SurfaceInteraction{}, nil, false
	}
	return t.getClosestInteractionInOverlappedNode(raySt, rayDir, node, options)
}

func (t *ObjectTree) getClosestInteractionInOverlappedNode(raySt, rayDir *mat.VecDense, node *ObjectNode, options shape.IntersectOptions) (shape.SurfaceInteraction, *Object, bool) {
	if node.Obj != nil {
		interaction, ok := node.Obj.Shape.IntersectAffine(raySt, rayDir, options)
		if !ok {
			return shape.SurfaceInteraction{}, nil, false
		}
		interaction.PrimitiveID = node.PrimitiveID
		return interaction, node.Obj, true
	}

	left := node.Children[0]
	right := node.Children[1]
	leftNear, leftOK := nodeOverlapNear(raySt, rayDir, left, options)
	rightNear, rightOK := nodeOverlapNear(raySt, rayDir, right, options)
	if !leftOK && !rightOK {
		return shape.SurfaceInteraction{}, nil, false
	}
	if rightOK && (!leftOK || rightNear < leftNear) {
		left, right = right, left
		leftNear, rightNear = rightNear, leftNear
		leftOK, rightOK = rightOK, leftOK
	}

	var (
		bestInteraction shape.SurfaceInteraction
		bestObj         *Object
		bestOK          bool
	)
	if leftOK && leftNear <= options.Range.Max {
		interaction, obj, ok := t.getClosestInteractionInOverlappedNode(raySt, rayDir, left, options)
		if ok {
			bestInteraction = interaction
			bestObj = obj
			bestOK = true
			options.Range.Max = interaction.Distance
		}
	}
	if rightOK && rightNear <= options.Range.Max {
		interaction, obj, ok := t.getClosestInteractionInOverlappedNode(raySt, rayDir, right, options)
		if ok && (!bestOK || interaction.Distance < bestInteraction.Distance) {
			bestInteraction = interaction
			bestObj = obj
			bestOK = true
		}
	}
	return bestInteraction, bestObj, bestOK
}

func nodeOverlapNear(raySt, rayDir *mat.VecDense, node *ObjectNode, options shape.IntersectOptions) (float64, bool) {
	if node == nil {
		return 0, false
	}
	if node.BoundBox == nil {
		return options.Range.Min, true
	}
	clipped, ok := node.BoundBox.ClipAffine(raySt, rayDir, options)
	return clipped.Min, ok
}

func (t *ObjectTree) GetSurfaceHit(raySt, rayDir *mat.VecDense) (*SurfaceHit, bool) {
	return t.GetSurfaceHitRange(raySt, rayDir, utils.EPS, math.MaxFloat64)
}

func (t *ObjectTree) GetSurfaceHitRange(raySt, rayDir *mat.VecDense, tMin, tMax float64) (*SurfaceHit, bool) {
	return t.GetSurfaceHitRangeInGeometry(raySt, rayDir, geometry.Euclidean(), tMin, tMax)
}

func (t *ObjectTree) GetSurfaceHitRangeInGeometry(
	raySt, rayDir *mat.VecDense,
	g geometry.Geometry,
	tMin, tMax float64,
) (*SurfaceHit, bool) {
	interaction, obj, ok := t.getClosestInteraction(raySt, rayDir, t.Root, shape.NewIntersectOptions(tMin, tMax))
	if !ok || obj == nil {
		return nil, false
	}
	return newSurfaceHitFromInteraction(interaction, obj, rayDir, g), true
}

func (t *ObjectTree) GetGeodesicSurfaceHit(
	raySt, rayDir *mat.VecDense,
	g geometry.Geometry,
	paramMin, paramMax float64,
) (*SurfaceHit, bool) {
	if g == nil {
		return nil, false
	}
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
		interaction, ok := obj.Shape.IntersectGeodesic(
			raySt,
			rayDir,
			g,
			shape.NewIntersectOptions(paramMin, paramMax),
		)
		if !ok {
			continue
		}
		arcLen := interaction.ArcLength
		if arcLen <= 0 {
			arcLen = interaction.Distance
		}
		if !bestOK || arcLen < bestInteraction.ArcLength {
			direction := mat.NewVecDense(rayDir.Len(), nil)
			g.GeodesicDirection(raySt, rayDir, arcLen, direction)
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
	return newSurfaceHitFromInteraction(bestInteraction, bestObj, bestDirection, g), true
}

func newSurfaceHitFromInteraction(
	interaction shape.SurfaceInteraction,
	obj *Object,
	frontFaceDir *mat.VecDense,
	g geometry.Geometry,
) *SurfaceHit {
	g = geometry.Get(g)
	ambientGradient := interaction.GeometricNormal
	if ambientGradient == nil {
		ambientGradient = obj.Shape.GetNormalVector(interaction.Point, mat.NewVecDense(interaction.Point.Len(), nil))
	}
	geometricNormal := mat.NewVecDense(ambientGradient.Len(), nil)
	g.IntrinsicNormal(interaction.Point, ambientGradient, geometricNormal)
	normalizeGeometryVector(g, interaction.Point, geometricNormal)

	frontFace := g.InnerProduct(interaction.Point, geometricNormal, frontFaceDir) < 0
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

func normalizeGeometryVector(g geometry.Geometry, p, v *mat.VecDense) bool {
	n2 := g.InnerProduct(p, v, v)
	if n2 <= 0 || math.IsNaN(n2) || math.IsInf(n2, 0) {
		return false
	}
	v.ScaleVec(1/math.Sqrt(n2), v)
	return true
}
