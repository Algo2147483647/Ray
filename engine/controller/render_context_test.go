package controller

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
)

func TestResolveRenderSpecOwnsRenderDefaults(t *testing.T) {
	resolved, err := ResolveRenderSpec(parser.RenderScript{CameraID: "main", SpectrumMode: "sampled"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Integrator != "path" || resolved.Samples != 20 || resolved.ThreadNum <= 0 {
		t.Fatalf("unexpected resolved defaults: %+v", resolved)
	}
	if resolved.WavelengthSamples != 4 {
		t.Fatalf("sampled wavelength default = %d, want 4", resolved.WavelengthSamples)
	}

	hero, err := ResolveRenderSpec(parser.RenderScript{CameraID: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if hero.SpectrumMode != "hero_wavelength" || hero.WavelengthSamples != 1 {
		t.Fatalf("unexpected hero defaults: %+v", hero)
	}
}

func TestResolveRenderSpecPreservesExplicitSampleCount(t *testing.T) {
	resolved, err := ResolveRenderSpec(parser.RenderScript{
		CameraID: "main", SpectrumMode: "sampled", WavelengthSamples: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.WavelengthSamples != 1 {
		t.Fatalf("explicit wavelength_samples changed to %d", resolved.WavelengthSamples)
	}
}
