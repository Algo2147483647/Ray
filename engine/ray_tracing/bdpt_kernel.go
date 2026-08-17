package ray_tracing

import (
	"fmt"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

// bdptKernel uses global work scheduling so the t=1 strategy may splat to a
// pixel other than the camera pixel selected for the remaining strategies.
type bdptPreparedState struct {
	scene        *bdptSceneState
	activeMask   []bool
	activePixels []int
	width        int
	height       int
	wavelengths  int64
	totalWork    int64
}

type bdptKernel struct {
	prepared *bdptPreparedState
}

func (k *bdptKernel) Prepare(context *RenderContext) error {
	if k == nil || context == nil {
		return fmt.Errorf("BDPT kernel or render context is nil")
	}
	scene, err := context.Handler.prepareBDPT(context.Camera, context.ObjectTree)
	if err != nil {
		return err
	}
	film := context.Camera.GetFilm()
	shape := film.Shape
	mask := make([]bool, shapeElementCount(shape))
	if len(film.PixelWindows) == 0 {
		for pixel := range mask {
			mask[pixel] = true
		}
	} else {
		mask, _ = buildPixelWindowMask(shape, film.PixelWindows)
	}
	activePixels := make([]int, 0, len(mask))
	for pixel, active := range mask {
		if active {
			activePixels = append(activePixels, pixel)
		}
	}

	state := &bdptPreparedState{
		scene:        scene,
		activeMask:   mask,
		activePixels: activePixels,
		width:        shape[0],
		height:       shape[1],
		wavelengths:  1,
	}
	if context.Handler.SpectrumMode == optics.SpectrumModeSampledWavelengths {
		state.wavelengths = int64(context.Handler.wavelengthSampleCount())
	}
	state.totalWork = context.Samples * int64(len(activePixels)) * state.wavelengths
	k.prepared = state
	return nil
}

func (k *bdptKernel) WorkCount(*RenderContext) int64 {
	if k == nil || k.prepared == nil {
		return 0
	}
	return k.prepared.totalWork
}

func (k *bdptKernel) TraceSample(context *RenderContext, workIndex int64) []FilmSplat {
	if k == nil || k.prepared == nil || len(k.prepared.activePixels) == 0 {
		return nil
	}
	activeCount := len(k.prepared.activePixels)
	pixel := k.prepared.activePixels[int(workIndex%int64(activeCount))]
	coords := context.Camera.GetFilm().SpectralBins[0].GetCoordinates(pixel)

	u := rand.Float64()
	if context.Handler.SpectrumMode == optics.SpectrumModeSampledWavelengths {
		stratum := (workIndex / int64(activeCount)) % k.prepared.wavelengths
		u = (float64(stratum) + u) / float64(k.prepared.wavelengths)
	}
	wavelength := context.Handler.wavelengthSampler().Sample(u)
	wavelengthNM, wavelengthPDF := wavelength.LambdaNM, wavelength.PDF
	local, remoteSplats := context.Handler.traceBidirectionalPrepared(
		k.prepared.scene,
		context.Camera,
		context.ObjectTree,
		wavelengthNM,
		wavelengthPDF,
		coords...,
	)

	splats := make([]FilmSplat, 0, 1+len(remoteSplats))
	if validSpectrum(local) {
		splats = append(splats, FilmSplat{
			Pixel: pixel, WavelengthNM: wavelengthNM, WavelengthPDF: wavelengthPDF,
			// Global work samples camera pixels uniformly. Multiplying the local
			// estimator by the active pixel count restores per-pixel spp after
			// splatSceneIntegrator divides by totalWork.
			Value: local.MulScalar(float64(activeCount)),
		})
	}

	for _, splat := range remoteSplats {
		splats = append(splats, filterBDPTSplat(
			splat, k.prepared.width, k.prepared.height, k.prepared.activeMask,
		)...)
	}
	return splats
}

func filterBDPTSplat(splat FilmSplat, width, height int, activeMask []bool) []FilmSplat {
	if width <= 0 || height <= 0 || len(activeMask) != width*height {
		return nil
	}
	if len(splat.projection.Position) < 2 {
		return nil
	}
	pixel, ok := camera.PixelIndex(
		splat.projection.Position[0], splat.projection.Position[1], width, height,
	)
	if !ok || !activeMask[pixel] {
		return nil
	}
	splat.Pixel = pixel
	return []FilmSplat{splat}
}
