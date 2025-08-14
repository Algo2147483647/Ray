package ray_tracing

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
	"src-golang/model/object"
	"src-golang/model/object/optics"
)

// TraceRay 迭代式光线追踪 (替代递归)
func TraceRay(objTree *object.ObjectTree, ray *optics.Ray, level int) *mat.VecDense {
	var (
		normal    = mat.NewVecDense(3, nil)
		origin    = mat.NewVecDense(3, nil)
		hitObject = ""
		distance  = float64(0)
	)
	origin.CloneFromVec(ray.Origin)

	defer func() {
		if ray.DebugSwitch && hitObject != "" {
			optics.DebugRayTraces = append(optics.DebugRayTraces, map[string]interface{}{
				"start":      origin.RawVector().Data,
				"end":        ray.Origin.RawVector().Data,
				"direction":  ray.Direction.RawVector().Data,
				"color":      ray.Color.RawVector().Data,
				"level":      level,
				"hit_object": hitObject,
				"distance":   distance,
			})
		}
	}()

	if level > MaxRayLevel {
		ray.Color.ScaleVec(0, ray.Color)
		return ray.Color
	}

	// 查找最近交点
	distance, obj := objTree.GetIntersection(ray.Origin, ray.Direction, objTree.Root)
	if distance >= math.MaxFloat64 {
		return math_lib.ScaleVec(ray.Color, 0, ray.Color) // 无交点返回黑色
	} else {
		hitObject = obj.Shape.Name()
	}

	// 计算新交点, 法向量：新交点 origin = origin + dis * direction, 确保法线朝向光源
	ray.Origin.AddVec(ray.Origin, math_lib.ScaleVec2(distance, ray.Direction))
	normal = obj.Shape.GetNormalVector(ray.Origin)
	if dot := mat.Dot(normal, ray.Direction); dot > 0 {
		normal.ScaleVec(-1, normal)
	}

	// 处理材质交互
	terminate := obj.Material.DielectricSurfacePropagation(ray, normal)
	if terminate {
		return ray.Color
	}

	return TraceRay(objTree, ray, level+1)
}
