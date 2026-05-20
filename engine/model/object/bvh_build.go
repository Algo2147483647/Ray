package object

import (
	"math"
	"sort"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
)

const binnedSAHBinCount = 12

type bvhBin struct {
	Count int
	Box   *shape.Cuboid
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
