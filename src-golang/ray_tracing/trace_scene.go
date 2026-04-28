package ray_tracing

import (
	"fmt"
	"src-golang/model/camera"
	"src-golang/model/object"
	"sync"
	"sync/atomic"
	"time"
)

// TraceScene renders the object tree from the supplied camera into the film.
func (h *Handler) TraceScene(renderCamera camera.Camera, objectTree *object.ObjectTree, film *camera.Film, samples int64) {
	var (
		progress    int64
		totalPixels = len(film.Data[0].Data)
		wg          sync.WaitGroup
		taskChan    = make(chan int, totalPixels)
		done        = make(chan bool)
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

	for i := 0; i < len(film.Data[0].Data); i++ {
		taskChan <- i
	}
	close(taskChan)

	for i := 0; i < h.ThreadNum; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			for pixel := range taskChan {
				x := film.Data[0].GetCoordinates(pixel)
				color := h.TracePixel(renderCamera, objectTree, samples, x...)
				for ch := 0; ch < 3; ch++ {
					film.Data[ch].Data[pixel] = color.AtVec(ch)
				}

				atomic.AddInt64(&progress, 1)
			}
		}(int64(i))
	}

	wg.Wait()
	close(done)
	time.Sleep(100 * time.Millisecond)

	film.Samples = samples
}
