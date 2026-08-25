package controller

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model/camera"
)

func testRenderSpec() parser.RenderScript {
	return parser.RenderScript{CameraID: "main", Film: &camera.FilmSpec{Shape: []int{4, 4}}, Output: "test.bin"}
}

func TestResolveRenderSpecOwnsRenderDefaults(t *testing.T) {
	spec := testRenderSpec()
	resolved, err := ResolveRenderSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Integrator != "path" || resolved.Samples != 20 || resolved.ThreadNum <= 0 {
		t.Fatalf("unexpected resolved defaults: %+v", resolved)
	}
	if resolved.WavelengthSamples != 1 {
		t.Fatalf("wavelength sample default = %d, want 1", resolved.WavelengthSamples)
	}
}

func TestResolveRenderSpecPreservesExplicitSampleCount(t *testing.T) {
	spec := testRenderSpec()
	spec.WavelengthSamples = 4
	resolved, err := ResolveRenderSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.WavelengthSamples != 4 {
		t.Fatalf("explicit wavelength_samples changed to %d", resolved.WavelengthSamples)
	}
}
