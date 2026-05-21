package object

import (
	"github.com/Algo2147483647/ray/engine/model/shape"
)

// ObjectNode represents a node in the object tree.
type ObjectNode struct {
	Obj         *Object        // Associated object
	BoundBox    *shape.Cuboid  // Bounding box
	Children    [2]*ObjectNode // Child nodes
	PrimitiveID int
}

// NewObjectNode creates a new node.
func NewObjectNode(obj *Object, left, right *ObjectNode) *ObjectNode {
	node := &ObjectNode{
		Obj:         obj,
		Children:    [2]*ObjectNode{left, right},
		PrimitiveID: -1,
	}

	if obj != nil {
		node.BoundBox = shape.NewCuboid(obj.Shape.BuildBoundingBox())
	}
	return node
}
