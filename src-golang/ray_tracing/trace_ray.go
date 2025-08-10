package ray_tracing

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/model/object"
	"src-golang/model/ray"
)

// TraceRay 迭代式光线追踪 (替代递归)
func TraceRay(objTree *object.ObjectTree, ray *ray.Ray, level int) *mat.VecDense {
	var (
		temp      = mat.NewVecDense(3, nil) // 复用向量避免内存分配
		normal    = mat.NewVecDense(3, nil)
		hitObject = "MISS"
		distance  = float64(0)
	)

	defer func() {
		if ray.DebugSwitch {
			ray.DebugTraces = append(ray.DebugTraces, map[string]interface{}{
				"origin":     ray.Origin,
				"direction":  ray.Direction,
				"color":      ray.Color,
				"level":      level,
				"hit_object": hitObject,
				"distance":   distance,
			})

			if level == 0 && hitObject != "MISS" {
				println(ray.DebugString())
			}
		}
	}()

	if level > MaxRayLevel {
		ray.Color.ScaleVec(0, ray.Color)
		return ray.Color
	}

	// 查找最近交点
	distance, obj := objTree.GetIntersection(ray.Origin, ray.Direction, objTree.Root)
	if distance >= math.MaxFloat64 {
		ray.Color.ScaleVec(0, ray.Color) // 无交点返回黑色
		return ray.Color
	} else {
		hitObject = obj.Shape.Name()
	}

	// 计算新交点, 法向量：新交点 origin = origin + dis * direction, 确保法线朝向光源
	temp.ScaleVec(distance, ray.Direction)
	ray.Origin.AddVec(ray.Origin, temp)
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
