package ray_tracing

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
)

func TestEffectiveSampleCountUsesWavelengthSubsamples(t *testing.T) {
	job := &RenderJob{samples: 10, wavelengthSamples: 3}

	if got := (&pixelSceneIntegrator{}).EffectiveSampleCount(job); got != 30 {
		t.Fatalf("unexpected effective sample count: got %d want 30", got)
	}
}

func TestEffectiveSampleCountDoesNotInventWavelengthSubsamples(t *testing.T) {
	job := &RenderJob{samples: 10}

	if got := (&pixelSceneIntegrator{}).EffectiveSampleCount(job); got != 0 {
		t.Fatalf("runtime handler invented a wavelength default: got %d", got)
	}
}

func TestNewHandlerUsesRussianRouletteFriendlyDefaults(t *testing.T) {
	handler := NewHandler(geometry.DefaultSceneSpace())

	if handler.RussianRouletteDepth != 3 {
		t.Fatalf("expected russian roulette after 3 bounces, got %d", handler.RussianRouletteDepth)
	}
	if handler.MaxRayLevel < 32 {
		t.Fatalf("expected hard max ray level to be a safety cap, got %d", handler.MaxRayLevel)
	}
}
