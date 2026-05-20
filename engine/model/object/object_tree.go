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
	UV              [2]float64
	DPDU            *mat.VecDense
	DPDV            *mat.VecDense
	PrimitiveID     int
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
		node.PrimitiveID = i
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

type bvhBin struct {
	Count int
	Box   *shape.Cuboid
}

func refitNode(node *ObjectNode) *shape.Cuboid {
	if node == nil {
		return nil
	}
	if node.Obj != nil {
		if node.Obj.Shape == nil {
			node.BoundBox = nil
			return nil
		}
		node.BoundBox = shape.NewCuboid(node.Obj.Shape.BuildBoundingBox())
		return node.BoundBox
	}

	left := refitNode(node.Children[0])
	right := refitNode(node.Children[1])
	node.BoundBox = unionBoxes(left, right)
	return node.BoundBox
}

func leafCount(node *ObjectNode) int {
	if node == nil {
		return 0
	}
	if node.Obj != nil {
		return 1
	}
	return leafCount(node.Children[0]) + leafCount(node.Children[1])
}

func chooseBinnedSAHSplit(nodes []*ObjectNode, bounds *shape.Cuboid) (int, int, bool) {
	if len(nodes) < 2 || bounds == nil {
		return 0, 0, false
	}

	parentArea := surfaceArea(bounds)
	if parentArea <= 0 || math.IsInf(parentArea, 0) || math.IsNaN(parentArea) {
		return 0, 0, false
	}

	bestCost := math.Inf(1)
	bestDim := 0
	bestSplitBin := 0

	for dim := 0; dim < bounds.Pmin.Len(); dim++ {
		minCenter, maxCenter := centroidRange(nodes, dim)
		if maxCenter-minCenter <= 1e-12 {
			continue
		}

		bins := make([]bvhBin, binnedSAHBinCount)
		for _, node := range nodes {
			binIndex := centroidBin(node, dim, minCenter, maxCenter)
			bins[binIndex].Count++
			bins[binIndex].Box = unionBoxes(bins[binIndex].Box, node.BoundBox)
		}

		leftCounts, leftBoxes := prefixBins(bins)
		rightCounts, rightBoxes := suffixBins(bins)
		for splitBin := 0; splitBin < binnedSAHBinCount-1; splitBin++ {
			leftCount := leftCounts[splitBin]
			rightCount := rightCounts[splitBin+1]
			if leftCount == 0 || rightCount == 0 {
				continue
			}

			cost := 1 + (surfaceArea(leftBoxes[splitBin])*float64(leftCount)+
				surfaceArea(rightBoxes[splitBin+1])*float64(rightCount))/parentArea
			if cost < bestCost {
				bestCost = cost
				bestDim = dim
				bestSplitBin = splitBin
			}
		}
	}

	return bestDim, bestSplitBin, !math.IsInf(bestCost, 1)
}

func partitionByBin(nodes []*ObjectNode, dim, splitBin int) int {
	minCenter, maxCenter := centroidRange(nodes, dim)
	left, right := 0, len(nodes)-1
	for left <= right {
		if centroidBin(nodes[left], dim, minCenter, maxCenter) <= splitBin {
			left++
			continue
		}
		nodes[left], nodes[right] = nodes[right], nodes[left]
		right--
	}
	return left
}

func prefixBins(bins []bvhBin) ([]int, []*shape.Cuboid) {
	counts := make([]int, len(bins))
	boxes := make([]*shape.Cuboid, len(bins))
	var box *shape.Cuboid
	count := 0
	for i, bin := range bins {
		count += bin.Count
		box = unionBoxes(box, bin.Box)
		counts[i] = count
		boxes[i] = box
	}
	return counts, boxes
}

func suffixBins(bins []bvhBin) ([]int, []*shape.Cuboid) {
	counts := make([]int, len(bins))
	boxes := make([]*shape.Cuboid, len(bins))
	var box *shape.Cuboid
	count := 0
	for i := len(bins) - 1; i >= 0; i-- {
		count += bins[i].Count
		box = unionBoxes(box, bins[i].Box)
		counts[i] = count
		boxes[i] = box
	}
	return counts, boxes
}

