package object

import (
	"fmt"
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
	"math"
	"sort"
	"strings"
)

const binnedSAHBinCount = 12

type BVHUpdateStrategy string

const (
	BVHUpdateAuto    BVHUpdateStrategy = "auto"
	BVHUpdateRefit   BVHUpdateStrategy = "refit"
	BVHUpdateRebuild BVHUpdateStrategy = "rebuild"
)

type ObjectTree struct {
	Root        *ObjectNode
	Objects     []*Object
	ObjectNodes []*ObjectNode
	Media       *medium.Registry
}

type SurfaceHit struct {
	Distance        float64
	Point           *mat.VecDense
	GeometricNormal *mat.VecDense
	ShadingNormal   *mat.VecDense
	FrontFace       bool
	Object          *Object
}

func (t *ObjectTree) AddObject(object *Object) *Object {
	t.Objects = append(t.Objects, object)
	return t.Objects[len(t.Objects)-1]
}

// Build constructs the object tree.
func (t *ObjectTree) Build() *ObjectTree {
	leaves := make([]*ObjectNode, 0, len(t.Objects))
	for i := range t.Objects {
		node := NewObjectNode(t.Objects[i], nil, nil)
		leaves = append(leaves, node)
	}

	t.ObjectNodes = leaves
	t.Root = t.build(leaves)
	return t
}

// Rebuild reconstructs BVH topology and leaf bounds from all objects.
func (t *ObjectTree) Rebuild() *ObjectTree {
	return t.Build()
}

// Refit refreshes leaf bounds and propagates merged bounds through existing topology.
func (t *ObjectTree) Refit() *ObjectTree {
	refitNode(t.Root)
	return t
}

// Update applies the requested dynamic-update strategy.
func (t *ObjectTree) Update(strategy BVHUpdateStrategy) *ObjectTree {
	switch strategy {
	case BVHUpdateRefit:
		return t.Refit()
	case BVHUpdateRebuild:
		return t.Rebuild()
	default:
		if t.Root == nil || leafCount(t.Root) != len(t.Objects) {
			return t.Rebuild()
		}
		return t.Refit()
	}
}

// build recursively constructs BVH topology using binned SAH splits.
func (t *ObjectTree) build(nodes []*ObjectNode) *ObjectNode {
	if len(nodes) == 0 {
		return nil
	}
	if len(nodes) == 1 {
		return nodes[0]
	}

	bounds := mergeNodeBounds(nodes)
	dim, splitBin, ok := chooseBinnedSAHSplit(nodes, bounds)
	if !ok {
		dim = largestCentroidExtentDimension(nodes)
		sortNodesByCentroid(nodes, dim)
		mid := len(nodes) / 2
		return t.newInternalNode(bounds, t.build(nodes[:mid]), t.build(nodes[mid:]))
	}

	mid := partitionByBin(nodes, dim, splitBin)
	if mid <= 0 || mid >= len(nodes) {
		sortNodesByCentroid(nodes, dim)
		mid = len(nodes) / 2
	}
	return t.newInternalNode(bounds, t.build(nodes[:mid]), t.build(nodes[mid:]))
}

func (t *ObjectTree) newInternalNode(bounds *shape.Cuboid, left, right *ObjectNode) *ObjectNode {
	node := NewObjectNode(nil, left, right)
	node.BoundBox = bounds
	t.ObjectNodes = append(t.ObjectNodes, node)
	return node
}

// GetIntersection finds the intersection between a ray and an object.
func (t *ObjectTree) GetIntersection(raySt, rayDir *mat.VecDense, node *ObjectNode) (float64, *Object) {
	if node == nil {
		return math.MaxFloat64, nil
	} else if node.Obj != nil {
		return node.Obj.Shape.Intersect(raySt, rayDir), node.Obj
	} else if node.BoundBox.Intersect(raySt, rayDir) >= math.MaxFloat64 {
		return math.MaxFloat64, nil
	}

	dis1, obj1 := t.GetIntersection(raySt, rayDir, node.Children[0])
	dis2, obj2 := t.GetIntersection(raySt, rayDir, node.Children[1])

	// Avoid the emitted light hitting oneself back and forth within the precision of the contact surface.
	if dis1 < utils.EPS {
		dis1 = math.MaxFloat64
	}
	if dis2 < utils.EPS {
		dis2 = math.MaxFloat64
	}

	if dis1 < dis2 {
		return dis1, obj1
	}
	return dis2, obj2
}

func (t *ObjectTree) GetSurfaceHit(raySt, rayDir *mat.VecDense) (*SurfaceHit, bool) {
	distance, obj := t.GetIntersection(raySt, rayDir, t.Root)
	if distance >= math.MaxFloat64 || obj == nil {
		return nil, false
	}

	point := mat.NewVecDense(raySt.Len(), nil)
	point.AddVec(raySt, math_lib.ScaleVec2(distance, rayDir))

	geometricNormal := obj.Shape.GetNormalVector(point, mat.NewVecDense(point.Len(), nil))
	math_lib.Normalize(geometricNormal)

	frontFace := mat.Dot(geometricNormal, rayDir) < 0
	shadingNormal := mat.VecDenseCopyOf(geometricNormal)
	if !frontFace {
		shadingNormal.ScaleVec(-1, shadingNormal)
	}

	return &SurfaceHit{
		Distance:        distance,
		Point:           point,
		GeometricNormal: geometricNormal,
		ShadingNormal:   shadingNormal,
		FrontFace:       frontFace,
		Object:          obj,
	}, true
}

func (ot *ObjectTree) TreeString() string {
	if ot.Root == nil {
		return "ObjectTree is empty"
	}

	var sb strings.Builder
	sb.WriteString("ObjectTree Structure:\n")
	sb.WriteString(ot.Root.TreeNodeString(0))

	// Add orphan objects that are not in the tree.
	if len(ot.Objects) > 0 {
		sb.WriteString("\n\nOrphan Objects (not in tree):")
		for _, obj := range ot.Objects {
			sb.WriteString(fmt.Sprintf("\n  - %s", obj.Shape.Name()))
		}
	}

	return sb.String()
}
