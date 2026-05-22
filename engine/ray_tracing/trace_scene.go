package ray_tracing

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"sync"
	"sync/atomic"
	"time"
)

const defaultSpectralBinCount = 64

// TraceScene renders the object tree from the supplied camera into the film.
func (h *Handler) TraceScene(renderCamera camera.Camera, objectTree *object.ObjectTree, film *camera.Film, samples int64) {
	if h.WorkingSpace == "" {
		h.WorkingSpace = film.ColorSpace
	}
	if film.ColorSpace == "" {
		film.ColorSpace = h.WorkingSpace
	}
	if h.SpectrumMode != optics.SpectrumModeRGB && !film.HasSpectralBins() {
		film.InitSpectralBins(defaultSpectralBinCount, optics.WavelengthMin, optics.WavelengthMax)
	}

	var (
		nextTile    int64
		progress    int64
		totalPixels = len(film.Data[0].Data)
		tiles       = buildRenderTiles(film.Data[0].Shape, h.tileWidth(), h.tileHeight())
		wg          sync.WaitGroup
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

	for i := 0; i < h.ThreadNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				tileIndex := int(atomic.AddInt64(&nextTile, 1) - 1)
				if tileIndex >= len(tiles) {
					return
				}

				tile := tiles[tileIndex]
				rendered := int64(0)
				for y := tile.Y0; y < tile.Y1; y++ {
					for x := tile.X0; x < tile.X1; x++ {
						pixel := tile.pixelIndex(x, y, film.Data[0].Shape)
						coords := film.Data[0].GetCoordinates(pixel)
						if h.SpectrumMode != optics.SpectrumModeRGB && film.HasSpectralBins() {
							for _, sample := range h.TracePixelSpectralSamples(renderCamera, objectTree, samples, coords...) {
								film.RecordSpectralSample(pixel, sample.WavelengthNM, sample.Value)
							}
						} else {
							color := h.TracePixel(renderCamera, objectTree, samples, coords...)
							for ch := 0; ch < 3; ch++ {
								film.Data[ch].Data[pixel] = color.AtVec(ch)
							}
						}
						rendered++
					}
				}

				atomic.AddInt64(&progress, rendered)
			}
		}()
	}

	wg.Wait()
	close(done)
	time.Sleep(100 * time.Millisecond)

	if h.SpectrumMode != optics.SpectrumModeRGB && film.HasSpectralBins() {
		film.ConvertSpectralBinsToWorkingSpace()
	}
	film.Samples = h.EffectiveSampleCount(samples)
}

type renderTile struct {
	X0 int
	X1 int
	Y0 int
	Y1 int
}

func (t renderTile) pixelIndex(x, y int, shape []int) int {
	if len(shape) == 2 {
		return y*shape[0] + x
	}
	return x
}

func buildRenderTiles(shape []int, tileWidth, tileHeight int) []renderTile {
	if tileWidth <= 0 {
		tileWidth = 8
	}
	if tileHeight <= 0 {
		tileHeight = 8
	}
	if len(shape) != 2 {
		total := 1
		for _, dim := range shape {
			total *= dim
		}
		chunkSize := tileWidth * tileHeight
		tiles := make([]renderTile, 0, (total+chunkSize-1)/chunkSize)
		for start := 0; start < total; start += chunkSize {
			end := start + chunkSize
			if end > total {
				end = total
			}
			tiles = append(tiles, renderTile{X0: start, X1: end, Y0: 0, Y1: 1})
		}
		return tiles
	}

	width := shape[0]
	height := shape[1]
	tiles := make([]renderTile, 0, ((width+tileWidth-1)/tileWidth)*((height+tileHeight-1)/tileHeight))
	for y := 0; y < height; y += tileHeight {
		y1 := y + tileHeight
		if y1 > height {
			y1 = height
		}
		for x := 0; x < width; x += tileWidth {
			x1 := x + tileWidth
			if x1 > width {
				x1 = width
			}
			tiles = append(tiles, renderTile{X0: x, X1: x1, Y0: y, Y1: y1})
		}
	}
	return tiles
}

func (h *Handler) tileWidth() int {
	if h.BlockCols <= 0 {
		return 8
	}
	return h.BlockCols
}

func (h *Handler) tileHeight() int {
	if h.BlockRows <= 0 {
		return 8
	}
	return h.BlockRows
}
