package ray_tracing

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"math"
	"sync"

	"github.com/Algo2147483647/ray/engine/model/camera"
)

// RenderSession centralizes validation, film preparation, accumulation and
// finalization for every execution model.
type RenderSession struct {
	Context     RenderContext
	Handler     *Handler
	Accumulator FilmAccumulator
}

func newRenderSession(handler *Handler, ctx RenderContext, concurrentFilmWrites bool) (*RenderSession, error) {
	if handler == nil {
		return nil, fmt.Errorf("render handler is nil")
	}
	if err := ctx.validate(); err != nil {
		return nil, err
	}
	film := ctx.Camera.GetFilm()
	pixelWindows, err := camera.NormalizePixelWindows(film.PixelWindows, film.Shape)
	if err != nil {
		return nil, err
	}
	film.PixelWindows = pixelWindows
	handler.prepareFilm(film)
	return &RenderSession{
		Context:     ctx,
		Handler:     handler,
		Accumulator: newFilmAccumulator(film, concurrentFilmWrites),
	}, nil
}

func (h *Handler) prepareFilm(film *camera.Film) {
	if !film.HasSpectralBins() {
		film.InitSpectralBins(
			defaultSpectralBinCount,
			optics.WavelengthMin,
			optics.WavelengthMax,
		)
	}
}

func (s *RenderSession) Finalize(samples int64) {
	if s == nil || s.Context.Camera == nil || s.Context.Camera.GetFilm() == nil {
		return
	}
	s.Context.Camera.GetFilm().Samples = samples
}

// FilmAccumulator hides the distinct synchronization requirements of
// exclusive pixel writes and arbitrary cross-thread splats.
type FilmAccumulator interface {
	AddSpectral(pixel int, wavelengthNM, value float64)
}

type filmAccumulator struct {
	film  *camera.Film
	locks []sync.Mutex
}

func newFilmAccumulator(film *camera.Film, concurrent bool) FilmAccumulator {
	accumulator := &filmAccumulator{film: film}
	if concurrent && film != nil {
		accumulator.locks = make([]sync.Mutex, film.ElementCount())
	}
	return accumulator
}

func (a *filmAccumulator) AddSpectral(pixel int, wavelengthNM, value float64) {
	if !a.validPixel(pixel) || math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	a.withPixelLock(pixel, func() {
		a.film.RecordSpectralSample(pixel, wavelengthNM, value)
	})
}

func (a *filmAccumulator) validPixel(pixel int) bool {
	return a != nil && a.film != nil && pixel >= 0 && pixel < a.film.ElementCount()
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
