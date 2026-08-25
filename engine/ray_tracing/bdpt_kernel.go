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

func (*bdptPreparedState) preparedIntegratorState() {}

type bdptKernel struct{}

func (k *bdptKernel) Prepare(context *RenderContext) (PreparedIntegratorState, error) {
	if k == nil || context == nil {
		return nil, fmt.Errorf("BDPT kernel or render context is nil")
	}
	scene, err := context.Handler.prepareBDPT(context.Camera, context.ObjectTree)
	if err != nil {
		return nil, err
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
	return state, nil
}

func (k *bdptKernel) WorkCount(_ *RenderContext, prepared PreparedIntegratorState) int64 {
	state, ok := prepared.(*bdptPreparedState)
	if k == nil || !ok || state == nil {
		return 0
	}
	return state.totalWork
}

func (k *bdptKernel) TraceSample(context *RenderContext, prepared PreparedIntegratorState, workIndex int64) []FilmSplat {
	state, ok := prepared.(*bdptPreparedState)
	if k == nil || !ok || state == nil || len(state.activePixels) == 0 {
		return nil
	}
	activeCount := len(state.activePixels)
	pixel := state.activePixels[int(workIndex%int64(activeCount))]
	coords := context.Camera.GetFilm().SpectralBins[0].GetCoordinates(pixel)

	u := rand.Float64()
	if context.Handler.SpectrumMode == optics.SpectrumModeSampledWavelengths {
		stratum := (workIndex / int64(activeCount)) % state.wavelengths
		u = (float64(stratum) + u) / float64(state.wavelengths)
	}
	wavelength := context.Handler.wavelengthSampler().Sample(u)
	wavelengthNM, wavelengthPDF := wavelength.LambdaNM, wavelength.PDF
	local, remoteSplats := context.Handler.traceBidirectionalPrepared(
		state.scene,
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
			splat, state.width, state.height, state.activeMask,
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
