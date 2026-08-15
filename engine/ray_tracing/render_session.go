package ray_tracing

import (
	"fmt"
	"math"
	"sync"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/optics"
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
	handler.prepareFilm(ctx.Film)
	return &RenderSession{
		Context:     ctx,
		Handler:     handler,
		Accumulator: newFilmAccumulator(ctx.Film, concurrentFilmWrites),
	}, nil
}

func (s *RenderSession) Finalize(samples int64) {
	if s == nil || s.Context.Film == nil {
		return
	}
	if s.Handler.usesSpectralRendering(s.Context.Film) {
		s.Context.Film.ConvertSpectralBinsToFilmColorSpace()
	}
	s.Context.Film.Samples = samples
}

// FilmAccumulator hides the distinct synchronization requirements of
// exclusive pixel writes and arbitrary cross-thread splats.
type FilmAccumulator interface {
	SetRGB(pixel int, color optics.Color3)
	AddRGB(pixel int, color optics.Color3)
	AddSpectral(pixel int, wavelengthNM, value float64)
}

type filmAccumulator struct {
	film  *camera.Film
	locks []sync.Mutex
}

func newFilmAccumulator(film *camera.Film, concurrent bool) FilmAccumulator {
	accumulator := &filmAccumulator{film: film}
	if concurrent && film != nil {
		accumulator.locks = make([]sync.Mutex, len(film.Data[0].Data))
	}
	return accumulator
}

func (a *filmAccumulator) SetRGB(pixel int, color optics.Color3) {
	if !a.validPixel(pixel) {
		return
	}
	a.withPixelLock(pixel, func() {
		for channel := range 3 {
			a.film.Data[channel].Data[pixel] = color[channel]
		}
	})
}

func (a *filmAccumulator) AddRGB(pixel int, color optics.Color3) {
	if !a.validPixel(pixel) {
		return
	}
	a.withPixelLock(pixel, func() {
		for channel := range 3 {
			a.film.Data[channel].Data[pixel] += color[channel]
		}
	})
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
	return a != nil && a.film != nil && pixel >= 0 && pixel < len(a.film.Data[0].Data)
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
