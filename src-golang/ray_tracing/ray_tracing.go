package ray_tracing

import (
	"bufio"
	"fmt"
	"gonum.org/v1/gonum/spatial/r3"
	"math"
	"math/rand"
	"os"
	"src-golang/model"
	"src-golang/model/object"
	"src-golang/model/object/shape_library"
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
func TraceRay(objTree *object.ObjectTree, ray *model.Ray, level int) r3.Vec {
	if level > MaxRayLevel {
		return r3.Vec{} // 黑色
	}

	// 查找最近交点
	dis, obj := objTree.GetIntersection(ray.Origin, ray.Direction, objTree.Root)
	if dis >= math.MaxFloat64 {
		return r3.Vec{} // 无交点
	}

	// 移动光线到交点
	ray.Origin = r3.Add(ray.Origin, r3.Scale(dis, ray.Direction))

	// 计算法向量
	normal := obj.Shape.NormalVector(ray.Origin)
	if r3.Dot(normal, ray.Direction) > 0 {
		normal = r3.Scale(-1, normal) // 确保法线朝向光源
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
func TraceRayThread(camera *model.Camera, objTree *object.ObjectTree, img *Image, rays []*model.Ray, wg *sync.WaitGroup) {
	defer wg.Done()

	for _, ray := range rays {
		color := TraceRay(objTree, ray, 0)
		x, y := camera.GetRayCoordinates(ray, img.Width, img.Height)

		// 线程安全地更新图像
		globalMutex.Lock()
		img.AddColor(x, y, color)
		globalMutex.Unlock()
	}
}

// TraceScene 追踪整个场景
func TraceScene(camera *model.Camera, objTree *object.ObjectTree, img *Image, samples int) {
	raysPerThread := (img.Width * img.Height) / ThreadNum

	var wg sync.WaitGroup
	wg.Add(ThreadNum)

	for sample := 0; sample < samples; sample++ {
		rays := camera.GetRays(img.Width, img.Height, rand.New(rand.NewSource(0)))

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

	// 平均采样结果
	img.Scale(1.0 / float64(samples))
}

// Image 表示渲染图像
type Image struct {
	R, G, B       [][]float64 // 红绿蓝通道
	Width, Height int
}

// NewImage 创建新图像
func NewImage(width, height int) *Image {
	img := &Image{
		Width:  width,
		Height: height,
		R:      make([][]float64, height),
		G:      make([][]float64, height),
		B:      make([][]float64, height),
	}

	for y := 0; y < height; y++ {
		img.R[y] = make([]float64, width)
		img.G[y] = make([]float64, width)
		img.B[y] = make([]float64, width)
	}
	return img
}

// AddColor 添加颜色到像素
func (img *Image) AddColor(x, y int, color r3.Vec) {
	if x >= 0 && x < img.Width && y >= 0 && y < img.Height {
		img.R[y][x] += color.X
		img.G[y][x] += color.Y
		img.B[y][x] += color.Z
	}
}

// Scale 缩放图像
func (img *Image) Scale(factor float64) {
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			img.R[y][x] *= factor
			img.G[y][x] *= factor
			img.B[y][x] *= factor
		}
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
			material := object.NewMaterial(r3.Vec{X: r, Y: g, Z: b})

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
				objTree.Add(shape_library.NewCuboid(p1, p2), material)

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
				objTree.Add(NewSphere(center, radius), material)
			}
		}
	}
	return nil
}

// parseVec 解析向量字符串
func parseVec(s string) r3.Vec {
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return r3.Vec{}
	}
	x, _ := strconv.ParseFloat(parts[0], 64)
	y, _ := strconv.ParseFloat(parts[1], 64)
	z, _ := strconv.ParseFloat(parts[2], 64)
	return r3.Vec{X: x, Y: y, Z: z}
}

// Cuboid 表示长方体
type Cuboid struct {
	Min, Max r3.Vec
}

func (c *Cuboid) Name() string { return "Cuboid" }

func (c *Cuboid) Intersect(rayStart, rayDir r3.Vec) float64 {
	return c.Intersect(rayStart, rayDir) // 使用之前定义的包围盒相交算法
}

func (c *Cuboid) NormalVector(point r3.Vec) r3.Vec {
	// 计算最接近的面
	distances := []float64{
		math.Abs(point.X - c.Min.X),
		math.Abs(point.X - c.Max.X),
		math.Abs(point.Y - c.Min.Y),
		math.Abs(point.Y - c.Max.Y),
		math.Abs(point.Z - c.Min.Z),
		math.Abs(point.Z - c.Max.Z),
	}

	minDist := math.MaxFloat64
	normal := r3.Vec{}
	for i, d := range distances {
		if d < minDist {
			minDist = d
			switch i {
			case 0:
				normal = r3.Vec{-1, 0, 0}
			case 1:
				normal = r3.Vec{1, 0, 0}
			case 2:
				normal = r3.Vec{0, -1, 0}
			case 3:
				normal = r3.Vec{0, 1, 0}
			case 4:
				normal = r3.Vec{0, 0, -1}
			case 5:
				normal = r3.Vec{0, 0, 1}
			}
		}
	}
	return normal
}

func (c *Cuboid) BoundingBox() (pmax, pmin r3.Vec) {
	return c.Max, c.Min
}

// Sphere 表示球体
type Sphere struct {
	Center r3.Vec
	Radius float64
}

func NewSphere(center r3.Vec, radius float64) *Sphere {
	return &Sphere{Center: center, Radius: radius}
}

func (s *Sphere) Name() string { return "Sphere" }

func (s *Sphere) Intersect(rayStart, rayDir r3.Vec) float64 {
	oc := r3.Sub(rayStart, s.Center)
	a := r3.Dot(rayDir, rayDir)
	b := 2 * r3.Dot(oc, rayDir)
	c := r3.Dot(oc, oc) - s.Radius*s.Radius
	discriminant := b*b - 4*a*c

	if discriminant < 0 {
		return math.MaxFloat64
	}

	sqrtD := math.Sqrt(discriminant)
	t1 := (-b - sqrtD) / (2 * a)
	t2 := (-b + sqrtD) / (2 * a)

	if t1 > 0 {
		return t1
	}
	if t2 > 0 {
		return t2
	}
	return math.MaxFloat64
}

func (s *Sphere) NormalVector(point r3.Vec) r3.Vec {
	return r3.Unit(r3.Sub(point, s.Center))
}

func (s *Sphere) BoundingBox() (pmax, pmin r3.Vec) {
	r := r3.Vec{s.Radius, s.Radius, s.Radius}
	return r3.Add(s.Center, r), r3.Sub(s.Center, r)
}
