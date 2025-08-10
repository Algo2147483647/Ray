package object

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"sort"
	"src-golang/controller"
)

// ObjectTree 管理场景中的物体层次结构
type ObjectTree struct {
	Root        *ObjectNode
	Objects     []Object
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
	delta := mat.NewVecDense(3, []float64{0, 0, 0})

	for i := l + 1; i <= r; i++ {
		box := t.ObjectNodes[i].BoundBox
		pmin = vecMin(pmin, box.Pmin)
		pmax = vecMax(pmax, box.Pmax)
		boxDelta := vecSub(box.Pmax, box.Pmin)

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

// LoadSceneFromScript 从脚本加载场景
func (t *ObjectTree) LoadSceneFromScript(script *controller.Script) error {
	// 创建材质映射表
	materials := make(map[string]*Material)

	// 解析材质
	for _, matDef := range script.Materials {
		// 将颜色从 [0-255] 转换为 [0.0-1.0]
		r := float64(matDef.Color[0]) / 255.0
		g := float64(matDef.Color[1]) / 255.0
		b := float64(matDef.Color[2]) / 255.0

		material := NewMaterial(mat.NewVecDense(3, []float64{r, g, b}))

		// 设置材质属性
		if matDef.Diffuse != 0 {
			material.DiffuseLoss = matDef.Diffuse
		}
		if matDef.Reflect != 0 {
			material.Reflectivity = matDef.Reflect
		}
		if matDef.Refractivity != 0 {
			material.Refractivity = matDef.Refractivity
		}
		if matDef.Radiate != 0 {
			material.Radiation = matDef.Radiate != 0
		}

		materials[matDef.Key] = material
	}

	// 解析物体
	for _, objDef := range script.Objects {
		material, exists := materials[objDef.Material]
		if !exists {
			continue // 跳过未定义材质的物体
		}

		position := mat.NewVecDense(3, []float64{
			float64(objDef.Position[0]),
			float64(objDef.Position[1]),
			float64(objDef.Position[2]),
		})

		switch objDef.Shape {
		case "Cuboid":
			if len(objDef.Size) < 3 {
				continue
			}

			// 计算长方体对角点
			size := mat.NewVecDense(3, []float64{
				float64(objDef.Size[0]),
				float64(objDef.Size[1]),
				float64(objDef.Size[2]),
			})

			p2 := mat.NewVecDense(3, nil)
			p2.AddVec(position, size)

			t.Add(NewCuboid(position, p2), material)

		case "Sphere":
			radius := float64(objDef.Radius)
			if radius == 0 {
				radius = 1.0 // 默认半径
			}

			t.Add(NewSphere(position, radius), material)

		case "Plane":
			// 平面需要法线方向，这里使用位置向量作为法线
			//normal := position
			//if normal.Norm() == 0 {
			//	normal = mat.NewVecDense(3, []float64{0, 1, 0}) // 默认法线向上
			//} else {
			//	normal.ScaleVec(1/normal.Norm(), normal) // 归一化
			//}

			// XZH t.Add(NewPlane(position, normal), material)

		case "Cylinder":
			//height := float64(objDef.Size[0])
			//radius := float64(objDef.Size[1])
			//if height == 0 {
			//	height = 1.0
			//}
			//if radius == 0 {
			//	radius = 0.5
			//}
			//
			//// 圆柱体方向，默认为Y轴
			//axis := mat.NewVecDense(3, []float64{0, 1, 0})
			//if len(objDef.Size) > 2 {
			//	axis = mat.NewVecDense(3, []float64{
			//		float64(objDef.Size[0]),
			//		float64(objDef.Size[1]),
			//		float64(objDef.Size[2]),
			//	})
			//}
			//
			//t.Add(NewCylinder(position, axis, height, radius), material)
		}
	}

	return nil
}
