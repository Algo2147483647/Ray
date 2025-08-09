package shape_library

type Triangle struct {
	P1, P2, P3 r3.Vec
	Engraving  func(r3.Vec) bool
}

func (t *Triangle) GetName() string {
	return "Triangle"
}

func (t *Triangle) Intersect(raySt, rayDir r3.Vec) float64 {
	// 从线程池获取资源
	edges := edgePool.Get().([2]r3.Vec)
	tmp := tmpPool.Get().(r3.Vec)
	defer func() {
		edgePool.Put(edges)
		tmpPool.Put(tmp)
	}()

	// 计算三角形边向量
	edges[0] = t.P2.Sub(t.P1)
	edges[1] = t.P3.Sub(t.P1)

	// 计算法向量和行列式
	p := rayDir.Cross(edges[1])
	a := edges[0].Dot(p)

	// 处理背面剔除
	if a > 0 {
		tmp = raySt.Sub(t.P1)
	} else {
		tmp = t.P1.Sub(raySt)
		a = -a
	}

	// 检查平行
	if a < EPS {
		return math.MaxFloat64
	}

	// 计算重心坐标 u
	u := tmp.Dot(p) / a
	if u < 0 || u > 1 {
		return math.MaxFloat64
	}

	// 计算重心坐标 v
	q := tmp.Cross(edges[0])
	v := q.Dot(rayDir) / a
	if v < 0 || u+v > 1 {
		return math.MaxFloat64
	}

	// 计算交点参数 t
	return q.Dot(edges[1]) / a
}

func (t *Triangle) GetNormalVector(intersect r3.Vec) r3.Vec {
	edge1 := t.P2.Sub(t.P1)
	edge2 := t.P3.Sub(t.P1)
	return edge1.Cross(edge2).Normalize()
}

func (t *Triangle) BuildBoundingBox() (r3.Vec, r3.Vec) {
	pmin := r3.Vec{
		X: min(t.P1.X, min(t.P2.X, t.P3.X)),
		Y: min(t.P1.Y, min(t.P2.Y, t.P3.Y)),
		Z: min(t.P1.Z, min(t.P2.Z, t.P3.Z)),
	}
	pmax := r3.Vec{
		X: max(t.P1.X, max(t.P2.X, t.P3.X)),
		Y: max(t.P1.Y, max(t.P2.Y, t.P3.Y)),
		Z: max(t.P1.Z, max(t.P2.Z, t.P3.Z)),
	}
	return pmax, pmin
}
