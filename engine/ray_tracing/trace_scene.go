package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
)

const (
	defaultSpectralBinCount = 64
	defaultTileSize         = 8
)

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
	return integrator.Render(RenderContext{
		Camera: renderCamera, ObjectTree: objectTree,
		Samples: samples,
	})
}
