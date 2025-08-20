package ray_tracing

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
	"src-golang/model"
	"sync"
	"sync/atomic"
	"time"
)

// TraceScene 追踪整个场景
func (h *Handler) TraceScene(scene *model.Scene, img [3]*mat.Dense, samples int) {
	rows, cols := img[0].Dims()
	totalPixels := rows * cols

	var wg sync.WaitGroup
	taskChan := make(chan [2]int, rows*cols)

	// 原子进度计数器
	var progress int64

	// 启动进度显示goroutine
	done := make(chan bool)
	go func() {
		startTime := time.Now()
		for {
			select {
			case <-done:
				elapsed := time.Since(startTime).Round(time.Second)
				fmt.Printf("\rRendering complete! Pixels: %d/%d (100%%) Time: %v\n",
					totalPixels, totalPixels, elapsed)
				return
			default:
				current := atomic.LoadInt64(&progress)
				percent := float64(current) / float64(totalPixels) * 100
				elapsed := time.Since(startTime).Round(time.Second)
				fmt.Printf("\rRendering: %d/%d pixels (%.2f%%) Time: %v",
					current, totalPixels, percent, elapsed)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// 创建任务队列
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			taskChan <- [2]int{r, c}
		}
	}
	close(taskChan)

	// 启动工作线程
	for i := 0; i < h.ThreadNum; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			for pixel := range taskChan {
				color := h.TracePixel(scene.Cameras[0], scene.ObjectTree, pixel[0], pixel[1], samples)
				for ch := 0; ch < 3; ch++ {
					img[ch].Set(pixel[0], pixel[1], color.AtVec(ch))
				}

				atomic.AddInt64(&progress, 1) // 原子更新进度
			}
		}(int64(i))
	}

	wg.Wait()
	close(done)                        // 通知进度条关闭
	time.Sleep(100 * time.Millisecond) // 确保最后进度信息被覆盖
}
