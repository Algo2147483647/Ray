package ray_tracing

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"sync"
	"sync/atomic"
)

// IntegratorKind is the serialized name of a light-transport algorithm.
type IntegratorKind string

const (
	IntegratorPathTracing  IntegratorKind = "path"
	IntegratorBDPT         IntegratorKind = "bdpt"
	IntegratorLightTracing IntegratorKind = "light_tracing"
)

// ParseIntegratorKind accepts canonical names and compatibility aliases at the
func ParseIntegratorKind(value string) (IntegratorKind, error) {
	switch value {
	case "", string(IntegratorPathTracing):
		return IntegratorPathTracing, nil
	case string(IntegratorBDPT):
		return IntegratorBDPT, nil
	case string(IntegratorLightTracing), "light_trace":
		return IntegratorLightTracing, nil
	default:
		return "", fmt.Errorf("unsupported integrator %q", value)
	}
}

// SceneIntegrator owns a work-distribution model. Algorithms are injected as
// kernels, so pixel-driven and splat-driven transports share one lifecycle
// without pretending to have the same scheduling semantics.
type SceneIntegrator interface {
	Run(*RenderContext) error
	EffectiveSampleCount(*RenderContext) int64
	ConcurrentFilmWrites() bool
}

// NewSceneIntegrator resolves configuration once, before rendering begins.
func NewSceneIntegrator(kind IntegratorKind, handler *Handler) (SceneIntegrator, error) {
	if handler == nil {
		return nil, fmt.Errorf("integrator handler is nil")
	}

	switch kind {
	case IntegratorPathTracing:
		return &pixelSceneIntegrator{kernel: pathTracingKernel{}}, nil
	case IntegratorBDPT:
		return &splatSceneIntegrator{kernel: &bdptKernel{}}, nil
	case IntegratorLightTracing:
		return &splatSceneIntegrator{kernel: &lightTracingKernel{}}, nil
	default:
		return nil, fmt.Errorf("unsupported integrator %q", kind)
	}
}

type pixelSceneIntegrator struct {
	kernel pixelKernel
}

func (d *pixelSceneIntegrator) ConcurrentFilmWrites() bool { return false }

func (d *pixelSceneIntegrator) EffectiveSampleCount(context *RenderContext) int64 {
	return context.Handler.EffectiveSampleCount(context.Samples)
}

func (d *pixelSceneIntegrator) Run(context *RenderContext) error {
	if d == nil || d.kernel == nil {
		return fmt.Errorf("pixel driver kernel is nil")
	}
	if context.Samples == 0 {
		return nil
	}

	tiles, totalPixels := buildTileCoordinatesForWindows(
		context.Camera.GetFilm().Shape,
		context.Camera.GetFilm().PixelWindows,
		context.Handler.BlockCols,
		context.Handler.BlockRows,
	)
	progress := newProgressReporter("Rendering", "pixels", totalPixels)
	defer progress.Close()

	workerCount := context.Handler.ThreadNum
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
				progress.Add(context.Handler.traceTile(d.kernel, context, tiles[index]))
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
	projection    camera.FilmProjection
}

type splatKernel interface {
	Prepare(*RenderContext) error
	WorkCount(*RenderContext) int64
	TraceSample(*RenderContext, int64) []FilmSplat
}

type splatSceneIntegrator struct {
	kernel splatKernel
}

const splatWorkBatchSize int64 = 64

func (d *splatSceneIntegrator) ConcurrentFilmWrites() bool { return true }

func (d *splatSceneIntegrator) EffectiveSampleCount(context *RenderContext) int64 {
	return context.Samples
}

func (d *splatSceneIntegrator) Run(context *RenderContext) error {
	if d == nil || d.kernel == nil {
		return fmt.Errorf("splat driver kernel is nil")
	}
	if err := d.kernel.Prepare(context); err != nil {
		return err
	}
	totalWork := d.kernel.WorkCount(context)
	if totalWork <= 0 {
		return nil
	}

	progress := newProgressReporter("Splat tracing", "paths", totalWork)
	defer progress.Close()
	workerCount := context.Handler.ThreadNum
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
				batchStart := nextWork.Add(splatWorkBatchSize) - splatWorkBatchSize
				if batchStart >= totalWork {
					return
				}
				batchEnd := min(batchStart+splatWorkBatchSize, totalWork)
				for workIndex := batchStart; workIndex < batchEnd; workIndex++ {
					for _, splat := range d.kernel.TraceSample(context, workIndex) {
						d.accumulate(context, splat, totalWork)
					}
				}
				progress.Add(batchEnd - batchStart)
			}
		}()
	}
	workers.Wait()
	return nil
}

func (d *splatSceneIntegrator) accumulate(context *RenderContext, splat FilmSplat, totalWork int64) {
	scale := 1 / float64(totalWork)
	value := optics.SpectralSampleRadiance(splat.Value.Sample(0), splat.WavelengthPDF) * scale
	context.Accumulator.AddSpectral(splat.Pixel, splat.WavelengthNM, value)
}
