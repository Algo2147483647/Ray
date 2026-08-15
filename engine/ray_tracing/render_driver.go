package ray_tracing

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

type pixelDriver struct {
	kernel pixelKernel
}

func (d *pixelDriver) ConcurrentFilmWrites() bool { return false }

func (d *pixelDriver) EffectiveSampleCount(session *RenderSession) int64 {
	return session.Handler.EffectiveSampleCount(session.Context.Samples)
}

func (d *pixelDriver) Run(session *RenderSession) error {
	if d == nil || d.kernel == nil {
		return fmt.Errorf("pixel driver kernel is nil")
	}
	if session.Context.Samples == 0 {
		return nil
	}

	shape := session.Context.Film.Data[0].Shape
	tiles, totalPixels := buildTileCoordinatesForWindows(
		shape,
		session.Context.PixelWindows,
		session.Handler.BlockCols,
		session.Handler.BlockRows,
	)
	progress := newProgressReporter("Rendering", "pixels", totalPixels)
	defer progress.Close()

	workerCount := session.Handler.ThreadNum
	if workerCount <= 0 {
		workerCount = 1
	}
	var nextTile atomic.Int64
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for {
				index := int(nextTile.Add(1) - 1)
				if index >= len(tiles) {
					return
				}
				progress.Add(session.Handler.traceTile(d.kernel, session, tiles[index]))
			}
		}()
	}
	workers.Wait()
	return nil
}

// FilmSplat is an unnormalized contribution produced by a global path sample.
type FilmSplat struct {
	Pixel         int
	WavelengthNM  float64
	WavelengthPDF float64
	Value         optics.Spectrum
}

type splatKernel interface {
	Prepare(*RenderSession) error
	WorkCount(*RenderSession) int64
	TraceSample(*RenderSession, int64) []FilmSplat
}

type splatDriver struct {
	kernel splatKernel
}

func (d *splatDriver) ConcurrentFilmWrites() bool { return true }

func (d *splatDriver) EffectiveSampleCount(session *RenderSession) int64 {
	return session.Context.Samples
}

func (d *splatDriver) Run(session *RenderSession) error {
	if d == nil || d.kernel == nil {
		return fmt.Errorf("splat driver kernel is nil")
	}
	if err := d.kernel.Prepare(session); err != nil {
		return err
	}
	totalWork := d.kernel.WorkCount(session)
	if totalWork <= 0 {
		return nil
	}

	progress := newProgressReporter("Splat tracing", "paths", totalWork)
	defer progress.Close()
	workerCount := session.Handler.ThreadNum
	if workerCount <= 0 {
		workerCount = 1
	}
	var nextWork atomic.Int64
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for {
				workIndex := nextWork.Add(1) - 1
				if workIndex >= totalWork {
					return
				}
				for _, splat := range d.kernel.TraceSample(session, workIndex) {
					d.accumulate(session, splat, totalWork)
				}
				progress.Add(1)
			}
		}()
	}
	workers.Wait()
	return nil
}

func (d *splatDriver) accumulate(session *RenderSession, splat FilmSplat, totalWork int64) {
	scale := 1 / float64(totalWork)
	if session.Handler.SpectrumMode == optics.SpectrumModeRGB {
		r, g, b := camera.LinearSRGBToFilmColorSpace(
			splat.Value.RGB[0], splat.Value.RGB[1], splat.Value.RGB[2],
			session.Handler.FilmColorSpace,
		)
		session.Accumulator.AddRGB(splat.Pixel, optics.Color3{r * scale, g * scale, b * scale})
		return
	}
	value := optics.SpectralSampleRadiance(splat.Value.Sample(0), splat.WavelengthPDF) * scale
	session.Accumulator.AddSpectral(splat.Pixel, splat.WavelengthNM, value)
}
