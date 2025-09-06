package ray_tracing

import (
	"fmt"
	"src-golang/model"
	"src-golang/model/camera"
	"sync"
	"sync/atomic"
	"time"
)

// TraceScene 追踪整个场景
func (h *Handler) TraceScene(scene *model.Scene, film *camera.Film, samples int64) {
	var (
		progress    int64 // 原子进度计数器
		totalPixels = len(film.Data[0].Data)
		wg          sync.WaitGroup
		taskChan    = make(chan []int, totalPixels)
		done        = make(chan bool) // 启动进度显示goroutine
	)

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
	for i := 0; i < len(film.Data[0].Data); i++ {
		taskChan <- film.Data[0].GetCoordinates(i)
	}
	close(taskChan)

	// 启动工作线程
	for i := 0; i < h.ThreadNum; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			for pixel := range taskChan {
				color := h.TracePixel(scene.Cameras[0], scene.ObjectTree, samples, pixel...)
				for ch := 0; ch < 3; ch++ {
					film.Data[ch].Set(color.AtVec(ch), pixel...)
				}

				atomic.AddInt64(&progress, 1) // 原子更新进度
			}
		}(int64(i))
	}

	wg.Wait()
	close(done)                        // 通知进度条关闭
	time.Sleep(100 * time.Millisecond) // 确保最后进度信息被覆盖

	film.Samples = samples
}
