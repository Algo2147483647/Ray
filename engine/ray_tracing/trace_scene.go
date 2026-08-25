package ray_tracing

import (
	"fmt"
)

const defaultTileSize = 8

// TraceScene selects one scene-level integrator and delegates the complete
// render schedule to it.
func (h *Handler) TraceScene(job RenderJob) error {
	if h == nil {
		return fmt.Errorf("render handler is nil")
	}
	if job.camera == nil || job.film == nil {
		return fmt.Errorf("render camera or Film is nil")
	}
	if job.threadNum <= 0 {
		return fmt.Errorf("render thread_num must be resolved before tracing")
	}
	if job.wavelengthSamples <= 0 {
		return fmt.Errorf("render wavelength_samples must be resolved before tracing")
	}

	integrator, err := NewSceneIntegrator(job.integrator)
	if err != nil {
		return err
	}

	job.handler = h
	job.accumulator = newFilmAccumulator(job.film, integrator.ConcurrentFilmWrites())

	prepared, err := integrator.Prepare(&job)
	if err != nil {
		return err
	}
	err = integrator.Run(&job, prepared)
	if err != nil {
		return err
	}

	job.film.Samples = integrator.EffectiveSampleCount(&job)
	return nil
}
