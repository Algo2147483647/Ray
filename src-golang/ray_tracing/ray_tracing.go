package ray_tracing

import (
	"bufio"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"math"
	"math/rand"
	"os"
	"src-golang/model"
	"src-golang/model/object"
	"strconv"
	"strings"
	"sync"
)

// 全局配置
var (
	MaxRayLevel = 6        // 最大光线递归深度
	ThreadNum   = 20       // 并发线程数
	globalMutex sync.Mutex // 全局互斥锁
)

// TraceRay 递归追踪光线
func TraceRay(objTree *object.ObjectTree, ray *model.Ray, level int) *mat.VecDense {
	if level > MaxRayLevel {
		return mat.NewVecDense(3, nil) // 返回黑色 (0,0,0)
	}

	// 查找最近交点
	dis, obj := objTree.GetIntersection(ray.Origin, ray.Direction, objTree.Root)
	if dis >= math.MaxFloat64 {
		return mat.NewVecDense(3, nil) // 无交点，返回黑色
	}

	// 移动光线到交点: origin = origin + dis * direction
	newOrigin := new(mat.VecDense)
	newOrigin.ScaleVec(dis, ray.Direction)
	newOrigin.AddVec(ray.Origin, newOrigin)
	ray.Origin = newOrigin

	// 计算法向量
	normal := obj.Shape.NormalVector(ray.Origin)

	// 确保法线朝向光源: 如果法线方向与光线方向夹角大于90度则翻转
	if mat.Dot(normal, ray.Direction) > 0 {
		normal.ScaleVec(-1, normal)
	}

	// 处理材质交互
	terminate := obj.Material.DielectricSurfacePropagation(ray, normal, rand.New(rand.NewSource(0)))
	if terminate {
		return ray.Color
	}

	// 递归追踪
	return TraceRay(objTree, ray, level+1)
}

// TraceRayThread 线程安全的光线追踪
func TraceRayThread(camera *model.Camera, objTree *object.ObjectTree, img [3]*mat.Dense, rays []*model.Ray, wg *sync.WaitGroup) {
	defer wg.Done()
	rows, cols := img[0].Dims()
	for _, ray := range rays {
		color := TraceRay(objTree, ray, 0)
		x, y := camera.GetRayCoordinates(ray, rows, cols)

		// 线程安全地更新图像
		globalMutex.Lock()
		for i, _ := range img {
			img[i].Set(x, y, color.AtVec(i))
		}
		globalMutex.Unlock()
	}
}

// TraceScene 追踪整个场景
func TraceScene(camera *model.Camera, objTree *object.ObjectTree, img [3]*mat.Dense, samples int) {
	// 计算每线程处理的光线数
	rows, cols := img[0].Dims()
	totalPixels := rows * cols
	raysPerThread := totalPixels / ThreadNum

	var wg sync.WaitGroup
	wg.Add(ThreadNum)

	for sample := 0; sample < samples; sample++ {
		// 生成所有光线
		rays := camera.GetRays(rows, cols, rand.New(rand.NewSource(0)))

		for i := 0; i < ThreadNum; i++ {
			start := i * raysPerThread
			end := start + raysPerThread
			if i == ThreadNum-1 {
				end = len(rays) // 最后一个线程处理剩余光线
			}

			go TraceRayThread(camera, objTree, img, rays[start:end], &wg)
		}

		wg.Wait()
	}

	for i := range img {
		img[i].Scale(1.0/float64(samples), img[i])
	}
}

// LoadSceneFromScript 从脚本加载场景
func LoadSceneFromScript(filepath string, objTree *object.ObjectTree) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	materials := make(map[string]*object.Material)
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
			material := object.NewMaterial(mat.NewVecDense(3, []float64{r, g, b}))

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
				objTree.Add(object.NewCuboid(p1, p2), material)

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
				objTree.Add(object.NewSphere(center, radius), material)
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
