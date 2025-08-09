package object

import (
	"gonum.org/v1/gonum/spatial/r3"
	"math"
	"sort"
)

// ObjectTree 管理场景中的物体层次结构
type ObjectTree struct {
	Root        *ObjectNode
	Objects     []Object
	ObjectNodes []*ObjectNode
}

// Add 添加新物体到场景
func (t *ObjectTree) Add(shape Shape, material *Material) *Object {
	t.Objects = append(t.Objects, Object{
		Shape:    shape,
		Material: material,
	})
	return &t.Objects[len(t.Objects)-1]
}

// Build 构建对象树
func (t *ObjectTree) Build() *ObjectTree {
	t.ObjectNodes = nil
	for i := range t.Objects {
		node := NewObjectNode(&t.Objects[i], nil, nil)
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
	delta := r3.Vec{}

	for i := l + 1; i <= r; i++ {
		box := t.ObjectNodes[i].BoundBox
		pmin = r3.Vec{
			X: math.Min(pmin.X, box.Pmin.X),
			Y: math.Min(pmin.Y, box.Pmin.Y),
			Z: math.Min(pmin.Z, box.Pmin.Z),
		}
		pmax = r3.Vec{
			X: math.Max(pmax.X, box.Pmax.X),
			Y: math.Max(pmax.Y, box.Pmax.Y),
			Z: math.Max(pmax.Z, box.Pmax.Z),
		}
		boxDelta := r3.Sub(box.Pmax, box.Pmin)
		delta = r3.Vec{
			X: math.Max(delta.X, boxDelta.X),
			Y: math.Max(delta.Y, boxDelta.Y),
			Z: math.Max(delta.Z, boxDelta.Z),
		}
	}

	(*node).BoundBox = &Cuboid{Pmin: pmin, Pmax: pmax}
	size := r3.Sub(pmax, pmin)
	dimRatios := r3.Vec{
		X: delta.X / size.X,
		Y: delta.Y / size.Y,
		Z: delta.Z / size.Z,
	}

	dim := 0
	if dimRatios.Y > dimRatios.X {
		dim = 1
	}
	if dimRatios.Z > dimRatios.X && dimRatios.Z > dimRatios.Y {
		dim = 2
	}

	sort.Slice(t.ObjectNodes[l:r+1], func(i, j int) bool {
		a := t.ObjectNodes[l+i].BoundBox.Pmin
		b := t.ObjectNodes[l+j].BoundBox.Pmin
		if a.Get(dim) != b.Get(dim) {
			return a.Get(dim) < b.Get(dim)
		}
		return t.ObjectNodes[l+i].BoundBox.Pmax.Get(dim) < t.ObjectNodes[l+j].BoundBox.Pmax.Get(dim)
	})

	mid := (l + r) / 2
	t.build(l, mid, &(*node).Children[0])
	t.build(mid+1, r, &(*node).Children[1])
}

// GetIntersection 查找光线与物体的交点
func (t *ObjectTree) GetIntersection(raySt, rayDir r3.Vec, node *ObjectNode) (float64, *Object) {
	if node == nil {
		return math.MaxFloat64, nil
	}

	if node.Obj != nil {
		return node.Obj.Shape.Intersect(raySt, rayDir), node.Obj
	}

	if node.BoundBox.Intersect(raySt, rayDir) >= math.MaxFloat64 {
		return math.MaxFloat64, nil
	}

	dis1, obj1 := t.GetIntersection(raySt, rayDir, node.Children[0])
	dis2, obj2 := t.GetIntersection(raySt, rayDir, node.Children[1])

	if dis1 < dis2 {
		return dis1, obj1
	}
	return dis2, obj2
}
