package ray_tracing

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
	"src-golang/model/object"
	"src-golang/model/optics"
	"src-golang/utils"
)

func (h *Handler) TraceRay(objTree *object.ObjectTree, ray *optics.Ray, level int64) *mat.VecDense {
	var (
		normal        = mat.NewVecDense(3, nil)
		DebugRayTrace = map[string]interface{}{}
	)

	if utils.IsDebug {
		DebugRayTrace = map[string]interface{}{
			"start":      append([]float64(nil), ray.Origin.RawVector().Data...),
			"direction":  append([]float64(nil), ray.Direction.RawVector().Data...),
			"color":      append([]float64(nil), ray.Color.RawVector().Data...),
			"level":      level,
			"hit_object": "",
		}

		normal.AddVec(ray.Origin, math_lib.ScaleVec2(1, ray.Direction))
		DebugRayTrace["end"] = append([]float64(nil), normal.RawVector().Data...)
	}

	defer func() {
		if utils.IsDebug && ray.DebugSwitch {
			optics.DebugRayTraces = append(optics.DebugRayTraces, DebugRayTrace)
		}
	}()

	if level > h.MaxRayLevel {
		ray.Color.ScaleVec(0, ray.Color)
		return ray.Color
	}

	distance, obj := objTree.GetIntersection(ray.Origin, ray.Direction, objTree.Root) // 查找最近交点
	if distance >= math.MaxFloat64 {
		return math_lib.ScaleVec(ray.Color, 0, ray.Color) // 无交点返回黑色
	}

	ray.Origin.AddVec(ray.Origin, math_lib.ScaleVec2(distance, ray.Direction)) // 计算新交点, 法向量：新交点 origin = origin + dis * direction, 确保法线朝向光源
	normal = obj.Shape.GetNormalVector(ray.Origin, normal)
	if dot := mat.Dot(normal, ray.Direction); dot > 0 {
		normal.ScaleVec(-1, normal)
	}

	if utils.IsDebug {
		DebugRayTrace["hit_object"] = obj.Shape.Name()
		DebugRayTrace["end"] = append([]float64(nil), ray.Origin.RawVector().Data...)
		DebugRayTrace["distance"] = distance
	}

	// 处理材质交互
	terminate := obj.Material.DielectricSurfacePropagation(ray, normal)
	if terminate {
		return ray.Color
	}

	return h.TraceRay(objTree, ray, level+1)
}