func centroidBin(node *ObjectNode, dim int, minCenter, maxCenter float64) int {
	t := (centroid(node, dim) - minCenter) / (maxCenter - minCenter)
	bin := int(t * binnedSAHBinCount)
	if bin < 0 {
		return 0
	}
	if bin >= binnedSAHBinCount {
		return binnedSAHBinCount - 1
	}
	return bin
}

func centroidRange(nodes []*ObjectNode, dim int) (float64, float64) {
	minCenter := math.Inf(1)
	maxCenter := math.Inf(-1)
	for _, node := range nodes {
		c := centroid(node, dim)
		if c < minCenter {
			minCenter = c
		}
		if c > maxCenter {
			maxCenter = c
		}
	}
	return minCenter, maxCenter
}

func largestCentroidExtentDimension(nodes []*ObjectNode) int {
	if len(nodes) == 0 || nodes[0].BoundBox == nil {
		return 0
	}
	bestDim := 0
	bestExtent := math.Inf(-1)
	for dim := 0; dim < nodes[0].BoundBox.Pmin.Len(); dim++ {
		minCenter, maxCenter := centroidRange(nodes, dim)
		if extent := maxCenter - minCenter; extent > bestExtent {
			bestExtent = extent
			bestDim = dim
		}
	}
	return bestDim
}

func sortNodesByCentroid(nodes []*ObjectNode, dim int) {
	sort.Slice(nodes, func(i, j int) bool {
		a := nodes[i]
		b := nodes[j]
		aCenter := centroid(a, dim)
		bCenter := centroid(b, dim)
		if aCenter != bCenter {
			return aCenter < bCenter
		}
		if a.BoundBox.Pmin.AtVec(dim) != b.BoundBox.Pmin.AtVec(dim) {
			return a.BoundBox.Pmin.AtVec(dim) < b.BoundBox.Pmin.AtVec(dim)
		}
		return a.BoundBox.Pmax.AtVec(dim) < b.BoundBox.Pmax.AtVec(dim)
	})
}

func centroid(node *ObjectNode, dim int) float64 {
	if node == nil || node.BoundBox == nil {
		return 0
	}
	return (node.BoundBox.Pmin.AtVec(dim) + node.BoundBox.Pmax.AtVec(dim)) * 0.5
}

func mergeNodeBounds(nodes []*ObjectNode) *shape.Cuboid {
	var box *shape.Cuboid
	for _, node := range nodes {
		if node != nil {
			box = unionBoxes(box, node.BoundBox)
		}
	}
	return box
}

func unionBoxes(a, b *shape.Cuboid) *shape.Cuboid {
	if a == nil {
		return cloneBox(b)
	}
	if b == nil {
		return cloneBox(a)
	}

	dim := a.Pmin.Len()
	pmin := mat.NewVecDense(dim, nil)
	pmax := mat.NewVecDense(dim, nil)
	for i := 0; i < dim; i++ {
		pmin.SetVec(i, math.Min(a.Pmin.AtVec(i), b.Pmin.AtVec(i)))
		pmax.SetVec(i, math.Max(a.Pmax.AtVec(i), b.Pmax.AtVec(i)))
	}
	return shape.NewCuboid(pmin, pmax)
}

func cloneBox(box *shape.Cuboid) *shape.Cuboid {
	if box == nil {
		return nil
	}
	return shape.NewCuboid(mat.VecDenseCopyOf(box.Pmin), mat.VecDenseCopyOf(box.Pmax))
}

func surfaceArea(box *shape.Cuboid) float64 {
	if box == nil || box.Pmin == nil || box.Pmax == nil {
		return 0
	}
	dim := box.Pmin.Len()
	if dim == 0 {
		return 0
	}

	extents := make([]float64, dim)
	for i := 0; i < dim; i++ {
		extent := box.Pmax.AtVec(i) - box.Pmin.AtVec(i)
		if extent < 0 || math.IsNaN(extent) {
			return 0
		}
		extents[i] = extent
	}
	if dim == 1 {
		return extents[0]
	}

	area := 0.0
	for excluded := 0; excluded < dim; excluded++ {
		product := 1.0
		for i, extent := range extents {
			if i != excluded {
				product *= extent
			}
		}
		area += product
	}
	return 2 * area
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
	}
	if leftOK {
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
