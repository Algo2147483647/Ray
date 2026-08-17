package ray_tracing

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
)

type BDPTFallbackPolicy string

const (
	BDPTFallbackError BDPTFallbackPolicy = ""
	BDPTFallbackPath  BDPTFallbackPolicy = "path"
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

	requested := h.IntegratorKind
	effective := requested
	h.LastRequestedIntegrator = requested
	h.LastEffectiveIntegrator = requested
	h.LastFallbackReason = ""
	if requested == IntegratorBDPT {
		if h.BDPTFallbackPolicy != BDPTFallbackError && h.BDPTFallbackPolicy != BDPTFallbackPath {
			return fmt.Errorf("unsupported BDPT fallback policy %q", h.BDPTFallbackPolicy)
		}
		if _, err := h.prepareBDPT(renderCamera, objectTree); err != nil {
			switch h.BDPTFallbackPolicy {
			case BDPTFallbackPath:
				effective = IntegratorPathTracing
				h.LastEffectiveIntegrator = effective
				h.LastFallbackReason = err.Error()
			case BDPTFallbackError:
				return err
			}
		}
	}

	integrator, err := NewSceneIntegrator(effective, h)
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
