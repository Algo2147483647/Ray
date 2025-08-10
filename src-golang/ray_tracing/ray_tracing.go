package ray_tracing

import (
	"gonum.org/v1/gonum/mat"
	"src-golang/model"
	"src-golang/model/object"
	"sync"
)

// 全局配置
var (
	MaxRayLevel = 6   // 最大光线递归深度
	ThreadNum   = 200 // 并发线程数
)

// TracePixel 追踪单个像素
func TracePixel(camera *model.Camera, objTree *object.ObjectTree, row, col, samples int) *mat.VecDense {
	color := mat.NewVecDense(3, nil)
	for s := 0; s < samples; s++ {
		ray := RayPool.Get().(*model.Ray)
		defer RayPool.Put(ray)

		camera.GenerateRay(ray, row, col)
		sampleColor := TraceRay(objTree, ray)
		color.AddVec(color, sampleColor)
	}
	color.ScaleVec(1.0/float64(samples), color)
	return color
}

// TraceScene 追踪整个场景
func TraceScene(camera *model.Camera, objTree *object.ObjectTree, img [3]*mat.Dense, samples int) {
	rows, cols := img[0].Dims()
	var wg sync.WaitGroup
	taskChan := make(chan [2]int, rows*cols)

	// 创建任务队列
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			taskChan <- [2]int{r, c}
		}
	}
	close(taskChan)

	// 启动工作线程
	for i := 0; i < ThreadNum; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()

			for pixel := range taskChan {
				r, c := pixel[0], pixel[1]
				color := TracePixel(camera, objTree, r, c, samples)

				for ch := 0; ch < 3; ch++ {
					img[ch].Set(r, c, color.AtVec(ch)) // 直接写入全局图像 (无锁写入，每个像素独立)
				}
			}
		}(int64(i))
	}
	wg.Wait()
}
