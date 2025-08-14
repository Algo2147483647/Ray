package object

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
	"math"
	"sort"
	"src-golang/math_lib"
	"src-golang/model/object/shape"
	"strings"
)

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

	fmt.Println(t.TreeString())
	return t
}

// build 递归构建树结构
func (t *ObjectTree) build(l, r int, node **ObjectNode) {
	if l > r {
		return
	}

	if l == r {
		*node = t.ObjectNodes[l]
		return
	}

	// 1. 计算所有物体的合并包围盒
	pmin := mat.NewVecDense(t.ObjectNodes[l].BoundBox.Pmin.Len(), nil)
	pmin.CopyVec(t.ObjectNodes[l].BoundBox.Pmin)
	pmax := mat.NewVecDense(t.ObjectNodes[l].BoundBox.Pmax.Len(), nil)
	pmax.CopyVec(t.ObjectNodes[l].BoundBox.Pmax)

	for i := l + 1; i <= r; i++ {
		box := t.ObjectNodes[i].BoundBox
		pmin = vecMin(pmin, box.Pmin)
		pmax = vecMax(pmax, box.Pmax)
	}

	// 2. 选择划分维度
	size := mat.NewVecDense(t.ObjectNodes[l].BoundBox.Pmax.Len(), nil)
	size.SubVec(pmax, pmin)
	dim := selectSplitDimension(t.ObjectNodes[l:r+1], size)

	// 3. 按选定维度排序
	sort.Slice(t.ObjectNodes[l:r+1], func(i, j int) bool {
		a := t.ObjectNodes[l+i]
		b := t.ObjectNodes[l+j]

		// 主维度比较：中心点位置
		aCenter := (a.BoundBox.Pmin.AtVec(dim) + a.BoundBox.Pmax.AtVec(dim)) / 2
		bCenter := (b.BoundBox.Pmin.AtVec(dim) + b.BoundBox.Pmax.AtVec(dim)) / 2
		if aCenter != bCenter {
			return aCenter < bCenter
		}

		// 次级维度比较：包围盒起点
		if a.BoundBox.Pmin.AtVec(dim) != b.BoundBox.Pmin.AtVec(dim) {
			return a.BoundBox.Pmin.AtVec(dim) < b.BoundBox.Pmin.AtVec(dim)
		}

		// 终极比较：包围盒终点
		return a.BoundBox.Pmax.AtVec(dim) < b.BoundBox.Pmax.AtVec(dim)
	})

	// 4. 创建内部节点并递归构建子树
	*node = NewObjectNode(nil, nil, nil)
	(*node).BoundBox = &shape.Cuboid{Pmin: pmin, Pmax: pmax}

	mid := (l + r) / 2
	t.build(l, mid, &(*node).Children[0])
	t.build(mid+1, r, &(*node).Children[1])
	t.ObjectNodes = append(t.ObjectNodes, *node)
}

// 选择划分维度（基于中心点方差）
func selectSplitDimension(nodes []*ObjectNode, size *mat.VecDense) int {
	// 计算每个维度的中心点方差
	var maxVariance = -math.MaxFloat64
	var bestDim = 0

	for dim := 0; dim < 3; dim++ {
		if size.AtVec(dim) < 1e-8 { // 跳过坍缩维度
			continue
		}

		// 计算中心点平均值
		var sum float64
		for _, node := range nodes {
			center := (node.BoundBox.Pmin.AtVec(dim) + node.BoundBox.Pmax.AtVec(dim)) / 2
			sum += center
		}
		mean := sum / float64(len(nodes))

		// 计算方差
		var variance float64
		for _, node := range nodes {
			center := (node.BoundBox.Pmin.AtVec(dim) + node.BoundBox.Pmax.AtVec(dim)) / 2
			diff := center - mean
			variance += diff * diff
		}
		variance /= float64(len(nodes))

		// 归一化方差（考虑包围盒尺寸）
		normalizedVariance := variance / size.AtVec(dim)

		if normalizedVariance > maxVariance {
			maxVariance = normalizedVariance
			bestDim = dim
		}
	}

	// 如果所有维度都坍缩，默认选择X轴
	if maxVariance == -math.MaxFloat64 {
		return 0
	}
	return bestDim
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

	if dis1 < math_lib.EPS {
		dis1 = math.MaxFloat64
	}
	if dis2 < math_lib.EPS {
		dis2 = math.MaxFloat64
	}

	if dis1 < dis2 {
		return dis1, obj1
	}
	return dis2, obj2
}

func (ot *ObjectTree) TreeString() string {
	if ot.Root == nil {
		return "ObjectTree is empty"
	}

	var sb strings.Builder
	sb.WriteString("ObjectTree Structure:\n")
	sb.WriteString(ot.Root.TreeNodeString(0))

	// 添加未在树中的孤立物体
	if len(ot.Objects) > 0 {
		sb.WriteString("\n\nOrphan Objects (not in tree):")
		for _, obj := range ot.Objects {
			sb.WriteString(fmt.Sprintf("\n  - %s", obj.Shape.Name()))
		}
	}

	return sb.String()
}
