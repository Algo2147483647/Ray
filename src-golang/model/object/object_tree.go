package object

import (
	"bufio"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
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
func (t *ObjectTree) LoadSceneFromScript(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	materials := make(map[string]*Material)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		switch parts[0] {
		case "Material":
			if len(parts) < 4 {
				continue
			}

			// 解析材质定义: Material name {r,g,b} [属性...]
			name := parts[1]
			colorStr := strings.Trim(parts[2], "{}")
			colorParts := strings.Split(colorStr, ",")
			if len(colorParts) != 3 {
				continue
			}

			r, _ := strconv.ParseFloat(colorParts[0], 64)
			g, _ := strconv.ParseFloat(colorParts[1], 64)
			b, _ := strconv.ParseFloat(colorParts[2], 64)
			material := NewMaterial(mat.NewVecDense(3, []float64{r, g, b}))

			// 解析材质属性
			for _, prop := range parts[3:] {
				kv := strings.Split(prop, "=")
				if len(kv) != 2 {
					continue
				}

				val, _ := strconv.ParseFloat(kv[1], 64)
				switch kv[0] {
				case "Diffuse":
					material.DiffuseLoss = val
				case "Reflect":
					material.Reflectivity = val
				case "Refract":
					material.Refractivity = val
				case "Refractivity":
					material.RefractiveIndex = val
				case "Radiate":
					material.Radiation = val > 0
				}
			}
			materials[name] = material

		case "Object":
			if len(parts) < 4 {
				continue
			}

			// 解析物体定义: Object shape material [参数...]
			shapeType := parts[1]
			materialName := parts[2]
			material, ok := materials[materialName]
			if !ok {
				continue
			}

			switch shapeType {
			case "Cuboid":
				// 格式: Cuboid {x1,y1,z1} {x2,y2,z2}
				if len(parts) < 5 {
					continue
				}
				p1Str := strings.Trim(parts[3], "{}")
				p2Str := strings.Trim(parts[4], "{}")
				p1 := parseVec(p1Str)
				p2 := parseVec(p2Str)
				t.Add(NewCuboid(p1, p2), material)

			case "Sphere":
				// 格式: Sphere {x,y,z} Radius=value
				if len(parts) < 4 {
					continue
				}
				centerStr := strings.Trim(parts[3], "{}")
				center := parseVec(centerStr)
				radius := 1.0

				for _, prop := range parts[4:] {
					if strings.HasPrefix(prop, "Radius=") {
						radius, _ = strconv.ParseFloat(prop[7:], 64)
					}
				}
				t.Add(NewSphere(center, radius), material)
			}
		}
	}
	return nil
}

// parseVec 解析向量字符串
func parseVec(s string) *mat.VecDense {
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return &mat.VecDense{}
	}
	x, _ := strconv.ParseFloat(parts[0], 64)
	y, _ := strconv.ParseFloat(parts[1], 64)
	z, _ := strconv.ParseFloat(parts[2], 64)
	return mat.NewVecDense(3, []float64{x, y, z})
}
