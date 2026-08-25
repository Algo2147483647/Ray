package ray_tracing

import (
	"fmt"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/model/camera"
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

func (k *bdptKernel) Prepare(job *RenderJob) (PreparedIntegratorState, error) {
	if k == nil || job == nil {
		return nil, fmt.Errorf("BDPT kernel or render context is nil")
	}
	scene, err := job.handler.prepareBDPT(job.camera, job.film, job.objectTree)
	if err != nil {
		return nil, err
	}
	renderFilm := job.film
	shape := renderFilm.Shape
	mask := make([]bool, shapeElementCount(shape))
	if len(renderFilm.PixelWindows) == 0 {
		for pixel := range mask {
			mask[pixel] = true
		}
	} else {
		mask, _ = buildPixelWindowMask(shape, renderFilm.PixelWindows)
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
		wavelengths:  int64(job.wavelengthSamples),
	}
	state.totalWork = job.samples * int64(len(activePixels)) * state.wavelengths
	return state, nil
}

func (k *bdptKernel) WorkCount(_ *RenderJob, prepared PreparedIntegratorState) int64 {
	state, ok := prepared.(*bdptPreparedState)
	if k == nil || !ok || state == nil {
		return 0
	}
	return state.totalWork
}

func (k *bdptKernel) TraceSample(job *RenderJob, prepared PreparedIntegratorState, workIndex int64) []FilmSplat {
	state, ok := prepared.(*bdptPreparedState)
	if k == nil || !ok || state == nil || len(state.activePixels) == 0 {
		return nil
	}
	activeCount := len(state.activePixels)
	pixel := state.activePixels[int(workIndex%int64(activeCount))]
	coords := job.film.SpectralBins[0].GetCoordinates(pixel)

	stratum := (workIndex / int64(activeCount)) % state.wavelengths
	u := (float64(stratum) + rand.Float64()) / float64(state.wavelengths)
	wavelength := job.handler.wavelengthSampler().Sample(u)
	wavelengthNM, wavelengthPDF := wavelength.LambdaNM, wavelength.PDF
	local, remoteSplats := job.handler.traceBidirectionalPrepared(
		state.scene,
		job.camera,
		job.film.Shape,
		job.objectTree,
		wavelengthNM,
		wavelengthPDF,
		coords...,
	)

	splats := make([]FilmSplat, 0, 1+len(remoteSplats))
	if validPower(local) {
		splats = append(splats, FilmSplat{
			Pixel: pixel, WavelengthNM: wavelengthNM, WavelengthPDF: wavelengthPDF,
			// Global work samples camera pixels uniformly. Multiplying the local
			// estimator by the active pixel count restores per-pixel spp after
			// splatSceneIntegrator divides by totalWork.
			Value: local * float64(activeCount),
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
