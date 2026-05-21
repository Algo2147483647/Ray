package object

import "github.com/Algo2147483647/ray/engine/model/shape"

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

func refitNode(node *ObjectNode) *shape.Cuboid {
	if node == nil {
		return nil
	} else if node.Obj != nil {
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
	} else if node.Obj != nil {
		return 1
	}
	return leafCount(node.Children[0]) + leafCount(node.Children[1])
}
