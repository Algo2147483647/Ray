package ray_tracing

import (
	"fmt"
	"math"
	"sync"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/film"
	"github.com/Algo2147483647/ray/engine/model/object"
)

// RenderJob is the single resolved representation of one render. Its public
// construction validates all required values; runtime-only state is attached
// to a private copy by Handler.TraceScene.
type RenderJob struct {
	integrator        IntegratorKind
	camera            camera.RayCamera
	film              *film.Film
	output            string
	objectTree        *object.ObjectTree
	samples           int64
	threadNum         int
	wavelengthSamples int

	handler     *Handler
	accumulator *filmAccumulator
}

func NewRenderJob(
	integrator IntegratorKind,
	renderCamera camera.RayCamera,
	renderFilm *film.Film,
	output string,
	objectTree *object.ObjectTree,
	samples int64,
	threadNum int,
	wavelengthSamples int,
) (RenderJob, error) {
	canonicalIntegrator, err := ParseIntegratorKind(string(integrator))
	if err != nil {
		return RenderJob{}, err
	}
	if renderCamera == nil {
		return RenderJob{}, fmt.Errorf("render camera is nil")
	}
	if renderFilm == nil || renderFilm.ElementCount() == 0 || !renderFilm.HasSpectralBins() {
		return RenderJob{}, fmt.Errorf("render Film is not initialized")
	}
	if renderCamera.RasterDimension() != len(renderFilm.Shape) {
		return RenderJob{}, fmt.Errorf(
			"render camera expects a %dD Film, got %dD",
			renderCamera.RasterDimension(), len(renderFilm.Shape),
		)
	}
	if output == "" {
		return RenderJob{}, fmt.Errorf("render output is required")
	}
	if objectTree == nil {
		return RenderJob{}, fmt.Errorf("render object tree is nil")
	}
	if samples <= 0 {
		return RenderJob{}, fmt.Errorf("render samples must be positive")
	}
	if threadNum <= 0 {
		return RenderJob{}, fmt.Errorf("render thread_num must be positive")
	}
	if wavelengthSamples <= 0 {
		return RenderJob{}, fmt.Errorf("render wavelength_samples must be positive")
	}
	return RenderJob{
		integrator: canonicalIntegrator, camera: renderCamera, film: renderFilm,
		output: output, objectTree: objectTree, samples: samples,
		threadNum: threadNum, wavelengthSamples: wavelengthSamples,
	}, nil
}

func (j RenderJob) Integrator() IntegratorKind { return j.integrator }
func (j RenderJob) Camera() camera.RayCamera   { return j.camera }
func (j RenderJob) Film() *film.Film           { return j.film }
func (j RenderJob) Output() string             { return j.output }
func (j RenderJob) Samples() int64             { return j.samples }
func (j RenderJob) ThreadNum() int             { return j.threadNum }
func (j RenderJob) WavelengthSamples() int     { return j.wavelengthSamples }

type filmAccumulator struct {
	film  *film.Film
	locks []sync.Mutex
}

func newFilmAccumulator(film *film.Film, concurrent bool) *filmAccumulator {
	accumulator := &filmAccumulator{film: film}
	if concurrent && film != nil {
		accumulator.locks = make([]sync.Mutex, film.ElementCount())
	}
	return accumulator
}

func (a *filmAccumulator) AddSpectral(pixel int, wavelengthNM, value float64) {
	if a == nil || a.film == nil ||
		pixel < 0 || pixel >= a.film.ElementCount() ||
		math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}

	a.withPixelLock(pixel, func() {
		a.film.RecordSpectralSample(pixel, wavelengthNM, value)
	})
}

func (a *filmAccumulator) withPixelLock(pixel int, write func()) {
	if len(a.locks) == 0 {
		write()
		return
	}
	a.locks[pixel].Lock()
	write()
	a.locks[pixel].Unlock()
}
