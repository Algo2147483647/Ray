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
	Prepare(*RenderJob) (PreparedIntegratorState, error)
	Run(*RenderJob, PreparedIntegratorState) error
	EffectiveSampleCount(*RenderJob) int64
	ConcurrentFilmWrites() bool
}

type PreparedIntegratorState interface {
	preparedIntegratorState()
}

type pixelPreparedState struct{}

func (pixelPreparedState) preparedIntegratorState() {}

// NewSceneIntegrator resolves configuration once, before rendering begins.
func NewSceneIntegrator(kind IntegratorKind) (SceneIntegrator, error) {
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

func (d *pixelSceneIntegrator) EffectiveSampleCount(job *RenderJob) int64 {
	return job.samples * int64(job.wavelengthSamples)
}

func (d *pixelSceneIntegrator) Prepare(*RenderJob) (PreparedIntegratorState, error) {
	return pixelPreparedState{}, nil
}

func (d *pixelSceneIntegrator) Run(job *RenderJob, _ PreparedIntegratorState) error {
	if d == nil || d.kernel == nil {
		return fmt.Errorf("pixel driver kernel is nil")
	}
	if job.samples == 0 {
		return nil
	}

	tiles, totalPixels := buildTileCoordinatesForWindows(
		job.film.Shape,
		job.film.PixelWindows,
		job.handler.BlockCols,
		job.handler.BlockRows,
	)
	progress := newProgressReporter("Rendering", "pixels", totalPixels)
	defer progress.Close()

	workerCount := job.threadNum
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
				progress.Add(job.handler.traceTile(d.kernel, job, tiles[index]))
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
	Value         float64
	projection    camera.FilmProjection
}

type splatKernel interface {
	Prepare(*RenderJob) (PreparedIntegratorState, error)
	WorkCount(*RenderJob, PreparedIntegratorState) int64
	TraceSample(*RenderJob, PreparedIntegratorState, int64) []FilmSplat
}

type splatSceneIntegrator struct {
	kernel splatKernel
}

const splatWorkBatchSize int64 = 64

func (d *splatSceneIntegrator) ConcurrentFilmWrites() bool { return true }

func (d *splatSceneIntegrator) EffectiveSampleCount(job *RenderJob) int64 {
	return job.samples
}

func (d *splatSceneIntegrator) Prepare(job *RenderJob) (PreparedIntegratorState, error) {
	if d == nil || d.kernel == nil {
		return nil, fmt.Errorf("splat driver kernel is nil")
	}
	return d.kernel.Prepare(job)
}

func (d *splatSceneIntegrator) Run(job *RenderJob, prepared PreparedIntegratorState) error {
	if d == nil || d.kernel == nil {
		return fmt.Errorf("splat driver kernel is nil")
	}
	totalWork := d.kernel.WorkCount(job, prepared)
	if totalWork <= 0 {
		return nil
	}

	progress := newProgressReporter("Splat tracing", "paths", totalWork)
	defer progress.Close()
	workerCount := job.threadNum
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
					for _, splat := range d.kernel.TraceSample(job, prepared, workIndex) {
						d.accumulate(job, splat, totalWork)
					}
				}
				progress.Add(batchEnd - batchStart)
			}
		}()
	}
	workers.Wait()
	return nil
}

func (d *splatSceneIntegrator) accumulate(job *RenderJob, splat FilmSplat, totalWork int64) {
	scale := 1 / float64(totalWork)
	value := optics.SpectralSampleRadiance(splat.Value, splat.WavelengthPDF) * scale
	job.accumulator.AddSpectral(splat.Pixel, splat.WavelengthNM, value)
}
