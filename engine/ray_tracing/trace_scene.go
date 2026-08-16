package ray_tracing

import (
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
	integrator, err := NewSceneIntegrator(h.IntegratorKind, h)
	if err != nil {
		return err
	}

	context := newRenderContext(h, RenderContext{
		Camera: renderCamera, ObjectTree: objectTree,
		Samples: samples,
	}, integrator.ConcurrentFilmWrites())
	err = integrator.Run(context)
	if err != nil {
		return err
	}

	context.Camera.GetFilm().Samples = integrator.EffectiveSampleCount(context)
	return nil
}
