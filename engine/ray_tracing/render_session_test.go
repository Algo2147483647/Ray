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
				accumulator.AddRGB(0, optics.Color3{1, 2, 3})
				accumulator.AddSpectral(0, 550, 0.5)
			}
		}()
	}
	group.Wait()

	writes := float64(workers * writesPerWorker)
	if film.Data[0].Data[0] != writes || film.Data[1].Data[0] != 2*writes || film.Data[2].Data[0] != 3*writes {
		t.Fatalf("unexpected RGB accumulation: %v %v %v", film.Data[0].Data[0], film.Data[1].Data[0], film.Data[2].Data[0])
	}
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
	return []FilmSplat{{Pixel: 0, Value: optics.NewRGBSpectrum(2, 4, 6)}}
}

func TestSplatDriverNormalizesByGlobalWorkCount(t *testing.T) {
	handler := NewHandler()
	handler.ThreadNum = 2
	handler.SpectrumMode = optics.SpectrumModeRGB
	film := camera.NewFilm(1, 1)
	session, err := newRenderSession(handler, RenderContext{
		Camera: fixedCamera{}, ObjectTree: &object.ObjectTree{},
		Film: film, Samples: 7,
	}, true)
	if err != nil {
		t.Fatalf("newRenderSession: %v", err)
	}
	driver := &splatDriver{kernel: &constantSplatKernel{work: 8}}
	if err := driver.Run(session); err != nil {
		t.Fatalf("splat Run: %v", err)
	}
	session.Finalize(driver.EffectiveSampleCount(session))

	for channel, want := range []float64{2, 4, 6} {
		if got := film.Data[channel].Data[0]; math.Abs(got-want) > 1e-12 {
			t.Fatalf("channel %d = %v, want %v", channel, got, want)
		}
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
		Camera: fixedCamera{}, ObjectTree: &object.ObjectTree{},
		Film: film, Samples: 1,
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
