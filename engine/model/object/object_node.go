package object

import (
	"fmt"
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"strings"
)

// ObjectNode represents a node in the object tree.
type ObjectNode struct {
	Obj      *Object        // Associated object
	BoundBox *shape.Cuboid  // Bounding box
	Children [2]*ObjectNode // Child nodes
}

// NewObjectNode creates a new node.
func NewObjectNode(obj *Object, left, right *ObjectNode) *ObjectNode {
	node := &ObjectNode{
		Obj:      obj,
		Children: [2]*ObjectNode{left, right},
	}

	if obj != nil {
		node.BoundBox = shape.NewCuboid(obj.Shape.BuildBoundingBox())
	}
	return node
}

// TreeNodeString returns the recursive string representation of an ObjectNode.
func (node *ObjectNode) TreeNodeString(depth int) string {
	if node == nil {
		return ""
	}

	// Create indentation for the current depth.
	indent := strings.Repeat("│   ", depth)
	nextIndent := strings.Repeat("│   ", depth+1)

	// Assemble the current node information.
	var nodeInfo strings.Builder
	if depth > 0 {
		nodeInfo.WriteString(fmt.Sprintf("%s├── ", indent[:len(indent)-4]))
	} else {
		nodeInfo.WriteString("Root: ")
	}

	// Add bounding box information.
	if node.BoundBox != nil {
		nodeInfo.WriteString(fmt.Sprintf("BoundBox: [%s → %s]",
			math_lib.FormatVec(node.BoundBox.Pmin),
			math_lib.FormatVec(node.BoundBox.Pmax)))
	} else {
		nodeInfo.WriteString("BoundBox: nil")
	}

	// Add associated object information.
	if node.Obj != nil && node.Obj.Shape != nil {
		nodeInfo.WriteString(fmt.Sprintf("  \t[Object: %s]", node.Obj.Shape.Name()))
	}

	// Process child nodes.
	children := make([]string, 0, 2)
	for i, child := range node.Children {
		childStr := child.TreeNodeString(depth + 1)
		if childStr != "" {
			// Add child node markers (left/right).
			marker := "L:"
			if i == 1 {
				marker = "R:"
			}
			children = append(children, fmt.Sprintf("%s%s %s", nextIndent, marker, childStr))
		}
	}

	// Combine node and child information.
	result := nodeInfo.String()
	if len(children) > 0 {
		result += "\n" + strings.Join(children, "\n")
	}
	return result
}
