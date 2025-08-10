package object

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"sort"
)

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
		node.BoundBox = NewCuboid(obj.Shape.BuildBoundingBox())
	}
	return node
}

// ObjectTree 管理场景中的物体层次结构
type ObjectTree struct {
	Root        *ObjectNode
	Objects     []*Object
	ObjectNodes []*ObjectNode
}

// 向量操作工具函数
func vecMin(a, b *mat.VecDense) *mat.VecDense {
	return mat.NewVecDense(3, []float64{
		math.Min(a.AtVec(0), b.AtVec(0)),
		math.Min(a.AtVec(1), b.AtVec(1)),
		math.Min(a.AtVec(2), b.AtVec(2)),
	})
}

func vecMax(a, b *mat.VecDense) *mat.VecDense {
	return mat.NewVecDense(3, []float64{
		math.Max(a.AtVec(0), b.AtVec(0)),
		math.Max(a.AtVec(1), b.AtVec(1)),
		math.Max(a.AtVec(2), b.AtVec(2)),
	})
}

func vecSub(a, b *mat.VecDense) *mat.VecDense {
	return mat.NewVecDense(3, []float64{
		a.AtVec(0) - b.AtVec(0),
		a.AtVec(1) - b.AtVec(1),
		a.AtVec(2) - b.AtVec(2),
	})
}

// Add 添加新物体到场景
func (t *ObjectTree) AddObject(object *Object) *Object {
	t.Objects = append(t.Objects, object)
	return t.Objects[len(t.Objects)-1]
}

// Build 构建对象树
func (t *ObjectTree) Build() *ObjectTree {
	// Build Leaf Nodes
	t.ObjectNodes = []*ObjectNode{}
	for i := range t.Objects {
		node := NewObjectNode(t.Objects[i], nil, nil)
		t.ObjectNodes = append(t.ObjectNodes, node)
	}

	t.build(0, len(t.ObjectNodes)-1, &t.Root)
	return t
}

// build 递归构建树结构
func (t *ObjectTree) build(l, r int, node **ObjectNode) {
	if l == r {
		*node = t.ObjectNodes[l]
		return
	}

	*node = NewObjectNode(nil, nil, nil)
	pmin := t.ObjectNodes[l].BoundBox.Pmin
	pmax := t.ObjectNodes[l].BoundBox.Pmax
	delta := mat.NewVecDense(3, []float64{0, 0, 0})

	for i := l + 1; i <= r; i++ {
		box := t.ObjectNodes[i].BoundBox
		pmin = vecMin(pmin, box.Pmin)
		pmax = vecMax(pmax, box.Pmax)
		boxDelta := mat.NewVecDense(3, nil)
		boxDelta.SubVec(box.Pmax, box.Pmin)

		// 计算最大维度差
		for d := 0; d < 3; d++ {
			if boxDelta.AtVec(d) > delta.AtVec(d) {
				delta.SetVec(d, boxDelta.AtVec(d))
			}
		}
	}

	(*node).BoundBox = &Cuboid{Pmin: pmin, Pmax: pmax}
	size := vecSub(pmax, pmin)
	dimRatios := make([]float64, 3)
	for d := 0; d < 3; d++ {
		dimRatios[d] = delta.AtVec(d) / size.AtVec(d)
	}

	// 选择最大扩展维度
	dim := 0
	if dimRatios[1] > dimRatios[0] {
		dim = 1
	}
	if dimRatios[2] > dimRatios[0] && dimRatios[2] > dimRatios[1] {
		dim = 2
	}

	// 按选定维度排序
	sort.Slice(t.ObjectNodes[l:r+1], func(i, j int) bool {
		a := t.ObjectNodes[l+i].BoundBox.Pmin
		b := t.ObjectNodes[l+j].BoundBox.Pmin
		if a.AtVec(dim) != b.AtVec(dim) {
			return a.AtVec(dim) < b.AtVec(dim)
		}
		return t.ObjectNodes[l+i].BoundBox.Pmax.AtVec(dim) < t.ObjectNodes[l+j].BoundBox.Pmax.AtVec(dim)
	})

	mid := (l + r) / 2
	t.build(l, mid, &(*node).Children[0])
	t.build(mid+1, r, &(*node).Children[1])
}

// GetIntersection 查找光线与物体的交点
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

	if dis1 < dis2 {
		return dis1, obj1
	}
	return dis2, obj2
}
