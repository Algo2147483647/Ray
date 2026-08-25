package ray_tracing

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

const defaultTileSize = 8

// TraceScene selects one scene-level integrator and delegates the complete
// render schedule to it.
func (h *Handler) TraceScene(
	target model.RenderTarget,
	objectTree *object.ObjectTree,
	samples int64,
) error {
	if h == nil {
		return fmt.Errorf("render handler is nil")
	}
	if target.Camera == nil || target.Film == nil {
		return fmt.Errorf("render camera or Film is nil")
	}
	if h.ThreadNum <= 0 {
		return fmt.Errorf("render thread_num must be resolved before tracing")
	}
	if h.SpectrumMode != optics.SpectrumModeRGB && h.WavelengthSamples <= 0 {
		return fmt.Errorf("render wavelength_samples must be resolved before tracing")
	}

	integrator, err := NewSceneIntegrator(h.IntegratorKind, h)
	if err != nil {
		return err
	}

	context := &RenderContext{
		Handler:     h,
		Target:      target,
		ObjectTree:  objectTree,
		Samples:     samples,
		Accumulator: newFilmAccumulator(target.Film, integrator.ConcurrentFilmWrites()),
	}

	prepared, err := integrator.Prepare(context)
	if err != nil {
		return err
	}
	err = integrator.Run(context, prepared)
	if err != nil {
		return err
	}

	context.Target.Film.Samples = integrator.EffectiveSampleCount(context)
	return nil
}
