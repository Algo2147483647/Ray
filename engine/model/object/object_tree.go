package object

import (
	"fmt"
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"math"
	"sort"
	"strings"
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
	// Build Leaf Nodes
	t.ObjectNodes = []*ObjectNode{}
	for i := range t.Objects {
		node := NewObjectNode(t.Objects[i], nil, nil)
		t.ObjectNodes = append(t.ObjectNodes, node)
	}

	t.build(0, len(t.ObjectNodes)-1, &t.Root)

	//fmt.Println(t.TreeString())
	return t
}

// build recursively constructs the tree structure.
func (t *ObjectTree) build(l, r int, node **ObjectNode) {
	if l > r {
		return
	}

	if l == r {
		*node = t.ObjectNodes[l]
		return
	}

	// 1. Compute the merged bounding box for all objects.
	pmin := mat.NewVecDense(t.ObjectNodes[l].BoundBox.Pmin.Len(), nil)
	pmin.CopyVec(t.ObjectNodes[l].BoundBox.Pmin)
	pmax := mat.NewVecDense(t.ObjectNodes[l].BoundBox.Pmax.Len(), nil)
	pmax.CopyVec(t.ObjectNodes[l].BoundBox.Pmax)

	for i := l + 1; i <= r; i++ {
		box := t.ObjectNodes[i].BoundBox
		pmin = math_lib.MinVec(pmin, box.Pmin)
		pmax = math_lib.MaxVec(pmax, box.Pmax)
	}

	// 2. Select the split dimension.
	size := mat.NewVecDense(t.ObjectNodes[l].BoundBox.Pmax.Len(), nil)
	size.SubVec(pmax, pmin)
	dim := selectSplitDimension(t.ObjectNodes[l:r+1], size)

	// 3. Sort by the selected dimension.
	sort.Slice(t.ObjectNodes[l:r+1], func(i, j int) bool {
		a := t.ObjectNodes[l+i]
		b := t.ObjectNodes[l+j]

		// Primary comparison: center position.
		aCenter := (a.BoundBox.Pmin.AtVec(dim) + a.BoundBox.Pmax.AtVec(dim)) / 2
		bCenter := (b.BoundBox.Pmin.AtVec(dim) + b.BoundBox.Pmax.AtVec(dim)) / 2
		if aCenter != bCenter {
			return aCenter < bCenter
		}

		// Secondary comparison: bounding box minimum.
		if a.BoundBox.Pmin.AtVec(dim) != b.BoundBox.Pmin.AtVec(dim) {
			return a.BoundBox.Pmin.AtVec(dim) < b.BoundBox.Pmin.AtVec(dim)
		}

		// Final comparison: bounding box maximum.
		return a.BoundBox.Pmax.AtVec(dim) < b.BoundBox.Pmax.AtVec(dim)
	})

	// 4. Create an internal node and recursively build subtrees.
	*node = NewObjectNode(nil, nil, nil)
	(*node).BoundBox = &shape.Cuboid{Pmin: pmin, Pmax: pmax}

	mid := (l + r) / 2
	t.build(l, mid, &(*node).Children[0])
	t.build(mid+1, r, &(*node).Children[1])
	t.ObjectNodes = append(t.ObjectNodes, *node)
}

// selectSplitDimension chooses the split dimension based on center variance.
func selectSplitDimension(nodes []*ObjectNode, size *mat.VecDense) int {
	// Compute center variance for each dimension.
	var maxVariance = -math.MaxFloat64
	var bestDim = 0

	for dim := 0; dim < utils.Dimension; dim++ {
		if size.AtVec(dim) < 1e-8 { // Skip collapsed dimensions.
			continue
		}

		// Compute the mean center point.
		var sum float64
		for _, node := range nodes {
			center := (node.BoundBox.Pmin.AtVec(dim) + node.BoundBox.Pmax.AtVec(dim)) / 2
			sum += center
		}
		mean := sum / float64(len(nodes))

		// Compute variance.
		var variance float64
		for _, node := range nodes {
			center := (node.BoundBox.Pmin.AtVec(dim) + node.BoundBox.Pmax.AtVec(dim)) / 2
			diff := center - mean
			variance += diff * diff
		}
		variance /= float64(len(nodes))

		// Normalize variance by bounding box size.
		normalizedVariance := variance / size.AtVec(dim)

		if normalizedVariance > maxVariance {
			maxVariance = normalizedVariance
			bestDim = dim
		}
	}

	// If all dimensions are collapsed, default to the X axis.
	if maxVariance == -math.MaxFloat64 {
		return 0
	}
	return bestDim
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
