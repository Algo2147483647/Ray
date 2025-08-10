package ray_tracing

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"src-golang/model"
	"src-golang/model/object"
)

// TraceRay 迭代式光线追踪 (替代递归)
func TraceRay(objTree *object.ObjectTree, ray *model.Ray) *mat.VecDense {
	var (
		color  = mat.NewVecDense(3, []float64{1, 1, 1}) // 初始颜色为白色
		weight = mat.NewVecDense(3, []float64{1, 1, 1}) // 累积权重
		temp   = mat.NewVecDense(3, nil)                // 复用向量避免内存分配
		normal = mat.NewVecDense(3, nil)
	)

	for level := 0; level <= MaxRayLevel; level++ {
		// 查找最近交点
		dis, obj := objTree.GetIntersection(ray.Origin, ray.Direction, objTree.Root)
		if dis >= math.MaxFloat64 {
			color.ScaleVec(0, color) // 无交点返回黑色
			break
		}

		// 计算新交点, 法向量：新交点 origin = origin + dis * direction, 确保法线朝向光源
		temp.ScaleVec(dis, ray.Direction)
		ray.Origin.AddVec(ray.Origin, temp)
		normal = obj.Shape.NormalVector(ray.Origin)
		if dot := mat.Dot(normal, ray.Direction); dot > 0 {
			normal.ScaleVec(-1, normal)
		}

		// 处理材质交互
		terminate := obj.Material.DielectricSurfacePropagation(ray, normal, rand.New(rand.NewSource(0)))
		if terminate {
			color.MulElemVec(color, ray.Color)
			break
		}

		// 累积颜色贡献
		color.MulElemVec(color, ray.Color)

		// 更新权重
		weight.MulElemVec(weight, ray.Color)
		if norm := mat.Norm(weight, 2); norm < 1e-3 {
			break // 权重过小提前终止
		}
	}
	return color
}
