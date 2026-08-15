package ray_tracing

import (
	"fmt"
	"math/rand/v2"
	"os"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

// bdptKernel uses global work scheduling so the t=1 strategy may splat to a
// pixel other than the camera pixel selected for the remaining strategies.
type bdptPreparedState struct {
	scene        *bdptSceneState
	projective   camera.ProjectiveCamera
	activeMask   []bool
	activePixels []int
	wavelengths  int64
	totalWork    int64
}

type bdptKernel struct {
	prepared *bdptPreparedState
}

func (k *bdptKernel) Prepare(session *RenderSession) error {
	if k == nil || session == nil {
		return fmt.Errorf("BDPT kernel or render session is nil")
	}
	shape := session.Context.Film.Data[0].Shape
	mask := make([]bool, shapeElementCount(shape))
	if len(session.Context.PixelWindows) == 0 {
		for pixel := range mask {
			mask[pixel] = true
		}
	} else {
		mask, _ = buildPixelWindowMask(shape, session.Context.PixelWindows)
	}
	activePixels := make([]int, 0, len(mask))
	for pixel, active := range mask {
		if active {
			activePixels = append(activePixels, pixel)
		}
	}

	state := &bdptPreparedState{
		scene:        prepareBDPTScene(session.Handler.SceneGeometry, session.Context.ObjectTree),
		activeMask:   mask,
		activePixels: activePixels,
		wavelengths:  1,
	}
	if session.Handler.SpectrumMode == optics.SpectrumModeSampledWavelengths {
		state.wavelengths = int64(session.Handler.wavelengthSampleCount())
	}
	state.projective, _ = session.Context.Camera.(camera.ProjectiveCamera)
	state.totalWork = session.Context.Samples * int64(len(activePixels)) * state.wavelengths
	k.prepared = state
	if state.scene.FallbackReason != "" {
		fmt.Fprintf(os.Stderr, "BDPT fallback: requested=bdpt effective=path reason=%s\n", state.scene.FallbackReason)
	}
	return nil
}

func (k *bdptKernel) WorkCount(*RenderSession) int64 {
	if k == nil || k.prepared == nil {
		return 0
	}
	return k.prepared.totalWork
}

func (k *bdptKernel) TraceSample(session *RenderSession, workIndex int64) []FilmSplat {
	if k == nil || k.prepared == nil || len(k.prepared.activePixels) == 0 {
		return nil
	}
	activeCount := len(k.prepared.activePixels)
	pixel := k.prepared.activePixels[int(workIndex%int64(activeCount))]
	coords := session.Context.Film.Data[0].GetCoordinates(pixel)

	wavelengthNM := 0.0
	wavelengthPDF := 0.0
	if session.Handler.SpectrumMode != optics.SpectrumModeRGB {
		u := rand.Float64()
		if session.Handler.SpectrumMode == optics.SpectrumModeSampledWavelengths {
			stratum := (workIndex / int64(activeCount)) % k.prepared.wavelengths
			u = (float64(stratum) + u) / float64(k.prepared.wavelengths)
		}
		wavelength := session.Handler.wavelengthSampler().Sample(u)
		wavelengthNM = wavelength.LambdaNM
		wavelengthPDF = wavelength.PDF
	}
	local, lightPath := session.Handler.traceBidirectionalPrepared(
		k.prepared.scene,
		session.Context.Camera,
		session.Context.ObjectTree,
		wavelengthNM,
		wavelengthPDF,
		coords...,
	)

	splats := make([]FilmSplat, 0, 1+len(lightPath))
	if validSpectrum(local) {
		splats = append(splats, FilmSplat{
			Pixel: pixel, WavelengthNM: wavelengthNM, WavelengthPDF: wavelengthPDF,
			// Global work samples camera pixels uniformly. Multiplying the local
			// estimator by the active pixel count restores per-pixel spp after
			// splatDriver divides by totalWork.
			Value: local.MulScalar(float64(activeCount)),
		})
	}

	// Delta-caustic t=1 strategies form a disjoint path family from the
	// continuous strategies currently handled by BDPT MIS. This is the path
	// family needed by light -> specular chain -> diffuse screen -> camera.
	if k.prepared.projective == nil {
		return splats
	}
	deltaSplats := session.Handler.projectBDPTDeltaCaustics(
		k.prepared.projective,
		session.Context.ObjectTree,
		lightPath,
		wavelengthNM,
		wavelengthPDF,
	)
	for _, splat := range deltaSplats {
		if splat.Pixel >= 0 && splat.Pixel < len(k.prepared.activeMask) &&
			k.prepared.activeMask[splat.Pixel] {
			splats = append(splats, splat)
		}
	}
	return splats
}

func (h *Handler) projectBDPTDeltaCaustics(
	projective camera.ProjectiveCamera,
	tree *object.ObjectTree,
	lightPath []bdptVertex,
	wavelengthNM, wavelengthPDF float64,
) []FilmSplat {
	if projective == nil {
		return nil
	}
	result := make([]FilmSplat, 0, len(lightPath))
	seenDelta := false
	for vertexIndex := range lightPath {
		if vertexIndex > 0 && lightPath[vertexIndex-1].SampledDelta {
			seenDelta = true
		}
		if !seenDelta {
			continue
		}
		value, projection, ok := h.projectLightVertex(projective, tree, &lightPath[vertexIndex])
		if !ok {
			continue
		}
		result = append(result, FilmSplat{
			Pixel: projection.Pixel, WavelengthNM: wavelengthNM,
			WavelengthPDF: wavelengthPDF, Value: value,
		})
	}
	return result
}
