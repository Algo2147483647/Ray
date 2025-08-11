package object

import (
	"fmt"
	"src-golang/model/object/shape"
	"src-golang/utils"
	"strings"
)

// ObjectNode 表示对象树中的节点
type ObjectNode struct {
	Obj      *Object        // 关联的物体
	BoundBox *shape.Cuboid  // 包围盒
	Children [2]*ObjectNode // 子节点
}

// NewObjectNode 创建新节点
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

// ObjectNode的递归字符串表示
func (node *ObjectNode) TreeNodeString(depth int) string {
	if node == nil {
		return ""
	}

	// 创建当前深度的缩进
	indent := strings.Repeat("│   ", depth)
	nextIndent := strings.Repeat("│   ", depth+1)

	// 组装当前节点信息
	var nodeInfo strings.Builder
	if depth > 0 {
		nodeInfo.WriteString(fmt.Sprintf("%s├── ", indent[:len(indent)-4]))
	} else {
		nodeInfo.WriteString("Root: ")
	}

	// 添加包围盒信息
	if node.BoundBox != nil {
		nodeInfo.WriteString(fmt.Sprintf("BoundBox: [%s → %s]",
			utils.FormatVec(node.BoundBox.Pmin),
			utils.FormatVec(node.BoundBox.Pmax)))
	} else {
		nodeInfo.WriteString("BoundBox: nil")
	}

	// 添加关联物体信息
	if node.Obj != nil && node.Obj.Shape != nil {
		nodeInfo.WriteString(fmt.Sprintf("  \t[Object: %s]", node.Obj.Shape.Name()))
	}

	// 处理子节点
	children := make([]string, 0, 2)
	for i, child := range node.Children {
		childStr := child.TreeNodeString(depth + 1)
		if childStr != "" {
			// 添加子节点标记（左/右）
			marker := "L:"
			if i == 1 {
				marker = "R:"
			}
			children = append(children, fmt.Sprintf("%s%s %s", nextIndent, marker, childStr))
		}
	}

	// 组合节点和子节点信息
	result := nodeInfo.String()
	if len(children) > 0 {
		result += "\n" + strings.Join(children, "\n")
	}
	return result
}
