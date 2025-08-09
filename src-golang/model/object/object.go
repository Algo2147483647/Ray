package object

// Object 表示场景中的物体
type Object struct {
	Shape    Shape     // 几何形状
	Material *Material // 材质属性
}

// ObjectNode 表示对象树中的节点
type ObjectNode struct {
	Obj      *Object        // 关联的物体
	BoundBox *Cuboid        // 包围盒
	Children [2]*ObjectNode // 子节点
}

// NewObjectNode 创建新节点
func NewObjectNode(obj *Object, left, right *ObjectNode) *ObjectNode {
	node := &ObjectNode{
		Obj:      obj,
		Children: [2]*ObjectNode{left, right},
	}

	if obj != nil {
		node.BuildComputeBoundingBox()
	}
	return node
}

// BuildComputeBoundingBox 构建包围盒
func (n *ObjectNode) BuildComputeBoundingBox() {
	n.BoundBox = &Cuboid{}
	pmax, pmin := n.Obj.Shape.BoundingBox()
	n.BoundBox.Pmax = pmax
	n.BoundBox.Pmin = pmin
}
