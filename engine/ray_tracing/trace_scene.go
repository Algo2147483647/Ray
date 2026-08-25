package ray_tracing

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
)

const defaultTileSize = 8

// TraceScene selects one scene-level integrator and delegates the complete
// render schedule to it.
func (h *Handler) TraceScene(
	renderCamera camera.RayCamera,
	objectTree *object.ObjectTree,
	samples int64,
) error {
	if h == nil {
		return fmt.Errorf("render handler is nil")
	}
	if renderCamera == nil || renderCamera.GetFilm() == nil {
		return fmt.Errorf("render camera or Film is nil")
	}

	if h.IntegratorKind == IntegratorBDPT {
		if _, err := h.prepareBDPT(renderCamera, objectTree); err != nil {
			return err
		}
	}

	integrator, err := NewSceneIntegrator(h.IntegratorKind, h)
	if err != nil {
		return err
	}

	context := &RenderContext{
		Handler:     h,
		Camera:      renderCamera,
		ObjectTree:  objectTree,
		Samples:     samples,
		Accumulator: newFilmAccumulator(renderCamera.GetFilm(), integrator.ConcurrentFilmWrites()),
	}

	err = integrator.Run(context)
	if err != nil {
		return err
	}

	context.Camera.GetFilm().Samples = integrator.EffectiveSampleCount(context)
	return nil
}
