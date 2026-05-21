package ray_tracing

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/model/optics"
)

func TestEffectiveSampleCountUsesWavelengthSubsamples(t *testing.T) {
	handler := NewHandler()
	handler.SpectrumMode = optics.SpectrumModeSampledWavelengths
	handler.WavelengthSamples = 3

	if got := handler.EffectiveSampleCount(10); got != 30 {
		t.Fatalf("unexpected effective sample count: got %d want 30", got)
	}
}

func TestEffectiveSampleCountDefaultsSampledModeWavelengths(t *testing.T) {
	handler := NewHandler()
	handler.SpectrumMode = optics.SpectrumModeSampledWavelengths
	handler.WavelengthSamples = 0

	if got := handler.EffectiveSampleCount(10); got != 40 {
		t.Fatalf("unexpected default sampled effective count: got %d want 40", got)
	}
}
