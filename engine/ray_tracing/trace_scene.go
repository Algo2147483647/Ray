package ray_tracing

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

const (
	defaultSpectralBinCount = 64
	defaultTileSize         = 8
	progressInterval        = 100 * time.Millisecond
)

// TraceScene renders the object tree from the supplied camera into the film.
func (h *Handler) TraceScene(
	renderCamera camera.Camera,
	objectTree *object.ObjectTree,
	film *camera.Film,
	samples int64,
) {
	h.prepareFilm(film)

	var (
		shape       = film.Data[0].Shape
		totalPixels = len(film.Data[0].Data)
		tiles       = buildTileCoordinates(shape, h.BlockCols, h.BlockRows)
		progress    int64
		done        = make(chan struct{})
		workerCount = h.ThreadNum
		nextTile    int64
		wg          sync.WaitGroup
	)

	if workerCount <= 0 {
		workerCount = 1
	}

	wg.Add(workerCount)

	go reportProgress(done, &progress, int64(totalPixels))

	for range workerCount {
		go func() {
			defer wg.Done()

			for {
				index := int(atomic.AddInt64(&nextTile, 1) - 1)
				if index >= len(tiles) {
					return
				}

				rendered := h.TraceTile(
					renderCamera,
					objectTree,
					film,
					samples,
					tiles[index],
				)

				atomic.AddInt64(&progress, rendered)
			}
		}()
	}

	wg.Wait()

	close(done)

	if h.usesSpectralRendering(film) {
		film.ConvertSpectralBinsToFilmColorSpace()
	}
	film.Samples = h.EffectiveSampleCount(samples)
}

func (h *Handler) prepareFilm(film *camera.Film) {
	if h.FilmColorSpace == "" {
		h.FilmColorSpace = film.ColorSpace
	}

	if film.ColorSpace == "" {
		film.ColorSpace = h.FilmColorSpace
	}

	if h.SpectrumMode != optics.SpectrumModeRGB && !film.HasSpectralBins() {
		film.InitSpectralBins(
			defaultSpectralBinCount,
			optics.WavelengthMin,
			optics.WavelengthMax,
		)
	}
}

func (h *Handler) usesSpectralRendering(film *camera.Film) bool {
	return h.SpectrumMode != optics.SpectrumModeRGB && film.HasSpectralBins()
}

func reportProgress(done <-chan struct{}, progress *int64, totalPixels int64) {
	start := time.Now()
	ticker := time.NewTicker(progressInterval)
	defer ticker.Stop()

	print := func(current int64) {
		percent := float64(current) / float64(totalPixels) * 100
		elapsed := time.Since(start).Round(time.Second)

		fmt.Printf(
			"\rRendering: %d/%d pixels (%.2f%%) Time: %v",
			current,
			totalPixels,
			percent,
			elapsed,
		)
	}

	for {
		select {
		case <-done:
			elapsed := time.Since(start).Round(time.Second)
			fmt.Printf(
				"\rRendering complete! Pixels: %d/%d (100%%) Time: %v\n",
				totalPixels,
				totalPixels,
				elapsed,
			)
			return

		case <-ticker.C:
			print(atomic.LoadInt64(progress))
		}
	}
}
