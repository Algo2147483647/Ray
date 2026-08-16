package ray_tracing

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

func TestConcurrentFilmAccumulatorSerializesPixelSplats(t *testing.T) {
	film := camera.NewFilm(1, 1)
	film.InitSpectralBins(4, 400, 800)
	accumulator := newFilmAccumulator(film, true)

	const workers = 16
	const writesPerWorker = 100
	var group sync.WaitGroup
	group.Add(workers)
	for range workers {
		go func() {
			defer group.Done()
			for range writesPerWorker {
				accumulator.AddSpectral(0, 550, 0.5)
			}
		}()
	}
	group.Wait()

	writes := float64(workers * writesPerWorker)
	bin := film.SpectralBinIndex(550)
	if got, want := film.SpectralBins[bin].Data[0], 0.5*writes; got != want {
		t.Fatalf("spectral accumulation = %v, want %v", got, want)
	}
}

type constantSplatKernel struct {
	work int64
}

func (*constantSplatKernel) Prepare(*RenderSession) error { return nil }

func (k *constantSplatKernel) WorkCount(*RenderSession) int64 { return k.work }

func (*constantSplatKernel) TraceSample(*RenderSession, int64) []FilmSplat {
	return []FilmSplat{{
		Pixel:         0,
		WavelengthNM:  550,
		WavelengthPDF: optics.UniformWavelengthPDF(),
		Value:         optics.NewSampledSpectrum([]float64{2}),
	}}
}

func TestSplatDriverNormalizesByGlobalWorkCount(t *testing.T) {
	handler := NewHandler()
	handler.ThreadNum = 2
	handler.SpectrumMode = optics.SpectrumModeHeroWavelength
	film := camera.NewFilm(1, 1)
	film.InitSpectralBins(8, optics.WavelengthMin, optics.WavelengthMax)
	session, err := newRenderSession(handler, RenderContext{
		Camera: fixedCamera{Camera: camera.Camera{Film: film}}, ObjectTree: &object.ObjectTree{},
		Samples: 7,
	}, true)
	if err != nil {
		t.Fatalf("newRenderSession: %v", err)
	}
	driver := &splatDriver{kernel: &constantSplatKernel{work: 8}}
	if err := driver.Run(session); err != nil {
		t.Fatalf("splat Run: %v", err)
	}
	session.Finalize(driver.EffectiveSampleCount(session))

	bin := film.SpectralBinIndex(550)
	if got, want := film.SpectralBins[bin].Data[0], 2.0; math.Abs(got-want) > 1e-12 {
		t.Fatalf("spectral bin = %v, want %v", got, want)
	}
	if film.Samples != 7 {
		t.Fatalf("film samples = %d, want 7", film.Samples)
	}
}

type recordingSplatKernel struct {
	visits []atomic.Int32
}

func (*recordingSplatKernel) Prepare(*RenderSession) error { return nil }

func (k *recordingSplatKernel) WorkCount(*RenderSession) int64 { return int64(len(k.visits)) }

func (k *recordingSplatKernel) TraceSample(_ *RenderSession, index int64) []FilmSplat {
	k.visits[index].Add(1)
	return nil
}

func TestSplatDriverBatchAcquisitionVisitsTailExactlyOnce(t *testing.T) {
	const workCount = splatWorkBatchSize*3 + 17
	handler := NewHandler()
	handler.ThreadNum = 8
	film := camera.NewFilm(1, 1)
	session, err := newRenderSession(handler, RenderContext{
		Camera: fixedCamera{Camera: camera.Camera{Film: film}}, ObjectTree: &object.ObjectTree{},
		Samples: 1,
	}, true)
	if err != nil {
		t.Fatalf("newRenderSession: %v", err)
	}
	kernel := &recordingSplatKernel{visits: make([]atomic.Int32, workCount)}
	if err := (&splatDriver{kernel: kernel}).Run(session); err != nil {
		t.Fatalf("splat Run: %v", err)
	}
	for index := range kernel.visits {
		if got := kernel.visits[index].Load(); got != 1 {
			t.Fatalf("work item %d visited %d times, want 1", index, got)
		}
	}
}
